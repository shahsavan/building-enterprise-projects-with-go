package grpcclient

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcClient struct {
	target   string
	conn     *grpc.ClientConn
	dialOpts []grpc.DialOption

	dialing chan struct{}
	dialErr error

	mu sync.Mutex
}

func NewGrpcClient(target string, opts ...grpc.DialOption) *GrpcClient {
	return &GrpcClient{
		target:   normalizeTarget(target),
		dialOpts: opts,
	}
}

// Dial establishes the connection using retry-friendly defaults and waits until the channel is ready.
// Concurrent callers share a single in-flight dial; individual contexts can time out, but the shared dial continues.
// Swap insecure.NewCredentials() with real TLS credentials in production.
func (c *GrpcClient) Dial(ctx context.Context) (*grpc.ClientConn, error) {
	c.mu.Lock()
	if c.conn != nil {
		conn := c.conn
		c.mu.Unlock()
		return conn, nil
	}

	// Another goroutine is dialing; wait for it to finish or ctx to expire.
	if c.dialing != nil {
		return c.waitForDial(ctx)
	}

	// Start a new dial attempt.
	wait := make(chan struct{})
	c.dialing = wait
	target := c.target
	c.mu.Unlock()

	baseOpts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  200 * time.Millisecond,
				Multiplier: 1.6,
				MaxDelay:   5 * time.Second,
			},
			MinConnectTimeout: 2 * time.Second,
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()), // replace with TLS creds in prod
	}

	opts := append(baseOpts, c.dialOpts...)

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		err = fmt.Errorf("grpc dial %q: %w", target, err)
	} else if errReady := c.waitForReady(ctx, conn); errReady != nil {
		_ = conn.Close()
		conn = nil
		err = fmt.Errorf("grpc wait ready %q: %w", target, errReady)
	}

	c.mu.Lock()
	if conn != nil {
		c.conn = conn
		c.dialErr = nil
	} else {
		c.dialErr = err
	}
	// Close the waiter channel while holding the lock to ensure visibility of conn/dialErr updates.
	close(wait)
	c.dialing = nil
	c.mu.Unlock()

	if conn != nil {
		return conn, nil
	}
	return nil, err
}

// waitForDial blocks until the in-flight dial finishes or ctx expires, then returns the dial result.
func (c *GrpcClient) waitForDial(ctx context.Context) (*grpc.ClientConn, error) {
	wait := c.dialing
	c.mu.Unlock()
	select {
	case <-wait:
		c.mu.Lock()
		conn, err := c.conn, c.dialErr
		c.mu.Unlock()
		if conn != nil {
			return conn, nil
		}
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close tears down the underlying connection.
// Close may block briefly if another goroutine is currently dialing.
func (c *GrpcClient) Close() error {
	// If a dial is in flight, wait for it to finish so Dial cannot publish
	// a live connection after Close returns.
	c.mu.Lock()
	wait := c.dialing
	c.mu.Unlock()

	if wait != nil {
		<-wait
	}

	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.dialErr = nil
	c.mu.Unlock()

	if conn == nil {
		return nil
	}
	return conn.Close()
}

// waitForReady blocks until the connection reaches Ready or ctx is done.
func (c *GrpcClient) waitForReady(ctx context.Context, conn *grpc.ClientConn) error {
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !conn.WaitForStateChange(ctx, state) {
			// ctx cancelled or deadline exceeded
			return ctx.Err()
		}
	}
}

func normalizeTarget(target string) string {
	if strings.Contains(target, "://") {
		return target
	}
	// Use passthrough to keep custom dialers working with raw endpoints (e.g., bufconn).
	return "passthrough:///" + target
}
