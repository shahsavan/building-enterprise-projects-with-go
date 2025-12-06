//go:build integration_test

package pulsarproducer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/yourname/transport/ride/internal/adapters/pulsarproducer"
	"github.com/yourname/transport/ride/internal/ports"
	"github.com/yourname/transport/ride/test_containers"
)

func TestAssignmentCreatedProducerPublishesMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	const (
		namespace   = "public/default"
		fullTopic   = "persistent://public/default/assignments"
		topic       = "assignments"
		subscribers = "assignment-producer-test"
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

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subscribers,
		Type:             pulsar.Exclusive,
	})
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	t.Cleanup(func() {
		consumer.Close()
	})

	producer, err := pulsarproducer.NewAssignmentCreatedProducer(client)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	t.Cleanup(func() {
		producer.Close()
	})

	driverID := "driver-123"
	msg := ports.AssignmentCreated{
		AssignmentID: "assign-001",
		VehicleID:    "vehicle-456",
		RouteID:      "route-789",
		Timestamp:    "2024-01-01T00:00:00Z",
		DriverID:     &driverID,
	}

	if _, err := producer.Send(ctx, msg); err != nil {
		t.Fatalf("failed to publish assignment: %v", err)
	}

	recvCtx, cancelRecv := context.WithTimeout(ctx, 30*time.Second)
	defer cancelRecv()

	got, err := consumer.Receive(recvCtx)
	if err != nil {
		t.Fatalf("failed to receive message: %v", err)
	}

	wantPayload := true
	gotPayload := len(got.Payload()) > 0
	if gotPayload != wantPayload {
		t.Fatalf("payload presence mismatch: want payload=%t got=%t", wantPayload, gotPayload)
	}

	if err := consumer.Ack(got); err != nil {
		t.Fatalf("failed to ack message: %v", err)
	}
}
