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
	"github.com/gin-gonic/gin"
)

type App struct {
	router *gin.Engine
	server *http.Server
	config *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) *App {
	router := gin.New()
	router.Use(gin.Recovery())

	httpSrv := &http.Server{
		Addr:         net.JoinHostPort(cfg.HTTP.Host, strconv.Itoa(cfg.HTTP.Port)),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	app := &App{
		router: router,
		server: httpSrv,
		config: cfg,
		logger: logger,
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
