package pulsar

import (
	"context"
	"fmt"

	"github.com/yourname/transport/ride/internal/ports"
)

// Abstract a minimal "send" so this stays stdlib-only here.
// Wire it to the real Pulsar client in your composition root.
type rawSender func(ctx context.Context, topic string, key string, payload []byte, props map[string]string) error

type Producer[T any] struct {
	topic   string
	encode  ports.Encoder[T]
	sendRaw rawSender
}

type ProducerConfig[T any] struct {
	Topic  string
	Encode ports.Encoder[T]
	Send   rawSender
}

func NewProducer[T any](cfg ProducerConfig[T]) *Producer[T] {
	return &Producer[T]{topic: cfg.Topic, encode: cfg.Encode, sendRaw: cfg.Send}
}

func (p *Producer[T]) Send(ctx context.Context, key string, value T, props map[string]string) error {
	b, err := p.encode(value)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return p.sendRaw(ctx, p.topic, key, b, props)
}
