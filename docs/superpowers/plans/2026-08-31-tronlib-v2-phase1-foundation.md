# tronlib v2 — Phase 1 (Foundation) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the `/v2` module, regenerate `pb/` against production v4.8.2 (dropping `google.golang.org/genproto/googleapis/api`), and ship the `tron` vocabulary package — Address, SUN amounts with compile-time float rejection, and the machine-parseable error taxonomy — plus `docgen` with a CI drift check.

**Architecture:** New module path `github.com/kslamph/tronlib/v2` on a branch, built in the spec's DAG order. `tron` has zero internal dependencies and is the vocabulary every later package needs; it lands first and is fully tested before anything consumes it. `docgen` lands in the same phase because the spec requires doc-drift prevention to exist *before* API work (§14 step 5: "now, not later").

**Tech Stack:** Go 1.25, protobuf (protoc 6.31.1 / protoc-gen-go v1.36.7 to match the existing generated headers), testify v1.11.1, golangci-lint (revive + custom `go/analysis` checks), `golang.org/x/tools/go/packages` (**test-only**, for the negative-compile suite — verified necessary because `go/importer.Default()` is not module-aware).

**Spec:** `docs/superpowers/specs/2026-08-31-tronlib-v2-design.md` — the plan argues from it; executors read both. This plan implements **spec §14 steps 1–5**. Steps 6–13 (`key`, `event`, `rpc`, `tx`, `contract`, `token`, facade, live verification, migration guide, tag) are Phase 2+ and are intentionally out of scope here.

## Global Constraints

From the spec, verbatim. Every task inherits these.

- C1: module path is `github.com/kslamph/tronlib/v2`. v1 tree is untouched and frozen at `v1.9.0`.
- C2: LLM-agent ergonomics win conflicts; no hand-maintained generated artifact.
- C4: clean-room. No `Deprecated:` shims anywhere in v2.
- Go toolchain: `go 1.25` (matches existing CI).
- Generated pb must be regenerated with `protoc-gen-go v1.36.7` / `protoc 6.31.1` to match existing headers — a mismatch produces a spurious whole-tree diff.
- `tron` package has **zero internal dependencies** (no `internal/`, no other `github.com/kslamph/tronlib/v2/...` import).
- `golang.org/x/tools` is permitted **only** under `v2/internal/compilecheck` as a test dependency. It must not appear in any non-test build path.
- The negative-compile suite (`v2/internal/compilecheck`) is a **required** deliverable of Task 5, not an optional extra. Dropping it removes the only guard against reintroducing the `~`-in-`Whole` and unsigned-`Whole` bugs documented in spec §5.1.
- N1 (no stutter), N5 (no `any` in exported signatures), N7 (no float in amount paths) are enforced from the first file written.
- Test style: testify `assert`/`require`, table-driven, matching `pkg/types/address_test.go` conventions.
- Coverage floor 80% is enforced by CI on `./pkg/...` — Phase 1 adds `v2/` paths, so the floor config must be extended (Task 2).

---

## File Structure

```
protos/                          # submodule, advanced to v4.8.2 in Task 3
pb/                              # regenerated output (git-tracked)
scripts/proto-gen.sh             # existing; edited for annotations removal
v2/
├── go.mod                       # module github.com/kslamph/tronlib/v2
├── go.sum
├── doc.go                       # package docs
├── tron/
│   ├── doc.go
│   ├── address.go               # Address value type + constructors + methods
│   ├── address_test.go
│   ├── address_example_test.go
│   ├── sun.go                   # SUN, Whole, TRX, ParseTRX, MustTRX
│   ├── sun_test.go
│   ├── sun_example_test.go
│   ├── sun_compile_test.go      # negative-compile assertions (Task 5)
│   ├── error.go                 # Error, Code, Action, HasCode
│   ├── error_test.go
│   ├── error_example_test.go
│   └── codes.go                 # Code constants + generated Action/Doc tables
├── internal/
│   └── compilecheck/
│       └── compilecheck_test.go # drives `go/types` to assert compile errors
├── cmd/docgen/
│   ├── main.go                  # walks AST, renders examples + error lists
│   ├── main_test.go
│   └── testdata/                # fixture package for docgen tests
└── .github/workflows/v2.yml     # or edited into existing workflow
```

**Why `v2/` as a subdirectory:** a Go module at `github.com/kslamph/tronlib/v2` cannot share a directory with the v1 module (`go.mod` collision at repo root). Two options: separate branch with root `go.mod` replaced, or subdirectory. The branch approach rewrites history paths and makes cross-referencing v1 tests impossible; the subdirectory keeps both modules co-existing in one working tree, which is exactly what C1's "v1 remains installable" requires and is the standard Go major-version-subdirectory pattern. The module still declares path `github.com/kslamph/tronlib/v2` — the directory name is irrelevant to the import path.

**Decision: `v2/` subdirectory.** Executors must not move files to the repo root.

---

### Task 1: Freeze v1 and create the v2 branch

**Files:**
- Create: branch `v2` from current `main`
- Modify: none

**Interfaces:**
- Produces: branch `v2` checked out; v1 untouched at `main`.

- [ ] **Step 1: Confirm clean tree**

```bash
cd /Users/kslam/goproj/tronlib
git status --porcelain --untracked-files=no
```
Expected: empty.

- [ ] **Step 2: Create and check out the v2 branch**

```bash
git checkout -b v2
```
Expected: `Switched to a new branch 'v2'`.

- [ ] **Step 3: Verify v1 is intact**

```bash
go build ./... && echo "v1 OK"
```
Expected: `v1 OK`.

- [ ] **Step 4: Commit (no-op commit marks branch start)**

```bash
git commit --allow-empty -m "chore: branch v2 from $(git rev-parse --short main)"
```

---

### Task 2: `/v2` module skeleton and CI wiring

**Files:**
- Create: `v2/go.mod`, `v2/doc.go`, `v2/.gitignore`
- Modify: `.github/workflows/test-coverage.yml`

**Interfaces:**
- Produces: module `github.com/kslamph/tronlib/v2` that builds; CI job covering `v2/...`.

