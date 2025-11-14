package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/transport/ride/configs"
	"github.com/yourname/transport/ride/internal/adapters/http/api"
	"github.com/yourname/transport/ride/internal/adapters/http/handler"
	"github.com/yourname/transport/ride/internal/ports"
	"github.com/yourname/transport/ride/internal/service"
)

// Run initializes and starts the HTTP server based on the provided configuration.
// It returns an error if the server fails to start.
func Run(cfg configs.ServerConfig, repo ports.AssignmentRepository) error {
	log.Printf("Starting server on port %d", cfg.Port)

	router := gin.Default()
	// Add health endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "Ok")
	})
	assignmentService := service.NewAssignmentService(repo)
	// Initialize your handler that implements api.ServerInterface, injecting any services needed
	hndlr := handler.NewAssignmentHandler(assignmentService)
	// Register OpenAPI routes (e.g. /assignments)
	api.RegisterHandlers(router, hndlr)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSec) * time.Second,
	}

	log.Println("Starting server on port", cfg.Port)
	// The ListenAndServe call is blocking. It will only return on an
	// unrecoverable error. We return that error to the caller (main).
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not listen on %s: %w", server.Addr, err)
	}

	log.Println("Server exited gracefully.")
	return nil
}
