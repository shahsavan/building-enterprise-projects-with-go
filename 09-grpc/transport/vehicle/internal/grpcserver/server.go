package grpcserver

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/yourname/transport/vehicle/configs"
	vehiclepb "github.com/yourname/transport/vehicle/internal/grpc"
	"google.golang.org/grpc"
)

// gRPC runs for VehicleService.
func Run(cfg configs.ServerConfig, grpcSrv vehiclepb.VehicleServiceServer) error {
	// --- gRPC Server ---
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", cfg.Port, err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(LoggingUnaryInterceptor),
		grpc.StreamInterceptor(LoggingStreamInterceptor),
		grpc.ConnectionTimeout(time.Duration(cfg.ConnectionTimeoutSec)*time.Second),
	)
	vehiclepb.RegisterVehicleServiceServer(grpcServer, grpcSrv)

	log.Printf("gRPC server running on port %d", cfg.Port)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}
	return nil
}