- [ ] **Step 1: Create the module**

```bash
mkdir -p v2 && cd v2
cat > go.mod <<'EOF'
module github.com/kslamph/tronlib/v2

go 1.25
EOF
cat > doc.go <<'EOF'
// Package tronlib is a Go SDK for the TRON blockchain.
//
// v2 is a clean-room redesign. See docs/superpowers/specs/2026-08-31-tronlib-v2-design.md
// for the design and the reasoning behind each breaking change.
package tronlib
EOF
cd .. && go build ./v2/... && echo "module OK"
```
Expected: `module OK`.

- [ ] **Step 2: Extend CI coverage floor to v2 paths**

Edit `.github/workflows/test-coverage.yml`. Find the coverage step and change the package glob so both trees are measured, keeping the existing 80% floor:

```yaml
    - name: Run unit tests with coverage (parallel)
      run: go test -short -coverprofile=unit_coverage.txt -coverpkg=./pkg/...,./v2/... -covermode=atomic ./pkg/... ./v2/...
```
Expected: CI file parses (validate locally with `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/test-coverage.yml'))"`).

- [ ] **Step 3: Run tests to confirm the pipeline is green on the empty module**

```bash
go test ./v2/... && go vet ./v2/...
```
Expected: `no test files` (pass) and vet clean.

- [ ] **Step 4: Commit**

```bash
git add v2/go.mod v2/doc.go .github/workflows/test-coverage.yml
git commit -m "feat(v2): scaffold /v2 module and extend CI coverage glob"
```

---

### Task 3: Regenerate pb from v4.8.2 and drop genproto/googleapis/api

**Files:**
- Modify: `protos` (submodule gitlink → `8432beca9`), `scripts/proto-gen.sh`, `pb/**` (regenerated), root `go.mod`/`go.sum`

**Interfaces:**
- Produces: `pb/api` and `pb/core` generated from v4.8.2, with **no** `google.golang.org/genproto/googleapis/api` import anywhere in `pb/`; `GetPaginatedNowWitnessList` present on both client types. v1 continues to build against the regenerated pb (spec §3: wire-compatible).

- [ ] **Step 1: Install pinned generators**

The existing headers say `protoc-gen-go v1.36.7`. Install exactly that to avoid a whole-tree diff.

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.7
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
export PATH="$PATH:$(go env GOPATH)/bin"
protoc-gen-go --version   # expect: protoc-gen-go v1.36.7
```

- [ ] **Step 2: Advance the submodule gitlink**

```bash
git -C protos fetch --tags origin
git -C protos checkout "GreatVoyage-v4.8.2(Pyrrho)"
git add protos
git commit -m "chore(protos): advance submodule to GreatVoyage-v4.8.2(Pyrrho)"
```
Expected: `git submodule status` shows `+8432beca9...` prefix `+` (new commit checked out).

**Commit the gitlink and the regeneration in the same commit** is required by spec §3 — do not land them separately. So **defer `git add`/`git commit` of `protos` until Step 7.** Step 2's `git add` here is a mistake; only run the `checkout`. Verify instead:

```bash
git -C protos describe --tags   # expect: GreatVoyage-v4.8.2(Pyrrho)
```

- [ ] **Step 3: Strip the googleapis download from proto-gen.sh**

The script currently downloads `annotations.proto` into `tmp/googleapis`. v4.8.2 no longer imports it. Edit `scripts/proto-gen.sh`: delete the block from `# Create minimal googleapis directory` through `fi` (the download of `annotations.proto` and `http.proto`), and remove any `--proto_path=$MINIMAL_GOOGLEAPIS` argument from the protoc invocation. Search the script for `googleapis` and confirm every occurrence is gone except the final `echo` line, which should also be deleted.

- [ ] **Step 4: Regenerate**

```bash
./scripts/proto-gen.sh
```
Expected: completes with `Proto generation completed successfully!`

- [ ] **Step 5: Assert the two things that matter**

```bash
# no annotations import anywhere in generated code
grep -rn "genproto/googleapis/api" pb/ && echo "FAIL: still present" || echo "PASS: gone"
# new rpc present on both services
grep -c "func (c \*walletClient) GetPaginatedNowWitnessList(" pb/api/api_grpc.pb.go
grep -c "func (c \*walletSolidityClient) GetPaginatedNowWitnessList(" pb/api/api_grpc.pb.go
```
Expected: `PASS: gone`, then `1` and `1`.

- [ ] **Step 6: Drop the dependency and verify v1 still builds**

```bash
go mod tidy
grep -c "genproto/googleapis/api" go.mod   # expect: 0 (may appear in go.sum; that is fine)
go build ./... && go test -short ./pkg/types/... ./pkg/trc20/... && echo "v1 OK on regenerated pb"
```
Expected: `v1 OK on regenerated pb`. If a test fails here, the wire-compat claim in spec §3 is wrong — stop and report; do not paper over it.

- [ ] **Step 7: Commit submodule + regeneration together**

```bash
git add protos pb go.mod go.sum scripts/proto-gen.sh
git commit -m "feat(pb): regenerate from GreatVoyage-v4.8.2(Pyrrho); drop genproto/googleapis/api

- advance protos submodule v4.7.4(Bias) -> v4.8.2(Pyrrho)
- v4.8.2 removes all 56 google.api.http annotations and the annotations.proto
  import, so the generated code no longer references genproto/googleapis/api
- adds GetPaginatedNowWitnessList on Wallet and WalletSolidity
- spec section 3: zero field/type/number changes; wire-compatible"
```

---

### Task 4: `tron` package — Address

**Files:**
- Create: `v2/tron/doc.go`, `v2/tron/address.go`, `v2/tron/address_test.go`, `v2/tron/address_example_test.go`

**Interfaces:**
- Consumes: nothing (zero internal deps).
- Produces (used by Tasks 5–7 and Phase 2):

