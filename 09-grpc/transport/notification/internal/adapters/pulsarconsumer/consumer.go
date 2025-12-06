package pulsarconsumer

import (
	"context"
	"errors"
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/yourname/transport/notification/internal/ports"
)

// Consumer wraps a Pulsar consumer and delegates message handling to a Processor.
type Consumer[T any] struct {
	consumer pulsar.Consumer
	decoder  ports.Decoder[T]
}

// ConsumerConfig holds the options needed to construct a Consumer.
type ConsumerConfig[T any] struct {
	Client           pulsar.Client
	Topic            string
	Subscription     string
	SubscriptionType pulsar.SubscriptionType // defaults to Shared if zero value
	Schema           pulsar.Schema           // optional; required if no Decoder is provided
	Decoder          ports.Decoder[T]        // optional; used when Schema is not provided
	Name             string                  // optional consumer name
}

// NewConsumer creates a Pulsar consumer for a topic/subscription pair.
func NewConsumer[T any](cfg ConsumerConfig[T]) (*Consumer[T], error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("pulsarconsumer: client is nil")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("pulsarconsumer: topic is required")
	}
	if cfg.Subscription == "" {
		return nil, fmt.Errorf("pulsarconsumer: subscription is required")
	}
	if cfg.Schema == nil && cfg.Decoder == nil {
		return nil, fmt.Errorf("pulsarconsumer: provide Schema or Decoder")
	}

	opts := pulsar.ConsumerOptions{
		Topic:            cfg.Topic,
		SubscriptionName: cfg.Subscription,
		Type:             cfg.SubscriptionType,
		Name:             cfg.Name,
	}
	if opts.Type == 0 {
		opts.Type = pulsar.Shared // sensible default for scaling out consumers
	}
	if cfg.Schema != nil {
		opts.Schema = cfg.Schema
	}

	cons, err := cfg.Client.Subscribe(opts)
	if err != nil {
		return nil, fmt.Errorf("pulsarconsumer: subscribe: %w", err)
	}
	return &Consumer[T]{consumer: cons, decoder: cfg.Decoder}, nil
}

// Start blocks, receiving messages and invoking the provided Processor until the
// context is canceled or a receive error occurs.
func (c *Consumer[T]) Start(ctx context.Context, processor ports.Processor[T]) error {
	if processor == nil {
		return fmt.Errorf("pulsarconsumer: processor is nil")
	}
	if c.consumer == nil {
		return fmt.Errorf("pulsarconsumer: consumer is not initialized")
	}

	for {
		msg, err := c.consumer.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("pulsarconsumer: receive: %w", err)
		}

		value, err := c.decode(msg)
		if err != nil {
			// Decoding failed; request redelivery or route to DLQ policy.
			c.consumer.Nack(msg)
			continue
		}

		wrapped := ports.Message[T]{
			Key:      msg.Key(),
			Value:    value,
			Attempt:  int(msg.RedeliveryCount()),
			Metadata: msg.Properties(),
			Ack: func() error {
				return c.consumer.Ack(msg)
			},
			Nack: func() {
				c.consumer.Nack(msg)
			},
		}

		if err := processor.Process(ctx, wrapped); err != nil {
			wrapped.Nack()
			continue
		}
		wrapped.Ack()
	}
}

// Stop closes the underlying Pulsar consumer.
func (c *Consumer[T]) Stop(context.Context) error {
	if c.consumer != nil {
		c.consumer.Close()
	}
	return nil
}

func (c *Consumer[T]) decode(msg pulsar.Message) (T, error) {
	var zero T

	if c.decoder != nil {
		return c.decoder(msg.Payload())
	}

	var value T
	if err := msg.GetSchemaValue(&value); err != nil {
		return zero, fmt.Errorf("pulsarconsumer: decode schema value: %w", err)
	}
	return value, nil
}

var _ ports.EventConsumer[any] = (*Consumer[any])(nil)
