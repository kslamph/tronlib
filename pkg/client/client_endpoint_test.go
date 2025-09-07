package client

import "testing"

func TestNewClient_InvalidEndpoint(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatalf("expected error for empty endpoint")
	}
	if _, err := NewClient("127.0.0.1:50051"); err == nil {
		t.Fatalf("expected error for missing scheme")
	}
	if _, err := NewClient("http://127.0.0.1:50051"); err == nil {
		t.Fatalf("expected error for unsupported scheme")
	}
	if _, err := NewClient("grpc://"); err == nil {
		t.Fatalf("expected error for missing host:port")
	}
}
