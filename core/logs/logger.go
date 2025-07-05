// Package logs provides a logger setup function that configures the logger based on the environment.
// It supports different logging levels for development and production environments.
// It uses the standard library's slog package for structured logging.
package logs

import (
	"log/slog"
	"os"
)

// Environment constants for logger setup
const (
	// envDev enables development logging with debug level
	envDev = "dev"
	// envProd enables production logging with info level
	envProd = "prod"
)

// SetupLogger initializes and returns a logger based on the provided environment.
func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}
