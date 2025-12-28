package configs // if file is ride/configs/config.go
// package ride   // if you keep it as ride/config.go at the service root

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port            int `yaml:"port"`
	ReadTimeoutSec  int `yaml:"read_timeout_sec"`
	WriteTimeoutSec int `yaml:"write_timeout_sec"`
}

type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Name            string `yaml:"name"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime_sec"`
	ConnMaxIdleTime int    `yaml:"conn_max_idle_time_sec"`
}

type PulsarConfig struct {
	URL string `yaml:"url"`

	OperationTimeout        time.Duration `yaml:"operation_timeout"`
	ConnectionTimeout       time.Duration `yaml:"connection_timeout"`
	ConnectionMaxIdleTime   time.Duration `yaml:"connection_max_idle_time"`
	KeepAliveInterval       time.Duration `yaml:"keep_alive_interval"`
	MaxConnectionsPerBroker int           `yaml:"max_connections_per_broker"`
	MemoryLimitBytes        int64         `yaml:"memory_limit_bytes"`

	Consumer PulsarConsumerConfig `yaml:"consumer"`
	Producer PulsarProducerConfig `yaml:"producer"`
}

type PulsarConsumerConfig struct {
	Topic                string        `yaml:"topic"`                   // topic name
	SubscriptionName     string        `yaml:"subscription_name"`       // subscription name
	Name                 string        `yaml:"name"`                    // optional consumer name
	SubscriptionType     string        `yaml:"subscription_type"`       // exclusive|shared|failover|key_shared; defaults to shared when empty
	ReceiverQueueSize    int           `yaml:"receiver_queue_size"`     // Receiver Queue Size
	NackRedeliveryDelay  time.Duration `yaml:"nack_redelivery_delay"`   // Nack Redelivery Delay
	MaxReconnectToBroker *uint         `yaml:"max_reconnect_to_broker"` // Maximum Reconnect To Broker
	AutoDiscoveryPeriod  time.Duration `yaml:"auto_discovery_period"`   // Auto Discovery Period
}

type PulsarProducerConfig struct {
	Topic                           string         `yaml:"topic"` // topic name
	Name                            *string        `yaml:"name"`
	CompressionType                 *string        `yaml:"compression_type"`
	PartitionsAutoDiscoveryInterval *time.Duration `yaml:"partitions_auto_discovery_interval"`
	SendTimeout                     time.Duration  `yaml:"send_timeout"` // bounds Send() latency; disabled when zero
	MaxPendingMessages              int            `yaml:"max_pending_messages"`
	DisableBlockIfQueueFull         bool           `yaml:"disable_block_if_queue_full"`
	MaxReconnectToBroker            *uint          `yaml:"max_reconnect_to_broker"`
	DisableBatching                 bool           `yaml:"disable_batching"`
	BatchingMaxPublishDelay         time.Duration  `yaml:"batching_max_publish_delay"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Pulsar   PulsarConfig   `yaml:"pulsar"`
}

// LoadConfig reads and parses the configuration file from the given path.
// It returns a Config struct or an error if loading fails.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path cannot be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}
	// Override with environment variables if set
	if pw := os.Getenv("DB_PASSWORD"); pw != "" {
		cfg.Database.Password = pw
	}

	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = p
		}
	}
	// end of env var overrides

	return &cfg, nil
}
