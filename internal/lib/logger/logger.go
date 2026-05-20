package logger

import (
	"log/slog"
	"os"
)

const (
	EnvLocal = "local"
	EnvProd  = "prod"
)

var Logger *slog.Logger

// Setup initializes the slog based on the environment.
func Setup(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case EnvLocal:
		// Text format, shows Debug level and above
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	case EnvProd:
		// JSON format, shows Info level and above
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	Logger = log

	return log
}
