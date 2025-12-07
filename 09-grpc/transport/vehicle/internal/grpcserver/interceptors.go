package grpcserver

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func LoggingUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	res, err := handler(ctx, req)
	duration := time.Since(start)

	log.Printf("method=%s duration=%s error=%v", info.FullMethod, duration, err)
	return res, err
}

func LoggingStreamInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	log.Printf("stream=%s started", info.FullMethod)
	err := handler(srv, ss)
	log.Printf("stream=%s ended with error=%v", info.FullMethod, err)
	return err
}
