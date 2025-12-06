package service

import (
	"context"
	"log"

	"github.com/yourname/transport/notification/internal/ports"
)

type NotificationIssuedProcessor struct{}

func (p NotificationIssuedProcessor) Process(ctx context.Context, msg ports.Message[ports.NotificationIssued]) error {
	log.Printf("Notification to %s via %s: %s (event=%s at %s)",
		msg.Value.RecipientID,
		msg.Value.Channel,
		msg.Value.Message,
		msg.Value.EventType,
		msg.Value.Timestamp)
	return nil // return error to trigger nack/DLQ path
}
