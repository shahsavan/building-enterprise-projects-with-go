package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type DeliveryRequest struct {
	ID      string
	Pickup  string
	Dropoff string
}

func processDelivery(req DeliveryRequest) string {
	// Simulate planning a route or printing a ticket for the delivery.
	return fmt.Sprintf("ticket %s: %s -> %s", req.ID, req.Pickup, req.Dropoff)
}

// merge is the fan-in stage: it drains multiple inputs into a single output.
func merge[T any](inputs ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	output := make(chan T)

	for _, ch := range inputs {
		wg.Go(func() {
			func(c <-chan T) {
				for v := range c {
					output <- v
				}
			}(ch)
		})
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}

// worker is the fan-out stage: each worker consumes from the shared input channel.
func worker[T any, R any](ctx context.Context, id int, in <-chan T, process func(T) R) <-chan R {
	out := make(chan R)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return

			case n, ok := <-in:
				if !ok {
					return
				}
				time.Sleep(150 * time.Millisecond) // simulate work
				fmt.Printf("worker %d handling %v\n", id, n)
				out <- process(n)

			}
		}
	}()
	return out
}

func main() {
	requests := []DeliveryRequest{
		{ID: "A1", Pickup: "Central Depot", Dropoff: "North Hub"},
		{ID: "B2", Pickup: "Warehouse", Dropoff: "Airport"},
		{ID: "C3", Pickup: "Port", Dropoff: "City Station"},
		{ID: "D4", Pickup: "Downtown", Dropoff: "University"},
		{ID: "E5", Pickup: "East Yard", Dropoff: "Harbor"},
	}
	jobs := make(chan DeliveryRequest)
	ctx, cancel := context.WithCancel(context.Background())

	worker1Output := worker(ctx, 1, jobs, processDelivery)
	worker2Output := worker(ctx, 2, jobs, processDelivery)
	worker3Output := worker(ctx, 3, jobs, processDelivery)
	cancel()
	// Feed work and then close the input channel.
	go func() {
		defer close(jobs)
		for _, req := range requests {
			jobs <- req
		}
	}()

	// Fan-in: merge worker outputs into a single stream.
	results := merge(worker1Output, worker2Output, worker3Output)

	for result := range results {
		fmt.Printf("completed: %s\n", result)
	}
}