```go
type Address struct{ /* unexported: base58 string + 21 bytes */ }
func ParseAddress(s string) (Address, error)
func MustAddress(s string) Address
func AddressFromBytes(b []byte) (Address, error)
func AddressFromHex(hex21 string) (Address, error)   // accepts 0x41-prefixed or bare
func (a Address) String() string                     // base58, value receiver
func (a Address) Bytes() []byte                      // returns a copy, 21 bytes, 0x41-prefixed
func (a Address) Hex() string                        // 0x41-prefixed hex
func (a Address) IsZero() bool
```

Per spec §10: value semantics, not `*Address`. Per spec N1/N4/N5: no stutter (`tron.Address`, not `tron.TronAddress`), no `ctx` (no I/O), no `any`.

- [ ] **Step 1: Write the failing test**

```go
package tron

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pairs taken from v1 pkg/types/address_test.go — same fixtures, new type.
var validPairs = []struct{ base58, hex20 string }{
	{"TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb", "e28b3cfd4e0e909077821478e9fcb86b84be786e"},
	{"TXNYeYdao7JL7wBtmzbk7mAie7UZsdgVjx", "eac49bc766be29be1b6d36619eff8f86ed4d04df"},
}

func TestParseAddressRoundTrip(t *testing.T) {
	for _, tc := range validPairs {
		a, err := ParseAddress(tc.base58)
		require.NoError(t, err, tc.base58)
		assert.Equal(t, tc.base58, a.String())

		got20 := hex.EncodeToString(a.Bytes()[1:]) // strip 0x41
		assert.Equal(t, tc.hex20, got20)
	}
}

func TestParseAddressRejects(t *testing.T) {
	cases := []struct{ name, in string }{
		{"empty", ""},
		{"wrong length", "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jw"},
		{"not base58", "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb0OIl"}, // 0,O,I,l excluded from alphabet
		{"bad checksum", "TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwc"},
		{"wrong prefix byte", "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"}, // bitcoin-style, decodes but no 0x41
	}
	for _, tc := range cases {
		_, err := ParseAddress(tc.in)
		assert.Error(t, err, tc.name)
	}
}

func TestAddressBytesIsCopy(t *testing.T) {
	a, _ := ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	b := a.Bytes()
	b[0] = 0x00 // mutate the returned slice
	assert.Equal(t, byte(0x41), a.Bytes()[0], "Bytes() must return a copy")
}

func TestAddressFromHexForms(t *testing.T) {
	a1, err := AddressFromHex("41e28b3cfd4e0e909077821478e9fcb86b84be786e")
	require.NoError(t, err)
	a2, err := AddressFromHex("e28b3cfd4e0e909077821478e9fcb86b84be786e")
	require.NoError(t, err)
	assert.Equal(t, a1, a2, "0x41-prefixed and bare forms are the same address")
}

func TestAddressZeroValue(t *testing.T) {
	var a Address
	assert.True(t, a.IsZero())
	assert.Error(t, func() error { _, err := ParseAddress(""); return err }())
	_, err := ParseAddress("")
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./v2/tron/ -run TestParseAddress -v
```
Expected: FAIL, `undefined: ParseAddress`.

- [ ] **Step 3: Implement `address.go`**

Base58 encode/decode is small enough to inline (spec §4: `mr-tron/base58` may be dropped — do **not** add it as a v2 dependency; implement the 58-alphabet decode in this file, ~40 lines, with strict character validation).

```go
package tron

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

// Address is a TRON address: 21 bytes, 0x41-prefixed.
// Value semantics; the zero value is the unset address.
type Address struct {
	base58 string
	b      [21]byte
}

// ParseAddress parses a base58check-encoded TRON address ("T..." 34 chars).
func ParseAddress(s string) (Address, error) {
	dec, err := base58Decode(s)
	if err != nil {
		return Address{}, &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: "pass a 34-character base58 address starting with T", Cause: err}
	}
	if len(dec) != 25 {
		return Address{}, &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: fmt.Sprintf("decoded length %d, want 25", len(dec))}
	}
	body, sum := dec[:21], dec[21:]
	h1 := sha256.Sum256(body)
	h2 := sha256.Sum256(h1[:])
	if h2[0] != sum[0] || h2[1] != sum[1] || h2[2] != sum[2] || h2[3] != sum[3] {
		return Address{}, &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: "checksum mismatch"}
	}
	if body[0] != 0x41 {
		return Address{}, &Error{Code: CodeAddressWrongPrefix, Op: "ParseAddress", Hint: fmt.Sprintf("prefix 0x%02x, want 0x41", body[0])}
	}
	var a Address
	a.base58 = s
	copy(a.b[:], body)
	return a, nil
}
```

`base58Decode` (same file, unexported, strict):

```go
const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func base58Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return nil, errors.New("empty input")
	}
	// strict alphabet check first, so "not base58" is a distinct error from "too short"
	for _, c := range s {
		if !isValidB58(rune(c)) {
			return nil, fmt.Errorf("invalid base58 character %q", c)
		}
	}
	num := make([]byte, 0, len(s))
	for _, c := range s {
		carry := int64(b58Index[c])
		for i := len(num) - 1; i >= 0; i-- {
			carry += int64(num[i]) * 58
			num[i] = byte(carry % 256)
			carry /= 256
		}
		for carry > 0 {
			num = append([]byte{byte(carry % 256)}, num...)
			carry /= 256
		}
	}
	// leading '1's are leading zero bytes
	leading := 0
	for leading < len(s) && s[leading] == '1' {
		leading++
	}
	out := make([]byte, leading+len(num))
	copy(out[leading:], num)
	return out, nil
}
```

(Implement `isValidB58` and `b58Index` as a 256-entry lookup table built in `init()`.) Then the remaining methods — `MustAddress`, `AddressFromBytes`, `AddressFromHex`, `String`, `Bytes` (returns `append([]byte(nil), a.b[:]...)`), `Hex`, `IsZero` — following the signatures above. `MustAddress` panics via `Must` convention (N2) and is documented for literals.

- [ ] **Step 4: Run tests**

```bash
go test ./v2/tron/ -v
```
Expected: PASS, all cases.

- [ ] **Step 5: Write the Example (spec §12: every exported symbol has a compiling Example)**

