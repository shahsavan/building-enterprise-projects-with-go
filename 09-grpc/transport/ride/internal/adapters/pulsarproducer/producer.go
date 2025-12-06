package pulsarproducer

import (
	"context"
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/yourname/transport/ride/internal/ports"
)

// Producer is a wrapper around pulsar.Producer that supports custom encoding.
type Producer[T any] struct {
	topic    string
	encoder  ports.Encoder[T]
	producer pulsar.Producer
}

// ProducerConfig holds settings for creating a Producer.
type ProducerConfig[T any] struct {
	Client  pulsar.Client    // Pulsar client (managed by caller)
	Topic   string           // Pulsar topic name (required)
	Schema  pulsar.Schema    // Optional Pulsar schema (e.g. Avro/JSON schema)
	Encoder ports.Encoder[T] // Optional encoder for payloads
}

// NewProducer creates a new Producer. The Pulsar client and resources
// are managed by the caller (not closed here).  An optional Schema may
// be provided (e.g. via pulsar.NewAvroSchema) and an encoder function
// for custom serialization (e.g. using an Avro library).
func NewProducer[T any](cfg ProducerConfig[T]) (*Producer[T], error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("pulsarproducer: client is nil")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("pulsarproducer: topic is required")
	}
	// Prepare Pulsar producer options.
	opts := pulsar.ProducerOptions{Topic: cfg.Topic}
	if cfg.Schema != nil {
		opts.Schema = cfg.Schema // register schema if provided
	}
	// Create the underlying Pulsar producer.
	prod, err := cfg.Client.CreateProducer(opts)
	if err != nil {
		return nil, fmt.Errorf("pulsarproducer: could not create producer: %w", err)
	}
	return &Producer[T]{producer: prod, topic: cfg.Topic, encoder: cfg.Encoder}, nil
}

// Send publishes a message with the given payload. If an encoder was
// provided, it is used to serialize the payload to bytes.  Otherwise,
// the payload is sent directly as the message Value (which requires a
// matching Pulsar schema).  Returns the Pulsar MessageID or an error.
func (p *Producer[T]) Send(ctx context.Context, payload T) (string, error) {
	var msg pulsar.ProducerMessage
	if p.encoder != nil {
		data, err := p.encoder(payload)
		if err != nil {
			return "", fmt.Errorf("pulsarproducer: encoding failed: %w", err)
		}
		msg.Payload = data
	} else {
		// Use Pulsar schema to encode the payload (payload must match schema).
		msg.Value = payload
	}
	// Send the message (blocking until acked or error).
	msgID, err := p.producer.Send(ctx, &msg)
	if err != nil {
		return "", fmt.Errorf("pulsarproducer: failed to send message: %w", err)
	}
	return msgID.String(), nil
}

// Close shuts down the producer and releases resources. Pending messages
// will be flushed or returned as errors according to Pulsar settings.
func (p *Producer[T]) Close() {
	p.producer.Close()
}

var _ ports.EventProducer[any] = (*Producer[any])(nil)
