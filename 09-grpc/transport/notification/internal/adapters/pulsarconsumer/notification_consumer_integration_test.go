//go:build integration_test

package pulsarconsumer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"

	avroschemas "github.com/yourname/transport/notification/avro_schemas"
	"github.com/yourname/transport/notification/internal/adapters/pulsarconsumer"
	"github.com/yourname/transport/notification/internal/ports"
	"github.com/yourname/transport/ride/test_containers"
)

func TestNotificationConsumerProcessesMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	const (
		namespace = "public/default"
		fullTopic = "persistent://public/default/notifications"
		topic     = "notifications"
		subName   = "notification-service"
	)

	pulsarEnv, err := test_containers.EnsurePulsarTopic(ctx, namespace, fullTopic, 0, nil, nil)
	if err != nil {
		t.Fatalf("pulsar setup failed: %v", err)
	}

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: fmt.Sprintf("pulsar://%s:%s", pulsarEnv.Host, pulsarEnv.Port),
	})
	if err != nil {
		t.Fatalf("failed to create pulsar client: %v", err)
	}
	t.Cleanup(func() {
		client.Close()
	})

	consumer, err := pulsarconsumer.NewNotificationConsumer(client, "notification-service")
	if err != nil {
		t.Fatalf("failed to create notification consumer: %v", err)
	}
	t.Cleanup(func() {
		consumer.Stop(context.Background())
	})

	received := make(chan ports.NotificationIssued, 1)
	proc := captureProcessor{ch: received}

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(ctx, proc)
	}()

	schema := pulsar.NewAvroSchema(string(avroschemas.Notification), nil)
	prod, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic:  topic,
		Schema: schema,
	})
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	t.Cleanup(func() {
		prod.Close()
	})

	want := ports.NotificationIssued{
		RecipientID: "user-1",
		Channel:     "EMAIL",
		Message:     "hello",
		EventType:   "NotificationIssued",
		Timestamp:   "2024-01-01T00:00:00Z",
	}
	if _, err := prod.Send(ctx, &pulsar.ProducerMessage{Value: want}); err != nil {
		t.Fatalf("failed to publish notification: %v", err)
	}

	select {
	case got := <-received:
		if got != want {
			t.Fatalf("notification mismatch: got %+v want %+v", got, want)
		}
	case err := <-errCh:
		if err != nil {
			t.Fatalf("consumer returned error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatalf("timed out waiting for notification")
	}
}

type captureProcessor struct {
	ch chan ports.NotificationIssued
}

func (p captureProcessor) Process(ctx context.Context, msg ports.Message[ports.NotificationIssued]) error {
	select {
	case p.ch <- msg.Value:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var _ ports.Processor[ports.NotificationIssued] = (*captureProcessor)(nil)
