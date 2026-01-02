package main

import (
	"log"

	"github.com/yourname/transport/vehicle/configs"
	"github.com/yourname/transport/vehicle/internal/grpcserver"
	"github.com/yourname/transport/vehicle/internal/service"
)

func main() {
	cfg, err := configs.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	vs := service.NewService()
	grpcServer := grpcserver.NewGRPCServer(cfg.Server, vs)
	grpcserver.Run(grpcServer, cfg.Server.Port)
}
