// Command shielded is a minimal, end-to-end walkthrough of TRON Shielded
// TRC-20 using only the low-level gRPC wrappers in pkg/client/lowlevel.
//
// It implements exactly five operations, one per -mode:
//
//	walletgen  derive the full sapling key hierarchy and the ztron payment address
//	mint       transparent TRC-20 -> shielded note      (t -> s)
//	scan       find owned notes by IVK or OVK
//	transfer   shielded note -> another ztron address   (s -> s)
//	burn       shielded note -> transparent TRC-20      (s -> t)
//
// Read docs/shielded.md first: it explains why every step exists, how the
// transaction is assembled, and what the trust model costs you.
//
// Usage:
//
//	go run ./example/shielded -mode=walletgen
//	go run ./example/shielded -mode=mint     -amount=10000000
//	go run ./example/shielded -mode=scan     -by=ivk -begin=70501000
//	go run ./example/shielded -mode=transfer -amount=4000000 -to=ztron1...
//	go run ./example/shielded -mode=burn     -amount=4000000
package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kslamph/tronlib/pkg/client"
	"github.com/kslamph/tronlib/pkg/signer"
	"github.com/kslamph/tronlib/pkg/types"
)

// config holds every knob the example needs, populated from flags.
type config struct {
	mode       string
	node       string
	privateKey string

	// token is the TRC-20 contract being shielded; shieldedContract is the
	// ShieldedTRC20 contract bound to it at deployment time.
	token            string
	shieldedContract string

	keyFile    string
	showKeys   bool
	amount     string
	to         string
	force      bool
	begin      int64
	end        int64
	scanBy     string
	checkSpent bool

	feeLimit int64
	timeout  time.Duration
}

