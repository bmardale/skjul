package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/bmardale/skjul/internal/config"
)

func Setup(cfg config.LoggerConfig) (*slog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
		case "json":
			handler = slog.NewJSONHandler(os.Stdout, opts)
		case "text":
			handler = slog.NewTextHandler(os.Stdout, opts)
		default:
			return nil, fmt.Errorf("unsupported log format: %s", cfg.Format)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %s", s)
	}
}
