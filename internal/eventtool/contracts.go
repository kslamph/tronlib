package eventtool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// DefaultSnapshotPath is where the contract ranking is snapshotted. It is a
// tracked file: capture reads it, never the network, so the corpus is a pure
// function of (snapshot, on-chain ABIs) and the same commit regenerates the
// same output.
const DefaultSnapshotPath = "internal/eventdata/top_contracts.json"

// tronscanContractsPath is the ranking endpoint. The site's default list is
// sorted by trxCount; the API's public limit caps at 50, so a top-100 snapshot
// is two pages.
const tronscanContractsPath = "https://apilist.tronscanapi.com/api/contracts"

// DefaultContractsURL is the real TronScan ranking endpoint; --api overrides it.
const DefaultContractsURL = tronscanContractsPath

// pageSize is the API's maximum rows per request (100 → HTTP 400).
const pageSize = 50

// pagePause paces requests under TronScan's 5 req/s public limit. Tests zero it.
var pagePause = 250 * time.Millisecond

// ContractEntry is one contract in the ranking snapshot. verify_status is
// recorded for provenance only: every deployed TRON contract carries an
// on-chain ABI whether or not its source is verified, so capture does not
// filter on it.
type ContractEntry struct {
	Rank         int    `json:"rank"`
	Address      string `json:"address"`
	Name         string `json:"name"`
	TrxCount     uint64 `json:"trxCount"`
	VerifyStatus int    `json:"verify_status"`
}

// Snapshot is the recorded TronScan contract ranking.
type Snapshot struct {
	Source    string          `json:"source"`
	RankBy    string          `json:"rank_by"`
	FetchedAt string          `json:"fetched_at"`
	Limit     int             `json:"limit"`
	Contracts []ContractEntry `json:"contracts"`
}

// tronscanPage is the subset of the endpoint's response this tool needs.
type tronscanPage struct {
	Data []struct {
		Address      string `json:"address"`
		Name         string `json:"name"`
		TrxCount     uint64 `json:"trxCount"`
		VerifyStatus int    `json:"verify_status"`
	} `json:"data"`
}

// FetchTop reads the top limit contracts ranked by trxCount and records them
// in a Snapshot. baseURL is the endpoint (a test server may substitute one);
// pass tronscanContractsPath for the real thing.
func FetchTop(ctx context.Context, hc *http.Client, baseURL string, limit int) (*Snapshot, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("eventtool: limit must be positive, got %d", limit)
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	snap := &Snapshot{
		Source:    baseURL + "?sort=-trxCount",
		RankBy:    "trxCount",
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Limit:     limit,
	}
	for start := 0; len(snap.Contracts) < limit; start += pageSize {
		if start > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pagePause):
			}
		}
		want := limit - len(snap.Contracts)
		if want > pageSize {
			want = pageSize
		}
		page, err := fetchPage(ctx, hc, baseURL, start, want)
		if err != nil {
			return nil, err
		}
		for _, c := range page.Data {
			snap.Contracts = append(snap.Contracts, ContractEntry{
				Rank:         len(snap.Contracts) + 1,
				Address:      c.Address,
				Name:         c.Name,
				TrxCount:     c.TrxCount,
				VerifyStatus: c.VerifyStatus,
			})
		}
		if len(page.Data) < want {
			break // final short page: nothing more to rank
		}
	}
	return snap, nil
}

func fetchPage(ctx context.Context, hc *http.Client, baseURL string, start, limit int) (*tronscanPage, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("eventtool: bad contracts URL %q: %w", baseURL, err)
	}
	q := u.Query()
	q.Set("sort", "-trxCount")
	q.Set("start", fmt.Sprintf("%d", start))
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if key := os.Getenv("TRONSCAN_API_KEY"); key != "" {
		req.Header.Set("TRON-PRO-API-KEY", key)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eventtool: fetch contracts: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("eventtool: read contracts response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eventtool: contracts API returned %s: %s", resp.Status, truncate(body, 200))
	}
	var page tronscanPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("eventtool: decode contracts response: %w", err)
	}
	return &page, nil
}

// WriteTo writes the snapshot atomically (temp file + rename), indented and
// newline-terminated so its diffs are reviewable.
func (s *Snapshot) WriteTo(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("eventtool: encode snapshot: %w", err)
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("eventtool: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("eventtool: replace %s: %w", path, err)
	}
	return nil
}

// LoadSnapshot reads a contract ranking snapshot. capture calls this and
// refuses to run when it fails — it never falls back to a live fetch.
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("eventtool: read snapshot %s: %w", path, err)
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("eventtool: decode snapshot %s: %w", path, err)
	}
	if len(s.Contracts) == 0 {
		return nil, fmt.Errorf("eventtool: snapshot %s has no contracts", path)
	}
	return &s, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
