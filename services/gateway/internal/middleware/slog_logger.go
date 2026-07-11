// Package middleware provides a shared structured logger for gateway middleware.
package middleware

import (
	"log/slog"
	"os"
)

// gatewayLogger is the default structured logger for gateway middleware.
// It emits JSON to stderr with request_id, tenant_id, user_id fields when available.
var gatewayLogger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// SetLogger allows overriding the gateway middleware logger.
// Useful for testing or when integrating with a service-level logger.
func SetLogger(l *slog.Logger) {
	if l != nil {
		gatewayLogger = l
	}
}

// GatewayLogger returns the default gateway middleware logger.
func GatewayLogger() *slog.Logger {
	return gatewayLogger
}
