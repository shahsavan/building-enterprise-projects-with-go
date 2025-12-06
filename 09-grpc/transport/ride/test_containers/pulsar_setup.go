//go:build integration_test

package test_containers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	pulsaradmin "github.com/apache/pulsar-client-go/pulsaradmin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	adminutils "github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
)

// PulsarInstance captures the connection details for a running Pulsar container.
type PulsarInstance struct {
	Host     string
	Port     string // Binary service port
	HTTPPort string
	AdminURL string
}

// EnsurePulsarTopic starts Pulsar (once) and ensures the given namespace and topic exist.
// It is safe to call from multiple tests; the container is shared via sync.Once.
func EnsurePulsarTopic(
	ctx context.Context,
	namespace string,
	topic string,
	partitions int,
	servicePort,
	httpPort *int,
) (*PulsarInstance, error) {
	host, pulsarPort, pulsarHTTP := GetPulsarContainer(ctx, servicePort, httpPort)
	adminURL := fmt.Sprintf("http://%s:%s", host, pulsarHTTP)

	if err := waitForPulsarAdmin(ctx, adminURL); err != nil {
		return nil, err
	}

	admin, err := pulsaradmin.NewClient(&pulsaradmin.Config{
		WebServiceURL: adminURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create pulsar admin client: %w", err)
	}

	if err := ensureNamespace(ctx, admin, namespace); err != nil {
		return nil, err
	}
	if err := ensureTopic(ctx, admin, topic, partitions); err != nil {
		return nil, err
	}

	return &PulsarInstance{
		Host:     host,
		Port:     pulsarPort,
		HTTPPort: pulsarHTTP,
		AdminURL: adminURL,
	}, nil
}

func ensureNamespace(ctx context.Context, admin pulsaradmin.Client, namespace string) error {
	err := admin.Namespaces().CreateNamespaceWithContext(ctx, namespace)
	if err == nil {
		return nil
	}
	var adminErr rest.Error
	if errors.As(err, &adminErr) && adminErr.Code == http.StatusConflict {
		return nil
	}
	return fmt.Errorf("create namespace %s: %w", namespace, err)
}

func ensureTopic(ctx context.Context, admin pulsaradmin.Client, topic string, partitions int) error {
	topicName, err := adminutils.GetTopicName(topic)
	if err != nil {
		return fmt.Errorf("parse topic name %s: %w", topic, err)
	}
	if err := admin.Topics().CreateWithContext(ctx, *topicName, partitions); err != nil {
		var adminErr rest.Error
		if errors.As(err, &adminErr) && adminErr.Code == http.StatusConflict {
			return nil
		}
		return fmt.Errorf("create topic %s: %w", topicName.String(), err)
	}
	return nil
}

func waitForPulsarAdmin(ctx context.Context, adminURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		req, _ := http.NewRequestWithContext(
			reqCtx,
			http.MethodGet,
			fmt.Sprintf("%s/admin/v2/brokers/health", adminURL),
			nil,
		)
		resp, err := client.Do(req)
		cancel()
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("pulsar admin not ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
