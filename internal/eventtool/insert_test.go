package eventtool

import "testing"

const sampleABI = `[
	{"type":"function","name":"transfer","inputs":[{"type":"address","name":"to"},{"type":"uint256","name":"v"}]},
	{"type":"event","name":"Anon","anonymous":true,"inputs":[{"type":"uint256","indexed":true,"name":"id"}]},
	{"type":"event","name":"","anonymous":false,"inputs":[]},
	{"type":"event","name":"Ping","inputs":[{"type":"uint256","indexed":false,"name":"n"}]}
]`

func TestInsertABIKeepsNamedEventsOnly(t *testing.T) {
	s := New("")
	n, err := InsertABI([]byte(sampleABI), s)
	if err != nil {
		t.Fatalf("InsertABI: %v", err)
	}
	if n != 1 || s.Len() != 1 {
		t.Fatalf("inserted %d (store %d), want 1/1 (function, anonymous, unnamed all skipped)", n, s.Len())
	}
	if got := s.Events()[0]; got.Signature != "Ping(uint256)" {
		t.Fatalf("stored %+v, want Ping(uint256)", got)
	}
	// Re-inserting the same ABI adds nothing (first-wins, insert-if-absent).
	again, err := InsertABI([]byte(sampleABI), s)
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Fatalf("re-insert added %d, want 0", again)
	}
}

func TestInsertABIAcceptsWrappedObject(t *testing.T) {
	s := New("")
	wrapped := `{"abi":` + sampleABI + `}`
	n, err := InsertABI([]byte(wrapped), s)
	if err != nil {
		t.Fatalf("InsertABI(wrapped): %v", err)
	}
	if n != 1 {
		t.Fatalf("inserted %d, want 1", n)
	}
}

func TestInsertABIRejectsGarbage(t *testing.T) {
	s := New("")
	if _, err := InsertABI([]byte("{not json"), s); err == nil {
		t.Fatal("want an error for malformed JSON")
	}
}
