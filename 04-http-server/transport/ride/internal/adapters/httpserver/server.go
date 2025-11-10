package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourname/transport/ride/configs"
)

// Run initializes and starts the HTTP server based on the provided configuration.
// It returns an error if the server fails to start.
func Run(cfg configs.Config) error {
	log.Printf("Starting server on port %d", cfg.Server.Port)

	// Create a simple handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from the ride service!")
	})

	// Liveness probe endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Ok")
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
	}

	// The ListenAndServe call is blocking. It will only return on an
	// unrecoverable error. We return that error to the caller (main).
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not listen on %s: %w", server.Addr, err)
	}

	log.Println("Server exited gracefully.")
	return nil
}

func RunGraceful(cfg configs.Config) error {
	log.Printf("starting server on :%d", cfg.Server.Port)

	// Build mux and routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from the ride service!")
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Ok")
	})

	// Server with timeouts from config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
	}

	// Use a context that cancels on SIGINT or SIGTERM
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		// http.ErrServerClosed is returned on Shutdown. Treat other errors as fatal.
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("listen failure on %s: %w", server.Addr, err)
			return
		}
		errCh <- nil
	}()

	// Wait for a termination signal or a startup error
	select {
	case <-sigCtx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		// Server failed to start or exited unexpectedly
		if err != nil {
			return err
		}
		// Clean exit (e.g., server closed elsewhere)
		log.Println("server exited")
		return nil
	}

	// Optional: hook for cleanup tasks
	server.RegisterOnShutdown(func() {
		log.Println("server is shutting down, cleanup started")
		// close queues, flush metrics, etc.
	})

	// Graceful shutdown with timeout
	// Use config if available, otherwise fall back to 15 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		// Force close if not graceful in time
		log.Printf("graceful shutdown timed out, forcing close: %v", err)
		_ = server.Close()
	}

	log.Println("server exited gracefully")
	return nil
}
