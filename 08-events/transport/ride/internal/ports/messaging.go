package ports

import "context"

// Encoder/Decoder isolate schema mechanics (Avro via hamba, etc.).
type Encoder[T any] func(T) ([]byte, error)
type Decoder[T any] func([]byte) (T, error)

// Message carries decoded payload + minimal metadata.
type Message[T any] struct {
	Key      string
	Value    T
	Attempt  int // redelivery count (if available)
	Metadata map[string]string
	Ack      func() error // ack on success
	Nack     func() error // request redelivery
}

// Business logic hook. Your app implements this per event.
type Processor[T any] interface {
	Process(ctx context.Context, msg Message[T]) error
}

// Outbound port for publishing events (incl. DLQ).
type EventProducer[T any] interface {
	Send(ctx context.Context, key string, value T, props map[string]string) error
}

// Inbound port: the consumer owns receive/ack/nack plumbing and delegates to Processor.
type EventConsumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
