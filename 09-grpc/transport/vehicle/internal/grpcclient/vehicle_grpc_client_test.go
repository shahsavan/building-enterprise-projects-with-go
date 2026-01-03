package grpcclient

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yourname/transport/vehicle/configs"
	vehiclepb "github.com/yourname/transport/vehicle/internal/grpc"
	"github.com/yourname/transport/vehicle/internal/grpcserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const testBufSize = 1024 * 1024

func startBufconnServer(t *testing.T, svc vehiclepb.VehicleServiceServer) (*grpc.Server, *bufconn.Listener) {
	t.Helper()

	lis := bufconn.Listen(testBufSize)
	srv := grpcserver.NewGRPCServer(configs.ServerConfig{ConnectionTimeoutSec: 5}, svc)

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

type flakyService struct {
	vehiclepb.UnimplementedVehicleServiceServer
	failTimes int32
	failCode  codes.Code
	calls     int32
}

func (f *flakyService) FindAvailableVehicle(ctx context.Context, req *vehiclepb.FindRequest) (*vehiclepb.FindResponse, error) {
	atomic.AddInt32(&f.calls, 1)

	if atomic.LoadInt32(&f.failTimes) > 0 {
		atomic.AddInt32(&f.failTimes, -1)
		return nil, status.Error(f.failCode, "transient failure")
	}

	return &vehiclepb.FindResponse{
		VehicleId: "bus-123",
		Status:    "available",
	}, nil
}

func dialBufnet(t *testing.T, lis *bufconn.Listener) *VehicleGrpcClient {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	client := NewVehicleGrpcClient("bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
	)
	if err := client.Dial(ctx); err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func TestDialVehicleRetriesTransientUnavailable(t *testing.T) {
	_, lis := startBufconnServer(t, &flakyService{
		failTimes: 2,
		failCode:  codes.Unavailable,
	})

	client := dialBufnet(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.FindAvailableVehicle(ctx, &vehiclepb.FindRequest{RouteId: "route-1"})
	if err != nil {
		t.Fatalf("FindAvailableVehicle failed: %v", err)
	}

	if resp.GetVehicleId() != "bus-123" || resp.GetStatus() != "available" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDialVehicleDoesNotRetryNonRetryableCode(t *testing.T) {
	flaky := &flakyService{
		failTimes: 1,
		failCode:  codes.InvalidArgument,
	}
	_, lis := startBufconnServer(t, flaky)

	client := dialBufnet(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.FindAvailableVehicle(ctx, &vehiclepb.FindRequest{RouteId: "bad"})
	if err == nil {
		t.Fatalf("expected error but got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}

	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected code %v, got %v", codes.InvalidArgument, st.Code())
	}

	if got := atomic.LoadInt32(&flaky.calls); got != 1 {
		t.Fatalf("expected 1 call, got %d", got)
	}
}
