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
	Nack     func()       // request redelivery
}

// Business logic hook. Your app implements this per event.
type Processor[T any] interface {
	Process(ctx context.Context, msg Message[T]) error
}

// Inbound port: the consumer owns receive/ack/nack plumbing and delegates to Processor.
type EventConsumer[T any] interface {
	// Start registers the Processor and begins consuming until ctx is done or a fatal error occurs.
	Start(ctx context.Context, processor Processor[T]) error
	Stop(ctx context.Context) error
}
