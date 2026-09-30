// Command eventtool maintains tronlib's 32-byte event corpus and the built-in
// table generated from it.
//
// Subcommands:
//
//	contracts  snapshot the TronScan top-N contract ranking (by call volume)
//	capture    fetch those contracts' on-chain ABIs into the corpus
//	insert     add events from an ABI file
//	migrate    rewrite a v1 selector corpus into the 32-byte schema
//	generate   render event/builtin_gen.go from the corpus
//
// The corpus is data under internal/eventdata/; event/builtin_gen.go is
// generated output. Regeneration recipe (see docs/runbook.md):
//
//	go run ./cmd/eventtool contracts
//	go run ./cmd/eventtool capture
//	go run ./cmd/eventtool generate
//
// Design: docs/superpowers/specs/2026-09-30-eventtool-design.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	tronlib "github.com/kslamph/tronlib/v2"
	"github.com/kslamph/tronlib/v2/internal/eventtool"
)

// defaultNode is the local Envoy gRPC proxy (~/envoy/envoy.yaml: listener 50051
// round-robins 19 mainnet full nodes with retries and no rate limits).
const defaultNode = "grpc://127.0.0.1:50051"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "contracts":
		err = runContracts(ctx, os.Args[2:])
	case "capture":
		err = runCapture(ctx, os.Args[2:])
	case "insert":
		err = runInsert(os.Args[2:])
	case "migrate":
		err = runMigrate(os.Args[2:])
	case "generate":
		err = runGenerate(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "eventtool: unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "eventtool: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `eventtool — 32-byte event corpus tooling

Usage: eventtool <command> [flags]

  contracts  snapshot the TronScan top-N contract ranking
             --limit=100 --out=internal/eventdata/top_contracts.json --api=<endpoint>
  capture    fetch snapshotted contracts' ABIs into the corpus
             --node=`+defaultNode+` --in=internal/eventdata/top_contracts.json
             --out=internal/eventdata/events_registry.json --concurrency=1
  insert     add events from an ABI file
             --in=<abi.json> --out=internal/eventdata/events_registry.json
  migrate    rewrite a v1 selector corpus into the 32-byte schema
             --in=internal/eventdata/events_registry.json --out=<same as --in>
  generate   render the built-in table from the corpus
             --in=internal/eventdata/events_registry.json --out=event/builtin_gen.go
`)
}

func runContracts(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("contracts", flag.ExitOnError)
	limit := fs.Int("limit", 100, "number of contracts to rank")
	out := fs.String("out", eventtool.DefaultSnapshotPath, "snapshot output path")
	api := fs.String("api", eventtool.DefaultContractsURL, "TronScan contracts endpoint")
	_ = fs.Parse(args)

	snap, err := eventtool.FetchTop(ctx, http.DefaultClient, *api, *limit)
	if err != nil {
		return err
	}
	if err := snap.WriteTo(*out); err != nil {
		return err
	}
	fmt.Printf("contracts: %d ranked by %s -> %s\n", len(snap.Contracts), snap.RankBy, *out)
	return nil
}

func runCapture(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	node := fs.String("node", defaultNode, "TRON gRPC endpoint")
	in := fs.String("in", eventtool.DefaultSnapshotPath, "contract snapshot path")
	out := fs.String("out", eventtool.DefaultCorpusPath, "corpus output path")
	concurrency := fs.Int("concurrency", 1, "parallel GetContract calls")
	_ = fs.Parse(args)

	snap, err := eventtool.LoadSnapshot(*in)
	if err != nil {
		return err
	}
	store, err := openStore(*out)
	if err != nil {
		return err
	}
	cli, err := tronlib.Dial(ctx, *node)
	if err != nil {
		return err
	}
	defer cli.Close()

	rep, err := eventtool.Capture(ctx, snap, eventtool.NewRPCFetcher(cli.Raw()), store, *concurrency)
	if err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}
	fmt.Printf("capture: %d contracts, %d with events, %d new events, %d skipped -> %s\n",
		rep.Contracts, rep.WithEvents, rep.NewEvents, rep.Skipped, *out)
	return nil
}

func runInsert(args []string) error {
	fs := flag.NewFlagSet("insert", flag.ExitOnError)
	in := fs.String("in", "", "ABI file to read (required)")
	out := fs.String("out", eventtool.DefaultCorpusPath, "corpus output path")
	_ = fs.Parse(args)
	if *in == "" {
		return fmt.Errorf("insert requires --in")
	}
	data, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	store, err := openStore(*out)
	if err != nil {
		return err
	}
	added, err := eventtool.InsertABI(data, store)
	if err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}
	fmt.Printf("insert: %d new event(s) added -> %s\n", added, *out)
	return nil
}

func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	in := fs.String("in", eventtool.DefaultCorpusPath, "v1 corpus to read")
	out := fs.String("out", eventtool.DefaultCorpusPath, "corpus output path")
	_ = fs.Parse(args)

	data, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	events, err := eventtool.Migrate(data)
	if err != nil {
		return err
	}
	store := eventtool.New(*out)
	for _, e := range events {
		store.Upsert(e)
	}
	if err := store.Save(); err != nil {
		return err
	}
	fmt.Printf("migrate: %d entries -> %s\n", store.Len(), *out)
	return nil
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	in := fs.String("in", eventtool.DefaultCorpusPath, "corpus to read")
	out := fs.String("out", eventtool.DefaultGeneratedPath, "generated file to write")
	_ = fs.Parse(args)

	store, err := eventtool.Load(*in)
	if err != nil {
		return err
	}
	src, err := eventtool.Generate(store.Events())
	if err != nil {
		return err
	}
	if err := writeFileAtomic(*out, src); err != nil {
		return err
	}
	fmt.Printf("generate: %d definitions -> %s\n", store.Len(), *out)
	return nil
}

// openStore loads the corpus at path when it exists (verifying it) and starts a
// fresh one otherwise, so the first capture/insert can create it.
func openStore(path string) (*eventtool.Store, error) {
	switch _, err := os.Stat(path); {
	case err == nil:
		return eventtool.Load(path)
	case !os.IsNotExist(err):
		return nil, err
	default:
		return eventtool.New(path), nil
	}
}

// writeFileAtomic writes data to path via a temp file + rename so a failed
// generation never leaves a half-written source file.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
