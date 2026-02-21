package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bmardale/skjul/internal/config"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/bmardale/skjul/internal/pastes"
	"github.com/bmardale/skjul/internal/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	router   *gin.Engine
	server   *http.Server
	config   *config.Config
	logger   *slog.Logger
	db       *pgxpool.Pool
	s3Client *storage.S3Client
}

func New(cfg *config.Config, logger *slog.Logger, db *pgxpool.Pool, s3Client *storage.S3Client) *App {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	if len(cfg.HTTP.CORSAllowOrigins) > 0 {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.HTTP.CORSAllowOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowHeaders:     []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
		}))
	}

	httpSrv := &http.Server{
		Addr:         net.JoinHostPort(cfg.HTTP.Host, strconv.Itoa(cfg.HTTP.Port)),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	app := &App{
		router:   router,
		server:   httpSrv,
		config:   cfg,
		logger:   logger,
		db:       db,
		s3Client: s3Client,
	}

	app.setupRoutes()

	return app
}

const shutdownTimeout = 10 * time.Second

func (a *App) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	a.logger.Info("api server starting", "addr", a.server.Addr)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case <-ctx.Done():
		a.logger.Info("context canceled, shutting down")
	case sig := <-signalCh:
		a.logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		return fmt.Errorf("http server err: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}

func (a *App) CleanupExpiredNotes(ctx context.Context) error {
	queries := sqlc.New(a.db)
	svc := pastes.NewService(queries, a.db, a.s3Client)
	return svc.CleanupExpiredNotes(ctx)
}

func (a *App) StartCleanupLoop(ctx context.Context) {
	if !a.config.Cleanup.Enabled {
		a.logger.Info("cleanup loop disabled")
		return
	}

	interval := a.config.Cleanup.Interval
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	a.logger.Info("cleanup loop starting", "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if err := a.CleanupExpiredNotes(ctx); err != nil {
		a.logger.Warn("initial cleanup failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("cleanup loop stopping")
			return
		case <-ticker.C:
			if err := a.CleanupExpiredNotes(ctx); err != nil {
				a.logger.Warn("cleanup failed", "error", err)
			}
		}
	}
}
