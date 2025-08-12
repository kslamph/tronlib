package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/network"
	"github.com/kslamph/tronlib/pkg/smartcontract"
	"github.com/kslamph/tronlib/pkg/types"
	"golang.org/x/crypto/sha3"
)

// SavedParam and SavedEvent define the persisted structure on disk.
type SavedParam struct {
	Type    string `json:"type"`
	Indexed bool   `json:"indexed"`
	Name    string `json:"name"`
}

type SavedEvent struct {
	// 4-byte selector in hex (lowercase, no 0x)
	Selector  string       `json:"selector"`
	Signature string       `json:"signature"` // e.g., Transfer(address,address,uint256)
	Name      string       `json:"name"`
	Inputs    []SavedParam `json:"inputs"`
}

type PersistentStore struct {
	mu      sync.Mutex
	by4Byte map[string]SavedEvent // selector hex -> event
	path    string
}

func NewPersistentStore(path string) *PersistentStore {
	return &PersistentStore{by4Byte: make(map[string]SavedEvent), path: path}
}

func (ps *PersistentStore) Load() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	f, err := os.Open(ps.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	// The file can be either an array or an object; we accept array of events
	var arr []SavedEvent
	if err := dec.Decode(&arr); err != nil && err != io.EOF {
		return err
	}
	for _, ev := range arr {
		ps.by4Byte[strings.ToLower(ev.Selector)] = ev
	}
	return nil
}

func (ps *PersistentStore) Save() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(ps.path), 0o755); err != nil {
		return err
	}

	// Write to temp then rename for atomicity
	tmp := ps.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	events := make([]SavedEvent, 0, len(ps.by4Byte))
	for _, ev := range ps.by4Byte {
		events = append(events, ev)
	}
	if err := enc.Encode(events); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, ps.path)
}

// Upsert inserts only if absent; returns true if inserted, false if exists
func (ps *PersistentStore) Upsert(ev SavedEvent) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	sel := strings.ToLower(ev.Selector)
	if _, exists := ps.by4Byte[sel]; exists {
		return false
	}
	ps.by4Byte[sel] = ev
	fmt.Printf("upserted %s\n", sel)
	return true
}

func main() {
	var (
		nodeAddr   string
		outFile    string
		startBlock int64
		sleepMs    int
	)
	flag.StringVar(&nodeAddr, "node", "grpc://127.0.0.1:50051", "TRON node URL (grpc://host:port or grpcs://host:port)")
	flag.StringVar(&outFile, "out", "tmp/events_registry.json", "output JSON file for collected events")
	flag.Int64Var(&startBlock, "start", -1, "start from this block number (default: latest)")
	flag.IntVar(&sleepMs, "sleep", 250, "sleep milliseconds between blocks to reduce load")
	flag.Parse()

	cli, err := client.NewClient(nodeAddr)
	if err != nil {
		fmt.Printf("failed to create client: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling for graceful stop
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nreceived signal, stopping...")
		cancel()
	}()

	// Managers
	netMgr := network.NewManager(cli)
	scMgr := smartcontract.NewManager(cli)

	// Persistent store
	store := NewPersistentStore(outFile)
	if err := store.Load(); err != nil {
		fmt.Printf("failed to load store: %v\n", err)
	}

	// Determine starting block
	var current int64
	if startBlock >= 0 {
		current = startBlock
	} else {
		nb, err := netMgr.GetNowBlock(ctx)
		if err != nil {
			fmt.Printf("failed to get now block: %v\n", err)
			os.Exit(1)
		}
		current = blockNumber(nb)
	}
	fmt.Printf("starting from block %d\n", current)

	// Caches
	seenContracts := make(map[string]struct{}) // address hex -> exists

	// Main loop: walk backwards until cancelled
	for {
		select {
		case <-ctx.Done():
			if err := store.Save(); err != nil {
				fmt.Printf("error saving store: %v\n", err)
			}
			return
		default:
		}

		if current < 0 {
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
			continue
		}

		// Fetch transaction infos for block
		tiList, err := cli.GetTransactionInfoByBlockNum(ctx, &api.NumberMessage{Num: current})
		if err != nil {
			fmt.Printf("block %d: failed to get tx infos: %v\n", current, err)
			// Step back anyway with a small pause
			current--
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
			continue
		}

		// Process logs
		for _, ti := range tiList.GetTransactionInfo() {
			for _, lg := range ti.GetLog() {
				addrBytes := lg.GetAddress()
				if len(addrBytes) == 0 {
					continue
				}
				addr, err := types.NewAddressFromBytes(addrBytes)
				if err != nil {
					continue
				}
				addrHex := strings.ToLower(hex.EncodeToString(addr.Bytes()))
				if _, ok := seenContracts[addrHex]; !ok {
					// Fetch ABI once per contract
					sc, err := scMgr.GetContract(ctx, addr)
					if err != nil {
						// Could be non-contract or missing ABI; skip
						continue
					}
					abi := sc.GetAbi()
					if abi != nil {
						// Extract events and upsert into store
						for _, e := range abi.GetEntrys() {
							if e == nil || e.GetType() != core.SmartContract_ABI_Entry_Event {
								continue
							}
							sig := buildEventSignature(e)
							selector := compute4Byte(sig)
							sev := SavedEvent{
								Selector:  selector,
								Signature: sig,
								Name:      e.GetName(),
								Inputs:    make([]SavedParam, len(e.GetInputs())),
							}
							for i, in := range e.GetInputs() {
								if in == nil {
									continue
								}
								sev.Inputs[i] = SavedParam{Type: in.GetType(), Indexed: in.GetIndexed(), Name: in.GetName()}
							}
							if inserted := store.Upsert(sev); inserted {
								// Save only on first insert for this selector
								if err := store.Save(); err != nil {
									fmt.Printf("error saving store: %v\n", err)
								}
							}
						}
					}
					seenContracts[addrHex] = struct{}{}
				}
			}
		}

		// Move to previous block
		current--
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
	}
}

func blockNumber(b *api.BlockExtention) int64 {
	if b == nil || b.GetBlockHeader() == nil || b.GetBlockHeader().GetRawData() == nil {
		return -1
	}
	return b.GetBlockHeader().GetRawData().GetNumber()
}

func buildEventSignature(e *core.SmartContract_ABI_Entry) string {
	inputs := make([]string, 0, len(e.GetInputs()))
	for _, in := range e.GetInputs() {
		if in == nil {
			continue
		}
		inputs = append(inputs, in.GetType())
	}
	return fmt.Sprintf("%s(%s)", e.GetName(), strings.Join(inputs, ","))
}

func compute4Byte(signature string) string {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(signature))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:4])
}
