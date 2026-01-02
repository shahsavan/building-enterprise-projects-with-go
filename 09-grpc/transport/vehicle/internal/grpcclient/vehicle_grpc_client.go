package grpcclient

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	vehiclepb "github.com/yourname/transport/vehicle/internal/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const vehicleServiceConfig = `{
	"methodConfig": [{
	  "name": [{ "service": "vehiclepb.VehicleService" }],
	  "retryPolicy": {
		"MaxAttempts": 4,
		"InitialBackoff": "0.2s",
		"MaxBackoff": "5s",
		"BackoffMultiplier": 1.6,
		"RetryableStatusCodes": ["UNAVAILABLE", "RESOURCE_EXHAUSTED"]
	  }
	}]
  }`

var ErrNotDialed = errors.New("grpc client is nil")

// VehicleGrpcClient wraps the vehicle client connection, dialing settings, and resilience primitives.
type VehicleGrpcClient struct {
	base   *GrpcClient
	client vehiclepb.VehicleServiceClient
	cb     *gobreaker.CircuitBreaker
	mu     sync.RWMutex
}

// NewGrpcClient configures a client with sensible defaults; call Dial to establish the connection.
func NewVehicleGrpcClient(target string, opts ...grpc.DialOption) *VehicleGrpcClient {
	// service-specific retry config is passed as a dial option to the shared client
	base := NewGrpcClient(target, append([]grpc.DialOption{
		grpc.WithDefaultServiceConfig(vehicleServiceConfig),
	}, opts...)...)

	return &VehicleGrpcClient{
		base: base,
		cb:   newCircuitBreaker(),
	}
}

// Dial establishes the shared connection and prepares the typed client.
func (c *VehicleGrpcClient) Dial(ctx context.Context) error {
	conn, err := c.base.Dial(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return nil
	}
	c.client = vehiclepb.NewVehicleServiceClient(conn)
	return nil
}

// Close tears down the underlying connection.
func (c *VehicleGrpcClient) Close() error {
	c.mu.Lock()
	c.client = nil
	c.mu.Unlock()
	return c.base.Close()
}

func (c *VehicleGrpcClient) FindAvailableVehicle(ctx context.Context, req *vehiclepb.FindRequest) (*vehiclepb.FindResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, ErrNotDialed
	}
	return c.callWithBreaker(ctx, client, req)
}

func (c *VehicleGrpcClient) callWithBreaker(ctx context.Context, client vehiclepb.VehicleServiceClient, req *vehiclepb.FindRequest) (*vehiclepb.FindResponse, error) {
	var out *vehiclepb.FindResponse
	_, err := c.cb.Execute(func() (any, error) {
		resp, err := client.FindAvailableVehicle(ctx, req)
		if err != nil {
			// map retryable codes to errors that count toward the breaker
			if st, ok := status.FromError(err); ok {
				switch st.Code() {
				case codes.Unavailable, codes.ResourceExhausted, codes.DeadlineExceeded:
					return nil, err
				}
			}
			// non-retryable errors propagate but won't trip due to IsSuccessful.
			return nil, err
		}
		out = resp
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func newCircuitBreaker() *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "vehicle-client",
		MaxRequests: 1,                // half-open probes
		Interval:    30 * time.Second, // reset counts window
		Timeout:     30 * time.Second, // open -> half-open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 10 {
				return false
			}
			failRate := float64(counts.TotalFailures) / float64(counts.Requests)
			return failRate >= 0.5
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			if st, ok := status.FromError(err); ok {
				switch st.Code() {
				// caller bugs or permanent errors: don't trip the breaker
				case codes.InvalidArgument, codes.Unauthenticated, codes.PermissionDenied:
					return true
				}
			}
			return false
		},
	})
}
