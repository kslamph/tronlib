module github.com/kslamph/tronlib/v2

go 1.25.0

require (
	github.com/ethereum/go-ethereum v1.16.5
	github.com/kslamph/bip39-hdwallet v1.1.0
	github.com/kslamph/tronlib v0.0.0
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.55.0
	golang.org/x/tools v0.49.0
	google.golang.org/grpc v1.76.0
)

// The v2 module lives inside the tronlib repo and consumes its protobuf
// types (core.SmartContract_ABI) from the root module via a local replace.
replace github.com/kslamph/tronlib => ../

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.6 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
