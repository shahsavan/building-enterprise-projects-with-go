package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/yourname/transport/vehicle/configs"
	vehiclepb "github.com/yourname/transport/vehicle/internal/grpc"
	"github.com/yourname/transport/vehicle/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// bufSize keeps the in-memory listener large enough for typical test payloads.
const bufSize = 1024 * 1024

func startBufconnServer(t *testing.T, svc vehiclepb.VehicleServiceServer) (*grpc.Server, *bufconn.Listener) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := NewGRPCServer(configs.ServerConfig{ConnectionTimeoutSec: 5}, svc)

	go func() {
		if err := srv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("server exited unexpectedly: %v", err)
		}
	}()

	t.Cleanup(func() {
		srv.Stop()
		lis.Close()
	})
	return srv, lis
}

func dialBufconn(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
	})
	return conn
}

func TestNewGRPCServerServesRequests(t *testing.T) {
	_, lis := startBufconnServer(t, service.NewService())
	client := vehiclepb.NewVehicleServiceClient(dialBufconn(t, lis))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.FindAvailableVehicle(ctx, &vehiclepb.FindRequest{RouteId: "route-42"})
	if err != nil {
		t.Fatalf("FindAvailableVehicle RPC failed: %v", err)
	}
	if resp.GetVehicleId() != "bus-123" || resp.GetStatus() != "available" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestNewGRPCServerHandlesConcurrentRequests(t *testing.T) {
	_, lis := startBufconnServer(t, service.NewService())
	client := vehiclepb.NewVehicleServiceClient(dialBufconn(t, lis))

	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(route string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			resp, err := client.FindAvailableVehicle(ctx, &vehiclepb.FindRequest{RouteId: route})
			if err != nil {
				errCh <- err
				return
			}
			if resp.GetVehicleId() == "" {
				errCh <- fmt.Errorf("empty vehicle id")
			}
		}(fmt.Sprintf("route-%d", i))
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent request failed: %v", err)
	}
}

func TestRunServesAndStopsOnSignal(t *testing.T) {
	port := freePort(t)
	grpcServer := NewGRPCServer(configs.ServerConfig{ConnectionTimeoutSec: 5}, service.NewService())

	done := make(chan error, 1)
	go func() { done <- Run(grpcServer, port) }()

	conn := dialTCP(t, port)
	client := vehiclepb.NewVehicleServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := client.FindAvailableVehicle(ctx, &vehiclepb.FindRequest{RouteId: "route-signal"}); err != nil {
		t.Fatalf("RPC before shutdown failed: %v", err)
	}

	self, _ := os.FindProcess(os.Getpid())
	_ = self.Signal(os.Interrupt)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after interrupt")
	}
}

func TestRunGracefulShutdownWaitsForInflight(t *testing.T) {
	port := freePort(t)
	svc := &slowService{delay: 200 * time.Millisecond, called: make(chan struct{})}
	grpcServer := NewGRPCServer(configs.ServerConfig{ConnectionTimeoutSec: 5}, svc)

	done := make(chan error, 1)
	go func() { done <- Run(grpcServer, port) }()

	conn := dialTCP(t, port)
	client := vehiclepb.NewVehicleServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var rpcErr error
	var rpcDone sync.WaitGroup
	rpcDone.Add(1)
	go func() {
		defer rpcDone.Done()
		_, rpcErr = client.FindAvailableVehicle(ctx, &vehiclepb.FindRequest{RouteId: "slow"})
	}()

	<-svc.called

	self, _ := os.FindProcess(os.Getpid())
	_ = self.Signal(os.Interrupt)

	rpcDone.Wait()
	if rpcErr != nil {
		t.Fatalf("RPC failed during graceful shutdown: %v", rpcErr)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after graceful shutdown")
	}
}

func dialTCP(t *testing.T, port int) *grpc.ClientConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to dial tcp server: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
	})
	return conn
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to reserve port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

type slowService struct {
	vehiclepb.UnimplementedVehicleServiceServer
	delay  time.Duration
	called chan struct{}
	once   sync.Once
}

func (s *slowService) FindAvailableVehicle(ctx context.Context, req *vehiclepb.FindRequest) (*vehiclepb.FindResponse, error) {
	if s.called == nil {
		s.called = make(chan struct{})
	}
	s.once.Do(func() {
		close(s.called)
	})
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &vehiclepb.FindResponse{
		VehicleId: "bus-123",
		Status:    "available",
	}, nil
}
