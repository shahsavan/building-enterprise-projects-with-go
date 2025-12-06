//go:build integration_test

package test_containers

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
)

var (
	runPulsarOnce sync.Once
	pulsarHost    string
	pulsarPort    string
	pulsarHTTP    string
)

// GetPulsarContainer starts a Pulsar standalone container once and returns the host
// plus mapped binary and HTTP ports. Ports can be bound explicitly via the optional
// pointers; otherwise random available ports are used.
func GetPulsarContainer(ctx context.Context, servicePort, httpPort *int) (string, string, string) {
	runPulsarOnce.Do(func() {
		c, err := testContainerRunner{
			servicePort:  6650,
			name:         "pulsar",
			image:        "apachepulsar/pulsar:3.3.1",
			exposedPorts: []string{"6650/tcp", "8080/tcp"},
			env:          map[string]string{},
			cmd:          []string{"/bin/bash", "-c", "bin/pulsar standalone"},
			hostConfigModifier: pulsarHostConfigModifier(
				servicePort,
				httpPort,
			),
		}.Run(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to run Pulsar container")
		}
		mp, err := c.MappedPort(ctx, "6650")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get Pulsar port")
		}
		httpMapped, err := c.MappedPort(ctx, "8080")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get Pulsar HTTP port")
		}
		h, err := c.Host(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get Pulsar host")
		}
		pulsarHost = h
		pulsarPort = mp.Port()
		pulsarHTTP = httpMapped.Port()
	})
	return pulsarHost, pulsarPort, pulsarHTTP
}
func pulsarHostConfigModifier(servicePort, httpPort *int) func(hostConfig *container.HostConfig) {
	return func(hostConfig *container.HostConfig) {
		hostConfig.AutoRemove = true
		if servicePort != nil || httpPort != nil {
			portMap := nat.PortMap{}
			if servicePort != nil {
				portMap["6650/tcp"] = []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: strconv.Itoa(*servicePort),
					},
				}
			}
			if httpPort != nil {
				portMap["8080/tcp"] = []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: fmt.Sprintf("%d", *httpPort),
					},
				}
			}
			hostConfig.PortBindings = portMap
		}
	}
}
