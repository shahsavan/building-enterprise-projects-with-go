package pulsar_connector

import (
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"

	avroschemas "github.com/yourname/transport/notification/avro_schemas"
	"github.com/yourname/transport/notification/internal/ports"
	"github.com/yourname/transport/ride/configs"
)

func NewPulsarClient(cfg configs.PulsarConfig) (pulsar.Client, error) {

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:                     cfg.URL,
		OperationTimeout:        cfg.OperationTimeout,
		ConnectionTimeout:       cfg.ConnectionTimeout,
		ConnectionMaxIdleTime:   cfg.ConnectionMaxIdleTime,
		KeepAliveInterval:       cfg.KeepAliveInterval,
		MaxConnectionsPerBroker: cfg.MaxConnectionsPerBroker,
		MemoryLimitBytes:        cfg.MemoryLimitBytes,
		MetricsCardinality:      pulsar.MetricsCardinalityTopic,
	})
	if err != nil {
		return nil, fmt.Errorf("error in instantiate Pulsar: %w", err)
	}
	return client, nil
}

func NewNotificationConsumer(client pulsar.Client, subscriptionName string) (*Consumer[ports.NotificationIssued], error) {

	schema := pulsar.NewAvroSchema(string(avroschemas.Notification), nil)
	cfg := configs.PulsarConsumerConfig{
		Topic:            "notifications",
		SubscriptionName: subscriptionName,
	}
	var decoder ports.Decoder[ports.NotificationIssued]

	consumer, err := NewConsumer(client, schema, decoder, cfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}

	return consumer, nil
}
