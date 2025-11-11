package service

import (
	"context"
	"log"

	"github.com/yourname/transport/ride/internal/adapters/avro"
	"github.com/yourname/transport/ride/internal/core/ports"
)

type AssignmentCreatedProcessor struct{}

func (p AssignmentCreatedProcessor) Process(ctx context.Context, msg ports.Message[avro.AssignmentCreated]) error {
	log.Printf("Assignment %s -> Vehicle %s on Route %s",
		msg.Value.AssignmentID, msg.Value.VehicleID, msg.Value.RouteID)
	return nil // return error to trigger nack/DLQ path
}
