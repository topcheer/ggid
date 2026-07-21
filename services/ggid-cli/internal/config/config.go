// Package config manages the GGID CLI configuration file (~/.ggid/config.json).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the persistent CLI configuration.
type Config struct {
	// ServerURL is the GGID Gateway base URL.
	ServerURL string `json:"server_url"`
	// ConsoleTenantID is the console (default) tenant UUID used for DCR.
	ConsoleTenantID string `json:"console_tenant_id"`
	// ClientID is the DCR-registered public client ID (no secret needed).
	ClientID string `json:"client_id,omitempty"`
	// AccessToken is the cached user access token (from device_code flow).
	AccessToken string `json:"access_token,omitempty"`
	// RefreshToken is the cached refresh token for automatic renewal.
	RefreshToken string `json:"refresh_token,omitempty"`
	// ExpiresAt is the token expiry timestamp (Unix seconds).
	ExpiresAt int64 `json:"expires_at,omitempty"`
	// OutputFormat controls output style: "json" or "table".
	OutputFormat string `json:"output_format,omitempty"`
}

const (
	configDir  = ".ggid"
	configFile = "config.json"
)

// configPath returns the full path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads the configuration from disk. Returns an empty Config if the file
// does not exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("cannot read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the configuration to disk.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}
	return nil
}

// Delete removes the configuration file from disk.
func Delete() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot delete config: %w", err)
	}
	return nil
}
