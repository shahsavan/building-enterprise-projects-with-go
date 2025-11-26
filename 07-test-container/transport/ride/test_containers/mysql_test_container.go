//go:build integration_test

package test_containers

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
)

var (
	mysqlOnce sync.Once
	mysqlHost string
	mysqlPort string
)

func GetMySqlContainer(ctx context.Context, db, user, pass string, port *int) (string, string) {
	mysqlOnce.Do(func() {
		c, err := testContainerRunner{
			servicePort:  3306,
			name:         "mysql",
			image:        "mysql:8.0.36", // pin to a stable version
			exposedPorts: []string{"3306/tcp"},
			env: map[string]string{
				"MYSQL_ROOT_PASSWORD": pass, // root password
				"MYSQL_DATABASE":      db,   // database name
				"MYSQL_USER":          user, // user
				"MYSQL_PASSWORD":      pass, // user password
			},
			hostConfigModifier: mysqlHostConfigModifier(port),
		}.Run(ctx)

		if err != nil {
			log.Fatal().Err(err).Msg("Failed to run Test Container")
		}
		mp, err := c.MappedPort(ctx, "3306")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get port")
		}
		h, err := c.Host(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get host")
		}
		mysqlHost = h
		mysqlPort = mp.Port()
	})
	return mysqlHost, mysqlPort
}

func mysqlHostConfigModifier(port *int) func(hostConfig *container.HostConfig) {
	return func(hostConfig *container.HostConfig) {
		hostConfig.AutoRemove = true
		if port != nil {
			hostConfig.PortBindings = nat.PortMap{"3306/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", *port),
				},
			}}
		}
	}
}
