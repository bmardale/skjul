package main

import (
	"context"
	"log"

	"github.com/bmardale/skjul/internal/app"
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

	application := app.New(cfg, slog)
	if err := application.Start(context.Background()); err != nil {
		log.Fatalf("application err: %v", err)
	}
}
