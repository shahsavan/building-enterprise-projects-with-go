package pulsarconsumer

import (
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"

	avroschemas "github.com/yourname/transport/notification/avro_schemas"
	"github.com/yourname/transport/notification/internal/ports"
)

func NewNotificationConsumer(client pulsar.Client, subscriptionName string) (*Consumer[ports.NotificationIssued], error) {

	schema := pulsar.NewAvroSchema(string(avroschemas.Notification), nil)

	consumer, err := NewConsumer(ConsumerConfig[ports.NotificationIssued]{
		Client:       client,
		Topic:        "notifications",
		Subscription: subscriptionName,
		Schema:       schema,
	})

	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}

	return consumer, nil
}
