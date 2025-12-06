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

// Outbound port for publishing events (incl. DLQ).
type EventProducer[T any] interface {
	Send(ctx context.Context, value T) (string, error)
}
