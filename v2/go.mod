module github.com/kslamph/tronlib/v2

go 1.27.1

require (
	github.com/ethereum/go-ethereum v1.17.0
	github.com/kslamph/bip39-hdwallet v1.1.0
	github.com/kslamph/tronlib v1.3.0
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.57.0
	golang.org/x/tools v0.49.0
	google.golang.org/grpc v1.83.1
	google.golang.org/protobuf v1.36.11
)

// The v2 module consumes the root module's protobuf types
// (core.SmartContract_ABI, api.PaginatedMessage) from the PUBLISHED v1
// module (v1.3.0+ carries the GreatVoyage-v4.8.2 regenerated pb). No local
// replace: local builds resolve v1.3.0 from the module proxy exactly like
// downstream consumers do (P11 resolved).

require (
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20251001021608-1fe7b43fc4d6 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.6 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
