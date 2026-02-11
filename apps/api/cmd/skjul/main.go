package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/bmardale/skjul/internal/app"
	"github.com/bmardale/skjul/internal/config"
	"github.com/bmardale/skjul/internal/db/migrations"
	"github.com/bmardale/skjul/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
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

	if err := runMigrations(cfg.Database.DatabaseURL); err != nil {
		slog.ErrorContext(context.Background(), "failed to run migrations", "error", err)
		os.Exit(1)
	}

	db, err := initDB(context.Background(), cfg)
	if err != nil {
		slog.ErrorContext(context.Background(), "failed initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	application := app.New(cfg, slog, db)
	if err := application.Start(context.Background()); err != nil {
		log.Fatalf("application err: %v", err)
	}
}

func initDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.Database.DatabaseURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = int32(cfg.Database.MaxOpenConns)
	config.ConnConfig.ConnectTimeout = cfg.Database.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func runMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	return migrations.Run(db)
}
