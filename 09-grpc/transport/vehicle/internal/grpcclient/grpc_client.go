package grpcclient

import (
	"context"
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

	once    sync.Once
	dialErr error

	mu sync.RWMutex
}

func NewGrpcClient(target string, opts ...grpc.DialOption) *GrpcClient {
	return &GrpcClient{
		target:   normalizeTarget(target),
		dialOpts: opts,
	}
}

// Dial establishes the connection using retry-friendly defaults and waits until the channel is ready.
// Swap insecure.NewCredentials() with real TLS credentials in production.
func (c *GrpcClient) Dial(ctx context.Context) (*grpc.ClientConn, error) {
	c.mu.RLock()
	conn := c.conn
	dialErr := c.dialErr
	c.mu.RUnlock()
	if conn != nil {
		return conn, nil
	}

	if dialErr != nil {
		return nil, dialErr
	}

	c.once.Do(func() {
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

		conn, err := grpc.NewClient(c.target, opts...)
		if err != nil {
			c.setDialErr(err)
			return
		}

		if err := c.waitForReady(ctx, conn); err != nil {
			_ = conn.Close()
			c.setDialErr(err)
			return
		}

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
	})

	c.mu.RLock()
	conn = c.conn
	dialErr = c.dialErr
	c.mu.RUnlock()

	if conn == nil {
		return nil, dialErr
	}
	return conn, nil
}

// Close tears down the underlying connection.
func (c *GrpcClient) Close() error {
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
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

func (c *GrpcClient) setDialErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dialErr = err
}

func normalizeTarget(target string) string {
	if strings.Contains(target, "://") {
		return target
	}
	// Use passthrough to keep custom dialers working with raw endpoints (e.g., bufconn).
	return "passthrough:///" + target
}
