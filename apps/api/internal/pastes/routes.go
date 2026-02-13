package pastes

import (
	"log/slog"

	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *gin.RouterGroup, db *pgxpool.Pool, logger *slog.Logger, authSvc *auth.Service) {
	queries := sqlc.New(db)
	svc := NewService(queries, db)
	handler := NewHandler(svc, logger)

	r.GET("/pastes/:id", handler.GetPaste)

	protected := r.Group("")
	protected.Use(auth.RequireAuth(authSvc))
	protected.POST("/pastes", handler.CreatePaste)
	protected.GET("/pastes", handler.ListPastes)
	protected.DELETE("/pastes/:id", handler.DeletePaste)
}
