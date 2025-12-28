package pulsar_connector

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/yourname/transport/notification/internal/ports"
	"github.com/yourname/transport/ride/configs"
)

// Consumer wraps a Pulsar consumer and delegates message handling to a Processor.
type Consumer[T any] struct {
	consumer pulsar.Consumer
	client   pulsar.Client
	options  pulsar.ConsumerOptions
	decoder  ports.Decoder[T]
}

// NewConsumer creates a Pulsar consumer for a topic/subscription pair.
func NewConsumer[T any](client pulsar.Client, schema pulsar.Schema, decoder ports.Decoder[T], cfg configs.PulsarConsumerConfig) (*Consumer[T], error) {
	if client == nil {
		return nil, fmt.Errorf("pulsarconsumer: client is nil")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("pulsarconsumer: topic is required")
	}
	if cfg.SubscriptionName == "" {
		return nil, fmt.Errorf("pulsarconsumer: subscription is required")
	}
	if schema == nil && decoder == nil {
		return nil, fmt.Errorf("pulsarconsumer: provide Schema or Decoder")
	}

	st := parseSubscriptionType(cfg.SubscriptionType)

	var mxRecon uint = 3

	subspos, err := getSubscriptionPosition(nil)
	if err != nil {
		return nil, fmt.Errorf("error in getting SubscriptionPosition %w", err)
	}

	subsmod, err := getSubscriptionMode(nil)
	if err != nil {
		return nil, fmt.Errorf("error in getting SubscriptionMode %w", err)
	}
	opts := pulsar.ConsumerOptions{
		Topic:                cfg.Topic,
		SubscriptionName:     cfg.SubscriptionName,
		Type:                 st,
		Name:                 cfg.Name,
		ReceiverQueueSize:    cfg.ReceiverQueueSize,
		DLQ:                  &pulsar.DLQPolicy{MaxDeliveries: 10, DeadLetterTopic: "notifications-dlq"},
		NackRedeliveryDelay:  cfg.NackRedeliveryDelay,
		NackBackoffPolicy:    nil,
		MaxReconnectToBroker: &mxRecon,
	}
	if subspos > 0 {
		opts.SubscriptionInitialPosition = subspos
	}
	if subsmod > 0 {
		opts.SubscriptionMode = subsmod
	}

	if schema != nil {
		opts.Schema = schema
	}

	cons, err := client.Subscribe(opts)
	if err != nil {
		return nil, fmt.Errorf("pulsarconsumer: subscribe: %w", err)
	}
	return &Consumer[T]{
		consumer: cons,
		client:   client,
		options:  opts,
		decoder:  decoder,
	}, nil
}

func getSubscriptionPosition(pos *string) (pulsar.SubscriptionInitialPosition, error) {
	if pos == nil {
		return -1, nil
	}
	posVal := strings.ToLower(*pos)
	if posVal == "latest" {
		return pulsar.SubscriptionPositionLatest, nil
	}
	if posVal == "earliest" {
		return pulsar.SubscriptionPositionEarliest, nil
	}
	return -1, fmt.Errorf("undefigned SubscriptionPosition %s", *pos)
}

func getSubscriptionMode(mod *string) (pulsar.SubscriptionMode, error) {
	if mod == nil {
		return -1, nil
	}
	modVal := strings.ToLower(*mod)
	if modVal == "durable" {
		return pulsar.Durable, nil
	}
	if modVal == "nonDurable" {
		return pulsar.NonDurable, nil
	}
	return -1, fmt.Errorf("undefigned SubscriptionMode %s", *mod)
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
		if ctx.Err() != nil {
			return nil
		}

		msg, err := c.consumer.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			continue
		}
		c.handleMessage(ctx, processor, msg)

	}
}

// Stop closes the underlying Pulsar consumer.
func (c *Consumer[T]) Stop(context.Context) error {
	if c.consumer != nil {
		c.consumer.Close()
	}
	return nil
}

func (c *Consumer[T]) handleMessage(ctx context.Context, processor ports.Processor[T], msg pulsar.Message) {
	value, err := c.decode(msg)
	if err != nil {
		// log it
		c.consumer.Nack(msg)
		return
	}

	wrapped := ports.Message[T]{
		Key:      msg.Key(),
		Value:    value,
		Metadata: msg.Properties(),
	}

	if err := processor.Process(ctx, wrapped); err != nil {
		c.consumer.Nack(msg)
		return
	}
	if err := c.consumer.Ack(msg); err != nil {
		// log err
	}
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

// parseSubscriptionType converts a human-friendly string into the Pulsar enum.
// Empty input defaults to Shared.
func parseSubscriptionType(s string) pulsar.SubscriptionType {
	switch strings.ToLower(strings.ReplaceAll(s, "-", "")) {
	case "", "shared":
		return pulsar.Shared
	case "exclusive":
		return pulsar.Exclusive
	case "failover":
		return pulsar.Failover
	case "keyshared", "key_shared":
		return pulsar.KeyShared
	default:
		return pulsar.Exclusive
	}
}

var _ ports.EventConsumer[any] = (*Consumer[any])(nil)
