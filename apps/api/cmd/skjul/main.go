package main

import (
	"log"

	"github.com/bmardale/skjul/internal/config"
	"github.com/bmardale/skjul/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	slog, err := logger.Setup(cfg.Logger)
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}

	slog.Info("starting skjul", "host", cfg.HTTP.Host, "port", cfg.HTTP.Port)
}
