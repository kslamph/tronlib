package eventtool

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// serveFixtures returns a server that answers the two recorded TronScan pages
// and records each request's (start, limit) pair, so the test can prove the
// paging shape rather than just the merged result.
func serveFixtures(t *testing.T) (*httptest.Server, *[][2]string) {
	t.Helper()
	page1, err := os.ReadFile("testdata/contracts_page1.json")
	if err != nil {
		t.Fatal(err)
	}
	page2, err := os.ReadFile("testdata/contracts_page2.json")
	if err != nil {
		t.Fatal(err)
	}
	var requests [][2]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		requests = append(requests, [2]string{q.Get("start"), q.Get("limit")})
		if q.Get("sort") != "-trxCount" {
			http.Error(w, "expected sort=-trxCount", http.StatusBadRequest)
			return
		}
		switch q.Get("start") {
		case "0":
			w.Write(page1)
		case "50":
			w.Write(page2)
		default:
			http.Error(w, "unexpected start", http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	old := pagePause
	pagePause = 0
	t.Cleanup(func() { pagePause = old })
	return srv, &requests
}

func TestFetchTopPagesAtFifty(t *testing.T) {
	srv, requests := serveFixtures(t)
	snap, err := FetchTop(context.Background(), srv.Client(), srv.URL, 100)
	if err != nil {
		t.Fatalf("FetchTop: %v", err)
	}
	if len(*requests) != 2 {
		t.Fatalf("requests = %d (%v), want exactly 2 (limit caps at 50)", len(*requests), *requests)
	}
	if (*requests)[0] != [2]string{"0", "50"} || (*requests)[1] != [2]string{"50", "50"} {
		t.Fatalf("paging = %v, want start 0 then 50, limit 50", *requests)
	}
	if len(snap.Contracts) != 100 {
		t.Fatalf("contracts = %d, want 100", len(snap.Contracts))
	}
	for i, c := range snap.Contracts {
		if c.Rank != i+1 {
			t.Fatalf("contract %d has rank %d", i, c.Rank)
		}
	}
	if snap.RankBy != "trxCount" || snap.Limit != 100 {
		t.Fatalf("snapshot header = %+v", snap)
	}
	if snap.Contracts[0].Address != "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" {
		t.Fatalf("rank 1 = %s, want USDT", snap.Contracts[0].Address)
	}
	if snap.Contracts[0].TrxCount != 3644148431 {
		t.Fatalf("rank 1 trxCount = %d", snap.Contracts[0].TrxCount)
	}
}

func TestFetchTopStopsAtFiftyWithSinglePage(t *testing.T) {
	srv, requests := serveFixtures(t)
	snap, err := FetchTop(context.Background(), srv.Client(), srv.URL, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(*requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(*requests))
	}
	if len(snap.Contracts) != 50 {
		t.Fatalf("contracts = %d, want 50", len(snap.Contracts))
	}
}

// TestFetchTopStopsOnShortPage: when the API returns fewer rows than requested
// (the ranking ran out early), the walk must stop after that page rather than
// keep requesting. Nothing more is coming; continuing would either loop or
// error. A mutated loop that ignores the short page issues start=100, which
// this server rejects, so the test fails loudly instead of hanging.
func TestFetchTopStopsOnShortPage(t *testing.T) {
	page1, err := os.ReadFile("testdata/contracts_page1.json")
	if err != nil {
		t.Fatal(err)
	}
	var p2 struct {
		Data []json.RawMessage `json:"data"`
	}
	raw2, err := os.ReadFile("testdata/contracts_page2.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw2, &p2); err != nil {
		t.Fatal(err)
	}
	short, err := json.Marshal(map[string]any{"data": p2.Data[:10]})
	if err != nil {
		t.Fatal(err)
	}
	var requests [][2]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		requests = append(requests, [2]string{q.Get("start"), q.Get("limit")})
		switch q.Get("start") {
		case "0":
			w.Write(page1)
		case "50":
			w.Write(short)
		default:
			http.Error(w, "past the short page", http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	old := pagePause
	pagePause = 0
	t.Cleanup(func() { pagePause = old })

	snap, err := FetchTop(context.Background(), srv.Client(), srv.URL, 100)
	if err != nil {
		t.Fatalf("FetchTop: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %v, want exactly 2 (short page must end the walk)", requests)
	}
	if len(snap.Contracts) != 60 {
		t.Fatalf("contracts = %d, want 60", len(snap.Contracts))
	}
}

func TestSnapshotWriteAndLoadRoundTrip(t *testing.T) {
	srv, _ := serveFixtures(t)
	snap, err := FetchTop(context.Background(), srv.Client(), srv.URL, 100)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "top_contracts.json")
	if err := snap.WriteTo(path); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	got, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if len(got.Contracts) != 100 || got.Contracts[99].Rank != 100 {
		t.Fatalf("round-trip lost entries: %d", len(got.Contracts))
	}
}

func TestLoadSnapshotRefusesMissing(t *testing.T) {
	if _, err := LoadSnapshot(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("want an error for a missing snapshot (capture must not live-fetch)")
	}
}

func TestFetchTopHonoursContextCancel(t *testing.T) {
	srv, _ := serveFixtures(t)
	old := pagePause
	pagePause = 50 * time.Millisecond
	defer func() { pagePause = old }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := FetchTop(ctx, srv.Client(), srv.URL, 100); err == nil {
		t.Fatal("want an error from a cancelled context")
	}
}
