package auth

import (
	"log/slog"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *gin.RouterGroup, db *pgxpool.Pool, logger *slog.Logger) {
	queries := sqlc.New(db)
	svc := NewService(queries, db)
	handler := NewHandler(svc, logger)

	r.POST("/auth/register", handler.Register)
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/logout", handler.Logout)

	protected := r.Group("")
	protected.Use(RequireAuth(svc))
	protected.GET("/me", handler.Me)
}
