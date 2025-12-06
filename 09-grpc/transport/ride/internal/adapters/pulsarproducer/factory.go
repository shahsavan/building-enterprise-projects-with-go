package pulsarproducer

import (
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"

	avroschemas "github.com/yourname/transport/ride/avro_schemas"
	"github.com/yourname/transport/ride/internal/ports"
)

func NewAssignmentCreatedProducer(client pulsar.Client) (*Producer[ports.AssignmentCreated], error) {
	schema := pulsar.NewAvroSchema(string(avroschemas.Assignment), nil)

	prod, err := NewProducer(ProducerConfig[ports.AssignmentCreated]{
		Client: client,
		Topic:  "assignments",
		Schema: schema,
	})
	if err != nil {
		return nil, fmt.Errorf("create producer: %w", err)
	}

	return prod, nil
}
