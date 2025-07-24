package client

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
)

// ConnPool manages a pool of gRPC client connections.
type ConnPool struct {
	mu          sync.Mutex
	conns       chan *grpc.ClientConn
	factory     func(ctx context.Context) (*grpc.ClientConn, error)
	initialSize int
}

// NewConnPool creates a new connection pool.
func NewConnPool(factory func(ctx context.Context) (*grpc.ClientConn, error), initialSize int, capacity int) (*ConnPool, error) {
	if initialSize < 0 || capacity <= 0 || initialSize > capacity {
		return nil, fmt.Errorf("invalid pool configuration")
	}

	p := &ConnPool{
		conns:       make(chan *grpc.ClientConn, capacity),
		factory:     factory,
		initialSize: initialSize,
	}

	// Don't create initial connections - let them be created on demand
	// This allows the pool to be created even if the target is not available

	return p, nil
}

// Get retrieves a connection from the pool. If no connection is available,
// it will try to create a new one if the pool has not reached its capacity.
func (p *ConnPool) Get(ctx context.Context) (*grpc.ClientConn, error) {
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
				return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
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

// Put returns a connection to the pool.
func (p *ConnPool) Put(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	select {
	case p.conns <- conn:
	default:
		// Pool is full, close the connection
		conn.Close()
	}
}

// Close closes all connections in the pool.
func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.conns)
	for conn := range p.conns {
		conn.Close()
	}
}
