package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/yourname/transport/ride/configs"
	"github.com/yourname/transport/ride/internal/adapters/httpserver"
	"github.com/yourname/transport/ride/internal/adapters/repository"
)

func main() {
	cfg, err := configs.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	safe := *cfg
	safe.Database.Password = "<redacted>"
	log.Printf("Loaded config: %+v", safe)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Pool configuration (database/sql pool lives inside *sql.DB)
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.Database.ConnMaxIdleTime) * time.Second)

	assignmentRepo := repository.NewSQLAssignmentRepository(db)

	// ctx := context.Background()
	// client, err := pulsar.NewClient(pulsar.ClientOptions{URL: cfg.Pulsar.URL})
	// if err != nil {
	// 	log.Fatalf("failed to open pulsar connection: %w", err)
	// }
	// defer client.Close()

	// producer, err := pulsarproducer.NewAssignmentCreatedProducer(client)
	// if err != nil {
	// 	log.Fatalf("failed to create pulsar producer: %v", err)
	// }
	// defer producer.Close()

	// driverID := "driver-123"
	// m := ports.AssignmentCreated{
	// 	AssignmentID: "assign-001",
	// 	VehicleID:    "vehicle-456",
	// 	RouteID:      "route-789",
	// 	Timestamp:    "2024-01-01T00:00:00Z",
	// 	DriverID:     &driverID,
	// }
	// if _, err := producer.Send(ctx, m); err != nil {
	// 	log.Printf("failed to publish assignment: %v", err)
	// }

	httpserver.Run(cfg.Server, assignmentRepo)
}
