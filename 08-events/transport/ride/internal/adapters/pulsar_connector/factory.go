package pulsar_connector

import (
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"

	avroschemas "github.com/yourname/transport/ride/avro_schemas"
	"github.com/yourname/transport/ride/configs"
	"github.com/yourname/transport/ride/internal/ports"
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
	})
	if err != nil {
		return nil, fmt.Errorf("error in instantiate Pulsar: %w", err)
	}
	return client, nil
}

func NewAssignmentCreatedProducer(client pulsar.Client, pcfg configs.PulsarProducerConfig) (*Producer[ports.AssignmentCreated], error) {
	schema := pulsar.NewAvroSchema(string(avroschemas.Assignment), nil)

	if pcfg.Topic == "" {
		pcfg.Topic = "assignments"
	}

	prod, err := NewProducer(ProducerConfig[ports.AssignmentCreated]{
		Client:        client,
		Topic:         pcfg.Topic,
		Schema:        schema,
		PulsarConfigs: pcfg,
	})
	if err != nil {
		return nil, fmt.Errorf("create producer: %w", err)
	}

	return prod, nil
}
