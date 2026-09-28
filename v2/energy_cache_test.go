package tronlib

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kslamph/tronlib/pb/api"
	"github.com/kslamph/tronlib/v2/tron"
	"github.com/kslamph/tronlib/v2/tx"
)

func TestEnergyPriceCachesWithinTTL(t *testing.T) {
	f := &fakeFacadeServer{}
	c := newFacadeTestClient(t, f)
	for i := 0; i < 2; i++ {
		if _, err := c.EnergyPrice(context.Background()); err != nil {
			t.Fatalf("EnergyPrice: %v", err)
		}
	}
	if n := f.energyPricesCalls.Load(); n != 1 {
		t.Errorf("GetEnergyPrices called %d times within the TTL, want 1", n)
	}
}

func TestEnergyPriceRefetchesAfterTTL(t *testing.T) {
	f := &fakeFacadeServer{}
	c := newFacadeTestClient(t, f)
	if _, err := c.EnergyPrice(context.Background()); err != nil {
		t.Fatalf("EnergyPrice: %v", err)
	}
	c.priceMu.Lock()
	c.priceAt = c.priceAt.Add(-(tx.MaintenancePeriod + time.Second))
	c.priceMu.Unlock()
	if _, err := c.EnergyPrice(context.Background()); err != nil {
		t.Fatalf("EnergyPrice after TTL: %v", err)
	}
	if n := f.energyPricesCalls.Load(); n != 2 {
		t.Errorf("GetEnergyPrices called %d times, want 2 (one refetch past the TTL)", n)
	}
}

func TestEnergyPriceFailedRefetchDoesNotPoisonCache(t *testing.T) {
	f := &fakeFacadeServer{}
	var failing atomic.Bool
	failing.Store(true)
	f.EnergyPrices = func(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
		if failing.Load() {
			return nil, &tron.Error{Code: tron.CodeRPCMethodFailed, Op: "fake.GetEnergyPrices"}
		}
		return &api.PricesResponseMessage{Prices: "1691500000000:420"}, nil
	}
	c := newFacadeTestClient(t, f)

	if _, err := c.EnergyPrice(context.Background()); err == nil {
		t.Fatal("EnergyPrice with a failing node = nil, want the error")
	}
	if c.price != nil {
		t.Fatal("a failed fetch stored a price; the next call would serve a zero read")
	}

	failing.Store(false)
	p, err := c.EnergyPrice(context.Background())
	if err != nil {
		t.Fatalf("EnergyPrice after the node recovers: %v", err)
	}
	if p == nil || p.SunPerEnergy != 420 {
		t.Fatalf("EnergyPrice = %+v, want SunPerEnergy 420", p)
	}
}

func TestEnergyPriceConcurrentSingleFetch(t *testing.T) {
	f := &fakeFacadeServer{}
	c := newFacadeTestClient(t, f)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.EnergyPrice(context.Background())
		}()
	}
	wg.Wait()
	if n := f.energyPricesCalls.Load(); n != 1 {
		t.Errorf("GetEnergyPrices called %d times under 16 concurrent callers, want 1", n)
	}
}
