package main

import "os"

// Configuration and constants for shielded TRC20 operations
var (
	Node             = "grpc://grpc.nile.trongrid.io:50051"
	PrivateKey       = os.Getenv("NILE_TEST_KEY1")
	TokenAddress     = "TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK" // SHL token on Nile
	ShieldedContract = "TV5mhPAhsK2rXKx1FAAgz58reKwW6zSTp2" // Nile Testnet shielded TRC20 contract
	ScalingFactor    = int64(1)                             // Scaling factor for this contract
	MintAmount       = "10000000"                           // 10 SHL tokens (6 decimals)
	BurnAmount       = "10000000"                           // 5 SHL tokens (6 decimals)
	BeginBlock       = int64(59808727)                      // where scan notes shall start from
	KeyFile          = "shielded_keys.json"                 // File to persist shielded keys
)

// Operation modes
const (
	ModeFullFlow = "full" // Run full flow including approval, mint, and burn
	ModeBurnOnly = "burn" // Skip approval/mint, only test burn with existing notes
	ModeTestOnly = "test" // Skip all broadcasting, test parameter generation only
)

// Current mode - set this to control what operations are performed
var CurrentMode = ModeBurnOnly
