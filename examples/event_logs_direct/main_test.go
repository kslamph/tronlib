package main

import (
	"testing"
)

func TestContractCache(t *testing.T) {
	// Test creating a new contract cache
	cache := NewContractCache()
	if cache == nil {
		t.Fatal("Failed to create contract cache")
	}

	// Test that cache is empty initially
	if len(cache.contracts) != 0 {
		t.Errorf("Expected empty cache, got %d contracts", len(cache.contracts))
	}

	// Test Get method on empty cache
	if contract := cache.Get("test"); contract != nil {
		t.Error("Expected nil contract from empty cache")
	}
}

func TestContractCacheThreadSafety(t *testing.T) {
	cache := NewContractCache()

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Set a contract
	go func() {
		cache.Set("test1", nil)
		done <- true
	}()

	// Goroutine 2: Get a contract
	go func() {
		cache.Get("test1")
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// If we get here without race conditions, the test passes
}
