package pulsar

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yourname/transport/ride/internal/ports"
)

// Minimal raw message & pull to stay stdlib-only.
type rawMsg struct {
	Key     string
	Payload []byte
	Props   map[string]string
	Attempt int
	Ack     func() error
	Nack    func() error
}

type rawPull func(ctx context.Context) (rawMsg, error) // blocks until a message or ctx done

type Consumer[T any] struct {
	topic        string
	subscription string
	decode       ports.Decoder[T]
	pull         rawPull

	processor    ports.Processor[T]
	dlq          ports.EventProducer[T] // optional
	maxRedeliver int
	parallelism  int

	stopOnce sync.Once
	stopCh   chan struct{}
}

type ConsumerConfig[T any] struct {
	Topic        string
	Subscription string
	Decode       ports.Decoder[T]
	Pull         rawPull

	Processor    ports.Processor[T]
	DLQ          ports.EventProducer[T] // nil to disable DLQ produce
	MaxRedeliver int                    // e.g., 5
	Parallelism  int                    // e.g., number of workers
}

func NewConsumer[T any](cfg ConsumerConfig[T]) *Consumer[T] {
	if cfg.Parallelism <= 0 {
		cfg.Parallelism = 1
	}
	return &Consumer[T]{
		topic:        cfg.Topic,
		subscription: cfg.Subscription,
		decode:       cfg.Decode,
		pull:         cfg.Pull,
		processor:    cfg.Processor,
		dlq:          cfg.DLQ,
		maxRedeliver: cfg.MaxRedeliver,
		parallelism:  cfg.Parallelism,
		stopCh:       make(chan struct{}),
	}
}

func (c *Consumer[T]) Start(ctx context.Context) error {
	wg := new(sync.WaitGroup)
	wg.Add(c.parallelism)

	worker := func(id int) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			default:
			}

			rm, err := c.pull(ctx)
			if err != nil {
				// Treat context cancellation as a clean shutdown.
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				// Backoff a bit on unexpected errors from the broker.
				time.Sleep(200 * time.Millisecond)
				continue
			}

			val, decErr := c.decode(rm.Payload)
			msg := ports.Message[T]{
				Key:      rm.Key,
				Value:    val,
				Attempt:  rm.Attempt,
				Metadata: rm.Props,
				Ack:      rm.Ack,
				Nack:     rm.Nack,
			}

			switch {
			case decErr != nil:
				c.handleFailure(ctx, msg, fmt.Errorf("decode: %w", decErr))
			default:
				if procErr := c.processor.Process(ctx, msg); procErr != nil {
					c.handleFailure(ctx, msg, procErr)
				} else {
					_ = rm.Ack()
				}
			}
		}
	}

	for i := 0; i < c.parallelism; i++ {
		go worker(i)
	}

	// Wait until context is cancelled.
	<-ctx.Done()
	close(c.stopCh)
	wg.Wait()
	return nil
}

func (c *Consumer[T]) Stop(ctx context.Context) error {
	c.stopOnce.Do(func() { close(c.stopCh) })
	// Real adapter would also close the underlying pulsar consumer.
	return nil
}

func (c *Consumer[T]) handleFailure(ctx context.Context, msg ports.Message[T], cause error) {
	// If we have exceeded redeliver threshold, send to DLQ (if configured).
	if c.dlq != nil && msg.Attempt >= c.maxRedeliver && c.maxRedeliver > 0 {
		_ = c.dlq.Send(ctx, msg.Key, msg.Value, map[string]string{
			"dlq.reason": cause.Error(),
			"src.topic":  c.topic,
			"sub":        c.subscription,
		})
		_ = msg.Ack() // ack original so it doesn't loop forever
		return
	}
	_ = msg.Nack() // let broker redeliver with backoff policy
}
