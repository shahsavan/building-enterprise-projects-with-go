package grpcserver

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourname/transport/vehicle/configs"
	vehiclepb "github.com/yourname/transport/vehicle/internal/grpc"
	"google.golang.org/grpc"
)

// Constructor: inject config, service, and logger.
func NewGRPCServer(cfg configs.ServerConfig, service vehiclepb.VehicleServiceServer) *grpc.Server {
	// Prepare TLS credentials if enabled (see TLS section)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(LoggingUnaryInterceptor),
		grpc.StreamInterceptor(LoggingStreamInterceptor),
		grpc.ConnectionTimeout(time.Duration(cfg.ConnectionTimeoutSec) * time.Second),
	}

	// Create the gRPC server with all options
	grpcSrv := grpc.NewServer(opts...)
	// Register the service implementation
	vehiclepb.RegisterVehicleServiceServer(grpcSrv, service)
	// (Optionally) register reflection or health servers here

	return grpcSrv
}

// Run constructs the server and blocks serving on the configured port.
func Run(grpcServer *grpc.Server, port int) error {
	// --- gRPC Server ---
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", port, err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("failed to serve gRPC: %w", err)
		}
	}()

	// Shutdown hook (runs on SIGINT/SIGTERM)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		grpcServer.GracefulStop() // waits for in-flight RPCs to finish
		return nil
	case err := <-errCh:
		return err
	}
}