func main() {
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() *config {
	c := &config{}

	flag.StringVar(&c.mode, "mode", "", "operation to run: walletgen | mint | scan | transfer | burn")
	flag.StringVar(&c.node, "node", envOr("TRON_NODE", "grpc://grpc.nile.trongrid.io:50051"),
		"full node endpoint (grpc:// or grpcs://)")
	flag.StringVar(&c.privateKey, "private-key", os.Getenv("NILE_TEST_KEY1"),
		"transparent account key; pays fees and owns the TRC-20 balance")
	flag.StringVar(&c.token, "token", envOr("SHIELDED_TOKEN", "TWRvzd6FQcsyp7hwCtttjZGpU1kfvVEtNK"),
		"TRC-20 contract address")
	flag.StringVar(&c.shieldedContract, "contract", envOr("SHIELDED_CONTRACT", "TV5mhPAhsK2rXKx1FAAgz58reKwW6zSTp2"),
		"ShieldedTRC20 contract address")
	// The default lives under tmp/ (gitignored) on purpose: a key file must
	// never land on a path the repository tracks, or the next walletgen writes
	// somebody's secrets straight into a commit.
	flag.StringVar(&c.keyFile, "keyfile", envOr("SHIELDED_KEYFILE", "tmp/shielded_keys.json"),
		"where shielded keys are persisted, relative to the directory you run from")
	flag.BoolVar(&c.showKeys, "show-keys", false,
		"walletgen only: print full key material instead of 4-byte fingerprints")

	flag.StringVar(&c.amount, "amount", "",
		"raw token amount in base units, i.e. exactly from_amount / to_amount (no decimal conversion)")
	flag.StringVar(&c.to, "to", "",
		"destination ztron payment address (transfer only)")

	flag.Int64Var(&c.begin, "begin", 0, "first block to scan (default: value stored in the key file)")
	flag.Int64Var(&c.end, "end", 0, "last block to scan (default: current head)")
	flag.StringVar(&c.scanBy, "by", "ivk", "scan with ivk (notes you received) or ovk (notes you sent)")
	flag.BoolVar(&c.force, "force", false, "walletgen only: replace an existing key file")
	flag.BoolVar(&c.checkSpent, "check-spent", false,
		"scan only: re-derive each note's nullifier and confirm its spend status on chain")

	flag.Int64Var(&c.feeLimit, "fee-limit", 350_000_000, "energy fee limit in sun (350 TRX)")
	flag.DurationVar(&c.timeout, "timeout", 5*time.Minute,
		"overall deadline; also the receipt-wait window, so a slow block means a false negative")
	flag.Parse()

	if c.mode == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\nerror: -mode is required (walletgen | mint | scan | transfer | burn)")
		os.Exit(2)
	}
	return c
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// session is the state every flow shares: the connection, the transparent
// account that pays fees, and the parsed shielded contract address.
//
// The context is deliberately not part of it (CODING_STANDARDS.md §2): it is
// passed to each call so cancellation stays visible at the call site.
type session struct {
	cli      *client.Client
	cfg      *config
	acct     *signer.PrivateKeySigner // nil for walletgen and scan
	contract *types.Address
}

// flows maps -mode to its handler. A map keeps main.go's dispatch flat; the
// alternative was a switch with five near-identical argument-parsing arms.
var flows = map[string]func(context.Context, *session) error{
	"walletgen": func(ctx context.Context, s *session) error { return runWalletGen(ctx, s) },
	"mint":      func(ctx context.Context, s *session) error { return runMint(ctx, s) },
	"scan":      func(ctx context.Context, s *session) error { return runScan(ctx, s) },
	"transfer":  func(ctx context.Context, s *session) error { return runTransfer(ctx, s) },
	"burn":      func(ctx context.Context, s *session) error { return runBurn(ctx, s) },
}

// modeNames returns the accepted -mode values in a stable order.
func modeNames() []string {
	names := make([]string, 0, len(flows))
	for name := range flows {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// run opens the shared state, then dispatches to the handler for -mode.
//
// Note there is no client-side cryptography here on purpose: key derivation,
// note commitments, nullifiers and zk-proofs are all produced by the node.
// See the trust warning in docs/shielded.md.
func run(c *config) error {
	handler, ok := flows[c.mode]
	if !ok {
		return fmt.Errorf("unknown -mode %q: want one of %s", c.mode, strings.Join(modeNames(), ", "))
	}

	cli, err := client.NewClient(c.node)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", c.node, err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	s, err := newSession(cli, c)
	if err != nil {
		return err
	}

	fmt.Printf("node:              %s\n", c.node)
	fmt.Printf("shielded contract: %s\n", s.contract.Base58())

	return handler(ctx, s)
}

// newSession resolves the state every flow needs. The transparent account is
// only used to pay for transactions and to own the TRC-20 balance; it is not
// part of the shielded key hierarchy, so modes that never broadcast (walletgen,
// scan) run without one.
func newSession(cli *client.Client, c *config) (*session, error) {
	s := &session{cli: cli, cfg: c}

	if c.privateKey != "" {
		acct, err := signer.NewPrivateKeySigner(c.privateKey)
		if err != nil {
			return nil, fmt.Errorf("parse transparent private key: %w", err)
		}
		s.acct = acct
		fmt.Printf("transparent account: %s\n", acct.Address().Base58())
	}

	contract, err := types.NewAddressFromBase58(c.shieldedContract)
	if err != nil {
		return nil, fmt.Errorf("parse shielded contract address %q: %w", c.shieldedContract, err)
	}
	if len(contract.Bytes()) != 21 {
		return nil, fmt.Errorf("shielded contract address %q decodes to %d bytes, want 21",
			c.shieldedContract, len(contract.Bytes()))
	}
	s.contract = contract
	return s, nil
}

// owner is the transparent account address. Callers must have gone through
// needAccount first, which is what makes the nil check there load-bearing.
func (s *session) owner() *types.Address { return s.acct.Address() }

// needAccount fails with an actionable message when a mode that broadcasts
// was started without a transparent key to pay fees with.
func (s *session) needAccount(why string) error {
	if s.acct == nil {
		return fmt.Errorf("-private-key (or NILE_TEST_KEY1) is required for %s", why)
	}
	return nil
}

// parseAmount reads -amount as a raw base-unit integer.
//
// Amounts are deliberately kept in the same representation the node API uses
// (from_amount / to_amount strings) rather than converted through the token's
// decimals. The only conversion in this example is the scaling factor, which is
// read from the contract itself.
func (c *config) parseAmount() (*big.Int, error) {
	raw := strings.TrimSpace(c.amount)
	if raw == "" {
		return nil, fmt.Errorf("-amount is required (raw token units, e.g. 10000000 for 10 tokens with 6 decimals)")
	}
	amount, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("-amount %q is not a base-10 integer", c.amount)
	}
	if amount.Sign() <= 0 {
		return nil, fmt.Errorf("-amount must be positive, got %s", amount)
	}
	return amount, nil
}
