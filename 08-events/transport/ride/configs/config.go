package configs // if file is ride/configs/config.go
// package ride   // if you keep it as ride/config.go at the service root

import (
	"errors"
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

	// Validate after all sources (file + env) have been applied.
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if err := c.validateServer(); err != nil {
		errs = append(errs, fmt.Errorf("server: %w", err))
	}
	if err := c.validateDatabase(); err != nil {
		errs = append(errs, fmt.Errorf("database: %w", err))
	}

	// If you prefer fail-fast, just return the first error instead of joining.
	return errors.Join(errs...)
}

func (c Config) validateServer() error {
	var errs []error

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("port %d is out of range (1..65535)", c.Server.Port))
	}

	// Timeouts: allow 0 to mean "no timeout" if that is your policy.
	if c.Server.ReadTimeoutSec < 0 {
		errs = append(errs, fmt.Errorf("read_timeout_sec %d must be >= 0", c.Server.ReadTimeoutSec))
	}
	if c.Server.WriteTimeoutSec < 0 {
		errs = append(errs, fmt.Errorf("write_timeout_sec %d must be >= 0", c.Server.WriteTimeoutSec))
	}

	// Optional enterprise sanity: warn-like validation (still error) for extreme values.
	// Example: timeouts too large are almost always a misconfig.
	if c.Server.ReadTimeoutSec > 3600 {
		errs = append(errs, fmt.Errorf("read_timeout_sec %d is unusually high (>3600)", c.Server.ReadTimeoutSec))
	}
	if c.Server.WriteTimeoutSec > 3600 {
		errs = append(errs, fmt.Errorf("write_timeout_sec %d is unusually high (>3600)", c.Server.WriteTimeoutSec))
	}

	return errors.Join(errs...)
}

func (c Config) validateDatabase() error {
	var errs []error

	if c.Database.Host == "" {
		errs = append(errs, errors.New("host is required"))
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		errs = append(errs, fmt.Errorf("port %d is out of range (1..65535)", c.Database.Port))
	}
	if c.Database.User == "" {
		errs = append(errs, errors.New("user is required"))
	}
	if c.Database.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	// Pool sizes: allow 0 to mean "driver default" if that's your policy.
	if c.Database.MaxOpenConns < 0 {
		errs = append(errs, fmt.Errorf("max_open_conns %d must be >= 0", c.Database.MaxOpenConns))
	}
	if c.Database.MaxIdleConns < 0 {
		errs = append(errs, fmt.Errorf("max_idle_conns %d must be >= 0", c.Database.MaxIdleConns))
	}

	// Relationship constraint.
	if c.Database.MaxOpenConns > 0 && c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		errs = append(errs, fmt.Errorf(
			"max_idle_conns %d must be <= max_open_conns %d",
			c.Database.MaxIdleConns, c.Database.MaxOpenConns,
		))
	}

	// Password policy remains environment-dependent; keep it out of generic validation.
	return errors.Join(errs...)
}
