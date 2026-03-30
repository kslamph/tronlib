package client

import (
	"testing"
	"time"

	"github.com/kslamph/tronlib/pkg/types"
)

func TestClient_GetNodeAddress(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	addr := c.GetNodeAddress()
	if addr == "" {
		t.Fatal("expected non-empty node address")
	}
}

func TestClient_IsConnected(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	if !c.IsConnected() {
		t.Fatal("expected client to be connected")
	}

	c.Close()

	if c.IsConnected() {
		t.Fatal("expected client to be disconnected after Close")
	}
}

func TestClient_Account(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	mgr := c.Account()
	if mgr == nil {
		t.Fatal("expected non-nil AccountManager")
	}
}

func TestClient_SmartContract(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	mgr := c.SmartContract()
	if mgr == nil {
		t.Fatal("expected non-nil SmartContract Manager")
	}
}

func TestClient_Network(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	mgr := c.Network()
	if mgr == nil {
		t.Fatal("expected non-nil NetworkManager")
	}
}

func TestClient_Resources(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	mgr := c.Resources()
	if mgr == nil {
		t.Fatal("expected non-nil ResourcesManager")
	}
}

func TestClient_TRC10(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	mgr := c.TRC10()
	if mgr == nil {
		t.Fatal("expected non-nil TRC10Manager")
	}
}

func TestClient_Voting(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	mgr := c.Voting()
	if mgr == nil {
		t.Fatal("expected non-nil VotingManager")
	}
}

func TestClient_TRC20(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	// TRC20 returns nil if NewManager fails, but we verify it doesn't panic
	addr, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	mgr := c.TRC20(addr)
	// mgr may be nil if initialization fails, but the method should not panic
	_ = mgr
}

func TestClient_ContractInstance(t *testing.T) {
	srv := &testWalletServer{}
	lis, _, cleanupSrv := newBufconnServer(t, srv)
	defer cleanupSrv()

	c, cleanupClient := newTestClientWithBufConn(t, lis, 500*time.Millisecond)
	defer cleanupClient()

	addr, _ := types.NewAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	inst, err := c.ContractInstance(addr, testERC20ABIForClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst == nil {
		t.Fatal("expected non-nil contract instance")
	}
}

const testERC20ABIForClient = `[
	{
		"constant": true,
		"inputs": [],
		"name": "name",
		"outputs": [{"name": "", "type": "string"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"type": "function"
	}
]`
