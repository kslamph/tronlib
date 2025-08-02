// Package client provides core infrastructure for gRPC client management
package client

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
)

// connPool manages a pool of gRPC client connections.
// connPool manages a pool of gRPC client connections.
// For testing purposes, GetFunc can be overridden to mock connection behavior.
type connPool struct {
	mu          sync.Mutex
	conns       chan *grpc.ClientConn
	factory     func(ctx context.Context) (*grpc.ClientConn, error)
	initialSize int

	// For testing only: A function to override the Get method's behavior.
	getFunc func(ctx context.Context) (*grpc.ClientConn, error)
}

// newConnPool creates a new connection pool.
func newConnPool(factory func(ctx context.Context) (*grpc.ClientConn, error), initialSize int, capacity int) (*connPool, error) {
	if initialSize < 0 || capacity <= 0 || initialSize > capacity {
		return nil, fmt.Errorf("invalid pool configuration")
	}

	p := &connPool{
		conns:       make(chan *grpc.ClientConn, capacity),
		factory:     factory,
		initialSize: initialSize,
	}

	// Don't create initial connections - let them be created on demand
	// This allows the pool to be created even if the target is not available

	return p, nil
}

// get retrieves a connection from the pool. If no connection is available,
// it will try to create a new one if the pool has not reached its capacity.
func (p *connPool) get(ctx context.Context) (*grpc.ClientConn, error) {
	// If a mock GetFunc is provided, use it
	if p.getFunc != nil {
		return p.getFunc(ctx)
	}

	select {
	case conn := <-p.conns:
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		p.mu.Lock()
		defer p.mu.Unlock()

		if len(p.conns) < cap(p.conns) {
			conn, err := p.factory(ctx)
			if err != nil {
				return nil, fmt.Errorf("connection failed: %v", err)
			}
			return conn, nil
		}
		// Wait for a connection to be returned to the pool
		select {
		case conn := <-p.conns:
			return conn, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// put returns a connection to the pool.
func (p *connPool) put(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	select {
	case p.conns <- conn:
	default:
		// Pool is full, close the connection and ignore close error
		_ = conn.Close()
	}
}

// close closes all connections in the pool.
func (p *connPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.conns)
	for conn := range p.conns {
		_ = conn.Close()
	}
}
