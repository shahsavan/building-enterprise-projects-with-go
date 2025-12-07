package configs_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/yourname/transport/vehicle/configs"
)

func TestLoadConfig(t *testing.T) {
	// Helper to create a temporary config file
	createTempConfigFile := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write temp config file: %v", err)
		}
		return path
	}

	validYAML := `
server:
  port: 8080
  read_timeout_sec: 30
  write_timeout_sec: 30
database:
  host: "mysql.internal"
  port: 3306
  user: "ride_user"
  password: "from_file"
  name: "transportdb"
  max_open_conns: 20
  max_idle_conns: 5
`
	invalidYAML := `
server:
  port: 8080
database
  host: "mysql.internal"
`

	testCases := []struct {
		name        string
		path        func(t *testing.T) string // Function to generate path
		envVars     map[string]string
		expectedCfg *configs.Config
		expectErr   bool
	}{
		{
			name: "success - load from file",
			path: func(t *testing.T) string {
				return createTempConfigFile(t, validYAML)
			},
			envVars: nil,
			expectedCfg: &configs.Config{
				Server: configs.ServerConfig{
					Port:            8080,
					ReadTimeoutSec:  30,
					WriteTimeoutSec: 30,
				},
				Database: configs.DatabaseConfig{
					Host:         "mysql.internal",
					Port:         3306,
					User:         "ride_user",
					Password:     "from_file",
					Name:         "transportdb",
					MaxOpenConns: 20,
					MaxIdleConns: 5,
				},
			},
			expectErr: false,
		},
		{
			name: "success - override with env vars",
			path: func(t *testing.T) string {
				return createTempConfigFile(t, validYAML)
			},
			envVars: map[string]string{
				"DB_PASSWORD": "from_env",
				"SERVER_PORT": "9090",
			},
			expectedCfg: &configs.Config{
				Server: configs.ServerConfig{
					Port:            9090, // overridden
					ReadTimeoutSec:  30,
					WriteTimeoutSec: 30,
				},
				Database: configs.DatabaseConfig{
					Host:         "mysql.internal",
					Port:         3306,
					User:         "ride_user",
					Password:     "from_env", // overridden
					Name:         "transportdb",
					MaxOpenConns: 20,
					MaxIdleConns: 5,
				},
			},
			expectErr: false,
		},
		{
			name: "success - ignore invalid server port env var",
			path: func(t *testing.T) string {
				return createTempConfigFile(t, validYAML)
			},
			envVars: map[string]string{
				"SERVER_PORT": "not-a-number",
			},
			expectedCfg: &configs.Config{
				Server: configs.ServerConfig{
					Port:            8080, // not overridden
					ReadTimeoutSec:  30,
					WriteTimeoutSec: 30,
				},
				Database: configs.DatabaseConfig{
					Host:         "mysql.internal",
					Port:         3306,
					User:         "ride_user",
					Password:     "from_file",
					Name:         "transportdb",
					MaxOpenConns: 20,
					MaxIdleConns: 5,
				},
			},
			expectErr: false,
		},
		{
			name:      "error - empty path",
			path:      func(t *testing.T) string { return "" },
			expectErr: true,
		},
		{
			name:      "error - file not found",
			path:      func(t *testing.T) string { return "non_existent_file.yaml" },
			expectErr: true,
		},
		{
			name: "error - invalid yaml",
			path: func(t *testing.T) string {
				return createTempConfigFile(t, invalidYAML)
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables for the test
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			path := tc.path(t)

			cfg, err := configs.LoadConfig(path)

			if tc.expectErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, but got: %v", err)
				}
				if !reflect.DeepEqual(cfg, tc.expectedCfg) {
					t.Errorf("config mismatch:\ngot:  %+v\nwant: %+v", cfg, tc.expectedCfg)
				}
			}
		})
	}
}
func FuzzLoadConfigEnvOverrides(f *testing.F) {
	const baseConfig = `
server:
  port: 8080
  read_timeout_sec: 30
  write_timeout_sec: 30
database:
  host: "mysql.internal"
  port: 3306
  user: "ride_user"
  password: "from_file"
  name: "transportdb"
  max_open_conns: 20
  max_idle_conns: 5
`

	// Seeds cover common and edge cases: valid override, invalid port, and max-ish port.
	f.Add("9090", "from_env")
	f.Add("not-a-number", "")
	f.Add("65535", "rot13$ecret")

	f.Fuzz(func(t *testing.T, portEnv, passwordEnv string) {
		// testing.F may generate strings with NULs which Setenv rejects; strip them to keep fuzzing valid inputs.
		portEnv = strings.ReplaceAll(portEnv, "\x00", "")
		passwordEnv = strings.ReplaceAll(passwordEnv, "\x00", "")

		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(baseConfig), 0600); err != nil {
			t.Fatalf("failed to write temp config file: %v", err)
		}

		if portEnv != "" {
			t.Setenv("SERVER_PORT", portEnv)
		}
		t.Setenv("DB_PASSWORD", passwordEnv)

		cfg, err := configs.LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig returned error: %v", err)
		}

		if p, err := strconv.Atoi(portEnv); err == nil {
			if cfg.Server.Port != p {
				t.Fatalf("expected server port %d after env override, got %d", p, cfg.Server.Port)
			}
		} else if cfg.Server.Port != 8080 {
			t.Fatalf("expected fallback server port 8080 when env is invalid, got %d", cfg.Server.Port)
		}

		expectedPassword := "from_file"
		if passwordEnv != "" {
			expectedPassword = passwordEnv
		}
		if cfg.Database.Password != expectedPassword {
			t.Fatalf("expected password %q, got %q", expectedPassword, cfg.Database.Password)
		}
	})
}
