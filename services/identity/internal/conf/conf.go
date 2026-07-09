// Package conf holds the configuration types for the Identity Service.
package conf

import (
	"time"
)

// Config holds all configuration for the Identity Service.
type Config struct {
	// Server configuration
	GRPC struct {
		Addr string `json:"addr" yaml:"addr"`
		Timeout time.Duration `json:"timeout" yaml:"timeout"`
	} `json:"grpc" yaml:"grpc"`

	HTTP struct {
		Addr string `json:"addr" yaml:"addr"`
		Timeout time.Duration `json:"timeout" yaml:"timeout"`
	} `json:"http" yaml:"http"`

	// Database configuration
	Database DBConfig `json:"database" yaml:"database"`
}

type DBConfig struct {
	URL             string        `json:"url" yaml:"url"`
	MaxConns        int32         `json:"max_conns" yaml:"max_conns"`
	MinConns        int32         `json:"min_conns" yaml:"min_conns"`
	MaxConnLifetime time.Duration `json:"max_conn_lifetime" yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `json:"max_conn_idle_time" yaml:"max_conn_idle_time"`
}
