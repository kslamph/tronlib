package main

import (
	"testing"
)

func BenchmarkNewAccount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newAccount()
	}
}
func BenchmarkNewAccountOptimized(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newAccountOptimized()
	}
}

func BenchmarkIsMaxChar(b *testing.B) {
	// Example strings to test against
	testStrings := []string{
		"TAtrLqrN8SiWLXSfFUUDAdDmTEm5K9Max9",
		"TXBwCB1RxvMPZTZE79aJn9KjLbdSXMax55",
		"TNoZEANFVPPaghtCP7Nkv6gBYGXimax666",
		"TRv7JdQNybFd7FEgJ75NvWUo822gRMax66",
		"TYBBjvywmeV2pL5YLcsePKVpF6ezYmax88",
		"TUKWmcJHwGboDRFgE6uj5PemKB5rMax88g",
		"TRYTnwDTnJWxVqpWEDQXXRdc75GiBMax33",
		"TRYTnwDTnJWxVqpWEDQXXRdc75GiMax335",
		"TRYTnwDTnJWxVqpWEDQXXRdc75GiMax333",
		"TKoxkzTAGQa1nk6X571xP4fnPhTmPmax77",
		"TNTi468NpecRzb3s9FzPcjLXKcEq8Max44",
		"TVpbt9sX9y18HM1b7YZPdF7tL2DuKmax77",
		"TWNrdi3PwY16DV8xTW7e7UHZWRcPcMax77",
		"TCkkuJ3pugdcpTXLACr5tfxBPyVoEMax66",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			IsMaxChar(s)
		}
	}
}

func BenchmarkIsMaxCharOptimized(b *testing.B) {
	testStrings := []string{
		"TAtrLqrN8SiWLXSfFUUDAdDmTEm5K9Max9",
		"TXBwCB1RxvMPZTZE79aJn9KjLbdSXMax55",
		"TNoZEANFVPPaghtCP7Nkv6gBYGXimax666",
		"TRv7JdQNybFd7FEgJ75NvWUo822gRMax66",
		"TYBBjvywmeV2pL5YLcsePKVpF6ezYmax88",
		"TUKWmcJHwGboDRFgE6uj5PemKB5rMax88g",
		"TRYTnwDTnJWxVqpWEDQXXRdc75GiBMax33",
		"TRYTnwDTnJWxVqpWEDQXXRdc75GiMax335",
		"TRYTnwDTnJWxVqpWEDQXXRdc75GiMax333",
		"TKoxkzTAGQa1nk6X571xP4fnPhTmPmax77",
		"TNTi468NpecRzb3s9FzPcjLXKcEq8Max44",
		"TVpbt9sX9y18HM1b7YZPdF7tL2DuKmax77",
		"TWNrdi3PwY16DV8xTW7e7UHZWRcPcMax77",
		"TCkkuJ3pugdcpTXLACr5tfxBPyVoEMax66",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			IsMaxCharOptimized(s)
		}
	}
}
