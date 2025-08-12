package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kslamph/tronlib/pb/core"
	"github.com/kslamph/tronlib/pkg/utils"
	"golang.org/x/crypto/sha3"
)

// SavedParam and SavedEvent mirror the on-disk format used by event_abi_generator
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
	return true
}

func main() {
	var (
		inFile  string
		outFile string
	)
	flag.StringVar(&inFile, "in", "", "path to ABI JSON file (either an array of entries, or an object with an 'abi' field)")
	flag.StringVar(&outFile, "out", "tmp/events_registry.json", "output JSON file for collected events")
	flag.Parse()

	if strings.TrimSpace(inFile) == "" {
		fmt.Println("-in is required: provide a path to the ABI JSON file")
		os.Exit(2)
	}

	// Load or create persistent store
	store := NewPersistentStore(outFile)
	if err := store.Load(); err != nil {
		fmt.Printf("failed to load store: %v\n", err)
	}

	// Read ABI JSON file
	content, err := os.ReadFile(inFile)
	if err != nil {
		fmt.Printf("failed to read ABI file: %v\n", err)
		os.Exit(1)
	}

	// Accept either a raw ABI array or an object with an "abi" field
	var root any
	if err := json.Unmarshal(content, &root); err != nil {
		fmt.Printf("failed to parse JSON: %v\n", err)
		os.Exit(1)
	}

	var abiJSON []byte
	switch v := root.(type) {
	case []any:
		// Raw ABI array
		abiJSON, err = json.Marshal(v)
		if err != nil {
			fmt.Printf("failed to normalize ABI array: %v\n", err)
			os.Exit(1)
		}
	case map[string]any:
		if inner, ok := v["abi"]; ok {
			if arr, ok2 := inner.([]any); ok2 {
				abiJSON, err = json.Marshal(arr)
				if err != nil {
					fmt.Printf("failed to normalize inner abi array: %v\n", err)
					os.Exit(1)
				}
				break
			}
		}
		// Not an expected format
		fmt.Println("input JSON must be an ABI array or an object with an 'abi' array field")
		os.Exit(2)
	default:
		fmt.Println("input JSON must be an ABI array or an object with an 'abi' array field")
		os.Exit(2)
	}

	// Parse ABI into core.SmartContract_ABI
	processor := utils.NewABIProcessor(nil)
	contractABI, err := processor.ParseABI(string(abiJSON))
	if err != nil {
		fmt.Printf("failed to parse ABI: %v\n", err)
		os.Exit(1)
	}

	// Extract events and upsert into store
	added := 0
	for _, e := range contractABI.GetEntrys() {
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
			added++
		}
	}

	if err := store.Save(); err != nil {
		fmt.Printf("error saving store: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("processed ABI: %d new event(s) added to %s\n", added, outFile)
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

