package eventtool

import (
	"context"
	"fmt"
	"sync"

	"github.com/kslamph/tronlib/v2/pb/core"
	"github.com/kslamph/tronlib/v2/tron"
)

// ABIFetcher resolves one contract's on-chain ABI. The real implementation
// wraps the node's GetContract; tests supply a stub.
type ABIFetcher interface {
	ABI(ctx context.Context, addr tron.Address) (*core.SmartContract_ABI, error)
}

// CaptureReport is the outcome of one capture run.
type CaptureReport struct {
	Contracts  int // addresses in the snapshot
	WithEvents int // addresses whose ABI carried at least one usable event
	NewEvents  int // corpus entries actually inserted
	Skipped    int // addresses whose ABI carried no usable event
}

// Capture fetches every snapshotted contract's ABI and upserts its named,
// non-anonymous events into the store. Selection only gates which contracts are
// read; once selected, the contract's whole event set is ingested (D5).
//
// concurrency <= 1 fetches sequentially; a larger value widens the fetch but
// upserts happen in rank order afterwards, so the result is identical either
// way. A fetch error is returned, never silently skipped: the corpus is
// supposed to mirror the snapshot exactly.
func Capture(ctx context.Context, snap *Snapshot, f ABIFetcher, s *Store, concurrency int) (CaptureReport, error) {
	rep := CaptureReport{Contracts: len(snap.Contracts)}
	if concurrency < 1 {
		concurrency = 1
	}
	abis := make([]*core.SmartContract_ABI, len(snap.Contracts))
	errs := make([]error, len(snap.Contracts))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := range snap.Contracts {
		addr, err := tron.ParseAddress(snap.Contracts[i].Address)
		if err != nil {
			errs[i] = fmt.Errorf("snapshot rank %d: invalid address %q: %w", i+1, snap.Contracts[i].Address, err)
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, addr tron.Address) {
			defer wg.Done()
			defer func() { <-sem }()
			abis[i], errs[i] = f.ABI(ctx, addr)
		}(i, addr)
	}
	wg.Wait()

	// Apply in rank order: deterministic regardless of fetch completion order.
	for i := range snap.Contracts {
		if errs[i] != nil {
			return rep, fmt.Errorf("eventtool: contract %s: %w", snap.Contracts[i].Address, errs[i])
		}
		events := eventsOf(abis[i])
		if len(events) == 0 {
			rep.Skipped++
			continue
		}
		rep.WithEvents++
		for _, e := range events {
			if s.Upsert(e) {
				rep.NewEvents++
			}
		}
	}
	return rep, nil
}

// eventsOf extracts the usable event definitions from an ABI: named,
// non-anonymous events only. Anonymous events have no topic0 and cannot be
// signature-decoded; functions/constructors are not events.
func eventsOf(abi *core.SmartContract_ABI) []SavedEvent {
	if abi == nil {
		return nil
	}
	var out []SavedEvent
	for _, e := range abi.GetEntrys() {
		if e == nil || e.GetType() != core.SmartContract_ABI_Entry_Event || e.GetAnonymous() || e.GetName() == "" {
			continue
		}
		inputs := make([]SavedInput, 0, len(e.GetInputs()))
		for _, p := range e.GetInputs() {
			if p == nil {
				continue
			}
			inputs = append(inputs, SavedInput{Type: p.GetType(), Indexed: p.GetIndexed(), Name: p.GetName()})
		}
		out = append(out, makeSavedEvent(e.GetName(), inputs))
	}
	return out
}