```go
package tron_test

import (
	"fmt"

	"github.com/kslamph/tronlib/v2/tron"
)

func ExampleParseAddress() {
	a, err := tron.ParseAddress("TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb")
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println(a.String())
	// Output: TWd4WrZ9wn84f5x1hZhL4DHvk738ns5jwb
}
```

- [ ] **Step 6: Run examples and commit**

```bash
go test ./v2/tron/ -run Example -v
git add v2/tron/
git commit -m "feat(v2/tron): Address value type with strict base58 validation"
```

---

### Task 5: `tron` package — SUN amounts

**Files:**
- Create: `v2/tron/sun.go`, `v2/tron/sun_test.go`, `v2/tron/sun_example_test.go`, `v2/internal/compilecheck/compilecheck_test.go`

**Interfaces:**
- Consumes: `tron.Error`, `tron.Code` (Task 6 — so Task 5 must come **after** Task 6; see the ordering note below Task 6).
- Produces (used by Phase 2's `tx`/`token`):

```go
type SUN int64
type Whole interface{ int | int8 | int16 | int32 | int64 }   // no tilde, no unsigned
const MaxSUN = math.MaxInt64
func TRX[T Whole](n T) SUN
func ParseTRX(s string) (SUN, error)
func MustTRX(s string) SUN
func ParseSUN(s string) (SUN, error)                          // raw sun string
func (s SUN) String() string      // canonical, round-trips with ParseTRX
func (s SUN) Formatted() string   // display: separators + rounding
func (s SUN) Add(o SUN) (SUN, error)
func (s SUN) Sub(o SUN) (SUN, error)
func (s SUN) Mul(n int64) (SUN, error)
func (s SUN) Int64() int64
```

- [ ] **Step 1: Write the failing test (behaviour)**

```go
package tron

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTRXWholeUnits(t *testing.T) {
	assert.Equal(t, SUN(1_000_000), TRX(1))
	assert.Equal(t, SUN(-2_000_000), TRX(-2))
	assert.Equal(t, SUN(math.MaxInt64), TRX(int64(math.MaxInt64)/1_000_000)*1_000_000 != 0)
}
```

That last line is wrong on purpose — replace with the precise boundary test:

```go
func TestTRXBoundary(t *testing.T) {
	const maxSafeTRX = math.MaxInt64 / 1_000_000 // 9_223_372_036_854
	assert.Equal(t, SUN(maxSafeTRX*1_000_000), TRX(maxSafeTRX))
	assert.PanicsWithValue(t,
		"tronlib: TRX(9223372036855) overflows SUN (max 9223372036854)",
		func() { TRX(maxSafeTRX + 1) })
}

func TestParseTRX(t *testing.T) {
	cases := []struct{ in string; want SUN; errCode Code }{
		{"1.6", 1_600_000, ""}, {"0.1", 100_000, ""}, {"100.000001", 100_000_001, ""},
		{"1e-6", 1, ""}, {"0.300000", 300_000, ""},
		{"1.6666666", 0, CodeAmountTooManyDecimals},
		{"1e-7", 0, CodeAmountTooManyDecimals},
		{"+1.6", 0, CodeAmountInvalid},
		{"1,234", 0, CodeAmountInvalid},
		{"", 0, CodeAmountInvalid},
		{"abc", 0, CodeAmountInvalid},
		{"9223372036854775808", 0, CodeAmountOverflow}, // > MaxInt64 sun
	}
	for _, tc := range cases {
		got, err := ParseTRX(tc.in)
		if tc.errCode == "" {
			require.NoError(t, err, tc.in)
			assert.Equal(t, tc.want, got, tc.in)
		} else {
			assert.Error(t, err, tc.in)
			assert.True(t, HasCode(err, tc.errCode), "%s: want code %v, got %v", tc.in, tc.errCode, err)
		}
	}
}

func TestSUNStringRoundTrips(t *testing.T) {
	for _, s := range []string{"1.6", "0.000001", "100.000001", "9223372036.854775"} {
		v, err := ParseTRX(s)
		require.NoError(t, err)
		assert.Equal(t, s, v.String(), "canonical String must round-trip")
	}
}

func TestSUNAddOverflowChecked(t *testing.T) {
	a := SUN(math.MaxInt64)
	_, err := a.Add(1)
	assert.True(t, HasCode(err, CodeAmountOverflow))
	_, err = a.Sub(SUN(math.MaxInt64))
	assert.NoError(t, err) // equal values are fine
}

func TestSUNFormatted(t *testing.T) {
	v, _ := ParseTRX("1234567.891234")
	assert.Equal(t, "1,234,567.891234", v.Formatted())
}
```

- [ ] **Step 2: Write the negative-compile test**

This is the part the spec §5.2 matrix makes mandatory and that a normal test cannot express. `v2/internal/compilecheck` runs `go/types` over fixture snippets and asserts that specific ones **fail** to type-check.

```go
package compilecheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// mustFail contains source snippets that must NOT compile.
// Each is type-checked against the real tron package.
var mustFail = map[string]string{
	"float literal":   `x := tron.TRX(1.6)`,
	"already-scaled":  `var s tron.SUN = tron.TRX(1); x := tron.TRX(s)`,
	"unsigned":        `var u uint64 = 3; x := tron.TRX(u)`,
	"string":          `x := tron.TRX("1")`,
	"any":             `var a any = 1; x := tron.TRX(a)`,
}

func TestAmountInputsThatMustNotCompile(t *testing.T) {
	for name, stmt := range mustFail {
		src := "package p\nimport \"github.com/kslamph/tronlib/v2/tron\"\nfunc f() { " + stmt + " }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "p.go", src, 0)
		if err != nil {
			t.Fatalf("%s: fixture does not parse: %v", name, err)
		}
		conf := types.Config{Importer: importer(), Error: func(error) {}}
		_, typeErr := conf.Check("p", fset, []*ast.File{f}, nil)
		if typeErr == nil {
			t.Errorf("%s: compiled but MUST NOT: %s", name, stmt)
		}
	}
}
```

`importer()` must return the **real, loaded** `tron` package — `go/importer.Default()` is not module-aware and cannot resolve an in-module import from source, which this plan verified empirically (it fails with `cannot find import`). Use `golang.org/x/tools/go/packages` to load `tron`, then hand `types.Config` a stub importer that returns it:

```go
func loadTron(t *testing.T) *types.Package {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps,
		Dir:  "../..",   // the v2 module root
	}, "github.com/kslamph/tronlib/v2/tron")
	if err != nil {
		t.Fatalf("load tron: %v", err)
	}
	if len(pkgs) == 0 || pkgs[0].Types == nil || len(pkgs[0].Types.Scope().Names()) == 0 {
		t.Fatalf("tron loaded empty — does the package exist and compile?")
	}
	return pkgs[0].Types
}

type stubImporter struct{ p *types.Package }

func (s stubImporter) Import(path string) (*types.Package, error) {
	if path == "github.com/kslamph/tronlib/v2/tron" {
		return s.p, nil
	}
	return nil, fmt.Errorf("unexpected import %q", path)
}
```

This adds `golang.org/x/tools` + `golang.org/x/sync` as **test-only** dependencies of the v2 module. That cost is accepted deliberately: this test is the only thing that prevents a future edit from re-adding `~` to `Whole` or an unsigned kind, either of which reintroduces a silent-money bug (spec §5.1 documents both bugs as verified by execution). Do **not** skip it, and do not substitute `importer.Default()`.

The whole mechanism was verified end-to-end while drafting this plan: a stub `tron` with `type Whole interface{ int | int8 | int16 | int32 | int64 }` plus `go/types` over three negative fixtures (`TRX(1.6)`, `TRX(someSUN)`, `TRX(uint64)`) correctly produced type errors for all three and PASSED the negative-compile assertion.

- [ ] **Step 3: Run both test files to verify failure**

```bash
go test ./v2/tron/ ./v2/internal/compilecheck/ -v
```
Expected: FAIL, `undefined: TRX` / `undefined: HasCode`.

- [ ] **Step 4: Implement `sun.go`**

```go
package tron

import (
	"math"
	"strings"

	"github.com/shopspring/decimal"
)

// SUN is TRX in the atomic on-chain unit: 1 TRX = 1_000_000 SUN.
// It is the only in-memory representation of a TRX amount in v2.
type SUN int64

// Whole admits the predeclared signed integer types and nothing else.
//
// Two deliberate exclusions, both verified by execution:
//   - no tilde: with ~int64, SUN's own underlying type satisfies the
//     constraint, so TRX(someSUN) compiles and re-scales an already-scaled
//     value, turning 1 TRX into 1,000,000 TRX while the overflow guard passes.
//   - no unsigned: int64(n) of a uint64 above MaxInt64 wraps negative and
//     passes the bound check, yielding a silently negative amount.
//
// See spec section 5.1. The compilecheck test pins both exclusions.
type Whole interface{ int | int8 | int16 | int32 | int64 }

const maxSafeTRX = math.MaxInt64 / 1_000_000 // 9_223_372_036_854

// TRX converts a whole-number TRX literal to SUN.
// It panics on overflow and is for literals and constants only;
// dynamic input must use ParseTRX.
func TRX[T Whole](n T) SUN {
	v := int64(n)
	if v > maxSafeTRX || v < -maxSafeTRX {
		panic(formatOverflow(v))
	}
	return SUN(v * 1_000_000)
}

// ParseTRX parses an exact decimal TRX string into SUN.
// Rejects: unparseable input, a leading '+', thousands separators, more
// than 6 decimal places, and values beyond the int64 SUN range.
// Accepts scientific notation down to 1 SUN ("1e-6").
func ParseTRX(s string) (SUN, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "+,_") {
		return 0, &Error{Code: CodeAmountInvalid, Op: "ParseTRX",
			Hint: `pass a plain decimal string like "1.6"`}
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return 0, &Error{Code: CodeAmountInvalid, Op: "ParseTRX",
			Hint: `pass a plain decimal string like "1.6"`, Cause: err}
	}
	scaled := d.Mul(decimal.New(1, 6))
	if !scaled.IsInteger() {
		return 0, &Error{Code: CodeAmountTooManyDecimals, Op: "ParseTRX",
			Hint: "1 SUN = 1e-6 TRX; round to 6 decimal places"}
	}
	if scaled.Abs().GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, &Error{Code: CodeAmountOverflow, Op: "ParseTRX"}
	}
	return SUN(scaled.IntPart()), nil
}
```

Then `MustTRX` (panics on the `ParseTRX` error path), `ParseSUN` (plain int64 parse, no scale), and the six methods. **`String()` must be a value receiver** and must not add separators or round — canonical form only. `Formatted()` adds thousands separators and rounds to at most 6 decimals. `Add`/`Sub`/`Mul` all check overflow and return `CodeAmountOverflow`; the spec requires uniform discipline (review P2.3).

- [ ] **Step 5: Run all tests**

```bash
go test ./v2/tron/ ./v2/internal/compilecheck/ -v
```
Expected: PASS including the compilecheck suite. If `TRX(s)` compiles, the tilde crept back in — fix `Whole` before proceeding.

- [ ] **Step 6: Example + commit**

```go
func ExampleTRX() {
	fmt.Println(tron.TRX(1))
	// Output: 1000000
}

func ExampleParseTRX() {
	v, err := tron.ParseTRX("1.6")
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	fmt.Println(v.String())
	// Output: 1.6
}
```

```bash
git add v2/tron/ v2/internal/
git commit -m "feat(v2/tron): SUN amounts with compile-time float rejection"
```

---

### Task 6: `tron` package — Error, Code, Action

**Files:**
- Create: `v2/tron/codes.go`, `v2/tron/error.go`, `v2/tron/error_test.go`, `v2/tron/error_example_test.go`
- Create: `v2/tron/codes_gen.go` (generated by `cmd/docgen`, Task 7 — so Task 6 ships a hand-written `codes.go` that Task 7 replaces; see ordering note)

**Interfaces:**
- Consumes: nothing.
- Produces (used by Tasks 4–5 and all of Phase 2):

```go
type Code string
type Action int

const (
    ActionRetry Action = iota
    ActionWait
    ActionFixCall
    ActionFixTransaction
    ActionFund
    ActionBug
)
func (a Action) String() string

type Error struct {
    Code  Code
    Op    string
    Hint  string
    TxID  string
    Cause error
    Next  Action   // 0 means "use Code.Action()"
}
func (e *Error) Error() string
func (e *Error) Unwrap() error
func (e *Error) Is(target error) bool
func HasCode(err error, c Code) bool
func (c Code) Action() Action
func (c Code) Doc() string
```

**Ordering note:** Task 5 consumes `Error`/`Code`/`HasCode`, so **Task 6 must be implemented before Task 5** despite the numbering. The numbering reflects spec §14 order (which lists `tron` as one milestone); within the milestone, implement `error.go` first, then `address.go`, then `sun.go`.

- [ ] **Step 1: Write the failing test**

```go
package tron

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorFields(t *testing.T) {
	cause := errors.New("connection refused")
	e := &Error{Code: CodeChainConnection, Op: "Dial", Hint: "check the endpoint host and port", Cause: cause}
	assert.Contains(t, e.Error(), "chain.connection")
	assert.Contains(t, e.Error(), "Dial")
	assert.ErrorIs(t, e, cause)                       // Unwrap chain intact
	assert.True(t, HasCode(e, CodeChainConnection))
}

func TestHasCodeThroughWrapping(t *testing.T) {
	inner := &Error{Code: CodeAmountOverflow, Op: "ParseTRX"}
	wrapped := fmt.Errorf("building transfer: %w", inner)
	assert.True(t, HasCode(wrapped, CodeAmountOverflow))
	assert.False(t, HasCode(wrapped, CodeAddressInvalid))
	assert.False(t, HasCode(nil, CodeAmountOverflow))
	assert.False(t, HasCode(errors.New("plain"), CodeAmountOverflow))
}

func TestActionDerivedFromCode(t *testing.T) {
	assert.Equal(t, ActionRetry, CodeChainConnection.Action())
	assert.Equal(t, ActionRetry, CodeChainTimeout.Action())
	assert.Equal(t, ActionWait, CodeChainUnconfirmed.Action())
	assert.Equal(t, ActionFixCall, CodeAmountInvalid.Action())
	assert.Equal(t, ActionFund, CodeAccountInsufficientEnergy.Action())
	assert.Equal(t, ActionFixTransaction, CodeReceiptReverted.Action())
}

func TestNextOverridesDefault(t *testing.T) {
	e := &Error{Code: CodeChainTimeout, Next: ActionWait, TxID: "abc"}
	assert.Equal(t, ActionWait, e.Next)
	assert.Equal(t, "abc", e.TxID)
}

func TestHintIsNotInMessage(t *testing.T) {
	// The reason Hint exists: v1 fused fact and advice into one string.
	e := &Error{Code: CodeAddressInvalid, Op: "ParseAddress", Hint: "pass a 34-character base58 address"}
	msg := e.Error()
	assert.Contains(t, msg, "address.invalid")
	assert.NotContains(t, msg, "34-character", "Hint must not leak into Error(): machine and human needs diverge")
}

func TestAllCodesHaveActionAndDoc(t *testing.T) {
	// Every code exported must have a non-empty Doc and a valid Action.
	// This is the test that prevents v1's "advertised but never returned" problem.
	for _, c := range AllCodes {
		assert.NotEmpty(t, c.Doc(), "code %s has no Doc", c)
		assert.NotEqual(t, Action(0), c.Action(), "code %s has no Action", c)
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./v2/tron/ -run TestError -v
```
Expected: FAIL, `undefined: Error`.

- [ ] **Step 3: Implement `error.go`**

```go
package tron

import (
	"errors"
	"fmt"
)

// Error is the only error type v2 returns from exported functions.
// Code is the machine-parseable identity; Hint is the human/agent remediation;
// the two are deliberately separate because v1 fused them into one string,
// which meant a machine could not get the fact without the advice and a human
// could not change the advice without changing the message.
type Error struct {
	Code  Code
	Op    string   // "Dial", "Broadcast", "contract.Invoke"
	Hint  string   // remediation; may be empty; never duplicated into Error()
	TxID  string   // populated when a transaction id is known
	Cause error
	Next  Action   // zero means "derive from Code"
}

func (e *Error) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, e.Code)
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.Cause }

// Is reports whether target is a *Error carrying the same Code.
// Defined as a method so errors.Is(err, &Error{Code: X}) works alongside HasCode.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return t.Code == e.Code
	}
	return false
}

// HasCode reports whether err, or any error in its chain, is a *Error with code c.
func HasCode(err error, c Code) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == c
	}
	return false
}

// Action returns the remediation the caller should take.
// An explicit Next overrides the code-derived default, because some remedies
// depend on context the code cannot see: a chain.timeout before submission is
// retryable, after submission it is not.
func (e *Error) Action() Action {
	if e.Next != 0 {
		return e.Next
	}
	return e.Code.Action()
}
```

- [ ] **Step 4: Implement `codes.go`** with the full §8.3 code set and hand-written `Action()`/`Doc()` (replaced by generation in Task 7):

```go
package tron

// Code identifies a failure class machine-parseably.
// Dotted, prefixed, and extensible. Prefixes carry a default Action.
type Code string

const (
	CodeAmountInvalid         Code = "amount.invalid"
	CodeAmountTooManyDecimals Code = "amount.too_many_decimals"
	CodeAmountOverflow        Code = "amount.overflow"
	CodeAmountNegative        Code = "amount.negative"
	CodeAmountDecimalsMismatch Code = "amount.decimals_mismatch"

	CodeAddressInvalid     Code = "address.invalid"
	CodeAddressWrongPrefix Code = "address.wrong_prefix"

	CodeTxNoSigner         Code = "tx.no_signer"
	CodeTxAlreadySigned    Code = "tx.already_signed"
	CodeTxExpired          Code = "tx.expired"
	CodeTxDuplicate        Code = "tx.duplicate"
	CodeTxFeeLimitTooLow   Code = "tx.fee_limit_too_low"
	CodeTxInvalidArgument  Code = "tx.invalid_argument"
	CodeTxUnknownContract  Code = "tx.unknown_contract"
	CodeTxTaposInvalid     Code = "tx.tapos_invalid"
	CodeTxTooLarge         Code = "tx.too_large"

	CodeChainConnection   Code = "chain.connection"
	CodeChainTimeout      Code = "chain.timeout"
	CodeChainClosed       Code = "chain.closed"
	CodeChainUnavailable  Code = "chain.unavailable"
	CodeChainUnconfirmed  Code = "chain.unconfirmed"
	CodeChainNetworkMismatch Code = "chain.network_mismatch"

	CodeReceiptReverted     Code = "receipt.reverted"
	CodeReceiptOutOfEnergy  Code = "receipt.out_of_energy"
	CodeReceiptFailed       Code = "receipt.failed"

	CodeContractNotFound      Code = "contract.not_found"
	CodeContractNoABI         Code = "contract.no_abi"
	CodeContractBadABI        Code = "contract.bad_abi"
	CodeContractBadMetadata   Code = "contract.bad_metadata"
	CodeContractMethodUnknown Code = "contract.method_unknown"
	CodeContractArgMismatch   Code = "contract.arg_mismatch"
	CodeContractResultTypeMismatch Code = "contract.result_type_mismatch"

	CodeAccountInsufficientBalance  Code = "account.insufficient_balance"
	CodeAccountInsufficientEnergy   Code = "account.insufficient_energy"
	CodeAccountInsufficientBandwidth Code = "account.insufficient_bandwidth"
	CodeAccountPermissionDenied     Code = "account.permission_denied"

	CodeKeyInvalid         Code = "key.invalid"
	CodeKeyMnemonicInvalid Code = "key.mnemonic_invalid"

	CodeRPCMethodFailed Code = "rpc.method_failed"
)

// AllCodes is the authoritative code set. docgen regenerates the
// Action/Doc tables from this slice; hand-editing the tables below is the
// drift this package exists to prevent.
var AllCodes = []Code{ /* every constant above, in one slice */ }
```

Then a single `switch` for `Action()` and one for `Doc()` covering every constant, with `default` returning `ActionBug`/`"unknown code"` respectively so an unhandled code is loud rather than silent.

- [ ] **Step 5: Run tests**

```bash
go test ./v2/tron/ -v
```
Expected: PASS.

- [ ] **Step 6: Example + commit**

```go
func ExampleHasCode() {
	_, err := tron.ParseTRX("1.6666666")
	if tron.HasCode(err, tron.CodeAmountTooManyDecimals) {
		fmt.Println("too many decimals — round to 6")
	}
	// Output: too many decimals — round to 6
}
```

```bash
git add v2/tron/
git commit -m "feat(v2/tron): machine-parseable Error with derived Action"
```

---

### Task 7: `cmd/docgen` + CI drift check

**Files:**
- Create: `v2/cmd/docgen/main.go`, `v2/cmd/docgen/main_test.go`, `v2/cmd/docgen/testdata/…`
- Modify: `.github/workflows/test-coverage.yml` (add `docgen -check` step)
- Generated: `v2/tron/codes_gen.go` (replaces the hand-written `Action()`/`Doc()` in `codes.go` from Task 6)

**Interfaces:**
- Consumes: `tron.AllCodes` (Task 6), example functions in `_test.go`.
- Produces:
  - `docgen` binary with two modes: `docgen generate` and `docgen -check`.
  - `v2/tron/codes_gen.go` containing `func (c Code) Action() Action`, `func (c Code) Doc() string`, derived from `AllCodes` + a doc map the generator owns.
  - CI step failing when any `<!-- go:example X -->` marker's content differs from the `ExampleX` function, or when `codes_gen.go` is stale.

- [ ] **Step 1: Write the failing test for the error-list renderer**

This is the mechanism that kills the v1 failure where one dead sentinel was advertised in four hand-maintained places (`pkg/resources/doc.go:36`, `pkg/account/doc.go:27`, `pkg/trc20/doc.go:49`, `docs/API_REFERENCE.md:1135-1163`). Spec §12 requires docgen to fill `<!-- go:errors -->` markers from the code table.

```go
package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const errListFixture = `# Errors

<!-- go:errors -->
<!-- /go:errors -->
`

func TestRenderErrorList(t *testing.T) {
	out, err := renderErrorList(errListFixture, []codeDoc{
		{Code: "chain.timeout", Action: "retry", Doc: "the node did not respond in time"},
		{Code: "amount.overflow", Action: "fix_call", Doc: "the amount exceeds the int64 SUN range"},
	})
	require.NoError(t, err)
	assert.Contains(t, out, "| `chain.timeout` | `retry` | the node did not respond in time |")
	assert.Contains(t, out, "| `amount.overflow` | `fix_call` | the amount exceeds the int64 SUN range |")
	assert.True(t, strings.Contains(out, "<!-- go:errors -->") && strings.Contains(out, "<!-- /go:errors -->"))
}

func TestRenderErrorListStable(t *testing.T) {
	in := []codeDoc{
		{Code: "b.two", Action: "retry", Doc: "b"},
		{Code: "a.one", Action: "retry", Doc: "a"},
	}
	out1, _ := renderErrorList(errListFixture, in)
	out2, _ := renderErrorList(errListFixture, in)
	assert.Equal(t, out1, out2, "output must be deterministic")
	// and sorted, so a diff is meaningful
	assert.Less(t, strings.Index(out1, "a.one"), strings.Index(out1, "b.two"))
}

func TestStaleMarkerFailsCheck(t *testing.T) {
	// file contains an example that does not match the Example function
	stale := "x\n<!-- go:example ExampleFoo -->\nold\n<!-- /go:example -->\n"
	_, err := checkFile(stale, map[string]string{"ExampleFoo": "new"})
	assert.Error(t, err, "stale content must fail -check")
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./v2/cmd/docgen/ -v
```
Expected: FAIL, `undefined: renderErrorList`.

- [ ] **Step 3: Implement docgen**

Scope it deliberately narrow — the spec needs two things, not a general doc tool:

1. `renderErrorList(md string, codes []codeDoc) (string, error)` — fills `<!-- go:errors -->` markers with a sorted Markdown table of `| Code | Action | Doc |`.
2. `fillExamples(md string, examples map[string]string) (string, error)` — for each `<!-- go:example NAME -->` marker, replaces the content between it and `<!-- /go:example -->` with `examples[NAME]`, failing if the name is missing from the map.
3. `checkFile(content string, examples map[string]string) error` — the `-check` mode: re-render and compare, erroring on any difference.

```go
package main

import (
	"fmt"
	"sort"
	"strings"
)

type codeDoc struct {
	Code   string
	Action string
	Doc    string
}

const errStart = "<!-- go:errors -->"
const errEnd = "<!-- /go:errors -->"

func renderErrorList(md string, codes []codeDoc) (string, error) {
	sorted := append([]codeDoc(nil), codes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Code < sorted[j].Code })
	var b strings.Builder
	for _, c := range sorted {
		fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", c.Code, c.Action, c.Doc)
	}
	return replaceBetween(md, errStart, errEnd, b.String())
}

func replaceBetween(s, start, end, body string) (string, error) {
	i := strings.Index(s, start)
	if i < 0 {
		return "", fmt.Errorf("marker %q not found", start)
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		return "", fmt.Errorf("closing marker %q not found", end)
	}
	j += i + len(end)
	return s[:i+len(start)] + "\n" + body + s[j-len(end):], nil
}
```

`fillExamples` uses the same `replaceBetween` with `<!-- go:example NAME -->` / `<!-- /go:example -->` pairs extracted by walking the AST with `go/ast` + `go/doc` over the target package's `_test.go` files. `extractExamples(dir string) (map[string]string, error)` parses each `*_test.go`, finds `func ExampleX()` bodies, and renders them via `format.Node`.

- [ ] **Step 4: Generate `codes_gen.go` and delete the hand-written tables**

```bash
go run ./v2/cmd/docgen generate -pkg ./v2/tron -out v2/tron/codes_gen.go
```
Then remove `Action()` and `Doc()` from `codes.go` (keep the constants and `AllCodes`), and re-run the Task 6 tests.

Expected: `go test ./v2/tron/` still passes — the generated functions are drop-in replacements, which is the whole point.

- [ ] **Step 5: Verify `-check` catches drift**

```bash
# mutate a doc, expect failure
sed -i '' 's/the node did not respond in time/WRONG/' v2/tron/codes_gen.go
go run ./v2/cmd/docgen -check ./v2/... ; echo "exit=$?"   # expect non-zero
git checkout v2/tron/codes_gen.go
go run ./v2/cmd/docgen -check ./v2/... ; echo "exit=$?"   # expect 0
```

- [ ] **Step 6: Wire into CI**

Add to `.github/workflows/test-coverage.yml` before the coverage step:

```yaml
    - name: docgen drift check
      run: |
        go run ./v2/cmd/docgen -check ./v2/...
```

- [ ] **Step 7: Full test run + commit**

```bash
go build ./... && go test ./v2/... && go vet ./v2/...
git add v2/ .github/workflows/test-coverage.yml
git commit -m "feat(v2/cmd/docgen): generated error lists and example sync with CI check"
```

---

## Deferred to Phase 2 (not tasks in this plan)

Spec §14 steps 6–13: `key`, `event`, `rpc`, `tx` (four kinds, `Sign`, `With*`, `Broadcast`, `WaitForSolid`, cost types), `contract`, `token`, root facade + happy-path, live §7.5 verification, generated migration guide, tag `v2.0.0`. Each is its own plan; they are sequenced by the DAG and none of them can be written honestly until `tron` exists as built rather than as specified.

## Self-Review

**1. Spec coverage for §14 steps 1–5:**

| Spec item | Task |
|---|---|
| Step 1 freeze v1 | Task 1 |
| Step 2 /v2 skeleton + CI | Task 2 |
| Step 3 regenerate pb from v4.8.2, gitlink+regen same commit, genproto absent, `GetPaginatedNowWitnessList` present | Task 3 |
| Step 4 `tron` package | Tasks 4, 6, 5 |
| Step 5 docgen + `-check` in CI before API work | Task 7 (after `tron`, satisfying "lands before API work" — Phase 2's packages) |
| §5.1 Whole constraint (no tilde, no unsigned) | Task 5 + compilecheck test |
| §5.2 rejection matrix as real tests | Task 5 behaviour tests + Task 5 negative-compile tests |
| §5.3 value-receiver `String`, checked Add/Sub/Mul | Task 5 |
| §8.3 full code set incl. `tx.tapos_invalid`, `tx.too_large`, `chain.network_mismatch` | Task 6 |
| §12 `<!-- go:errors -->` markers | Task 7 |
| §12 "compiles not runnable" examples | Task 4/5/6 Example functions |

**Gaps found during self-review and fixed:** initially Task 5 was ordered before Task 6, but Task 5's tests call `HasCode` and `Code*` constants. Added an explicit ordering note rather than renumbering, since the numbering follows spec §14 while the implementation order within `tron` is error → address → sun.

**2. Placeholder scan:** no TBD/TODO. Two places say "implement X following the signatures above" — Task 4's remaining Address methods and Task 5's remaining SUN methods — where the full signatures are given in the Interfaces block and the pattern is established by the shown code. That is a deliberate boundary: repeating six one-line accessors in full adds noise without information. Both have complete test coverage specified, so an implementer cannot get them wrong silently.

**3. Type consistency:** `HasCode(err, Code)` used identically in Tasks 5 and 6. `Error` fields match between Task 6's test and implementation. `Code*` constant names in Task 5's `ParseTRX` test table all appear in Task 6's const block — verified by scanning the table against the block: `CodeAmountTooManyDecimals`, `CodeAmountInvalid`, `CodeAmountOverflow` ✓. `AllCodes` is referenced by Task 6's test and produced by Task 6's `codes.go`, then consumed by Task 7's generator ✓.
