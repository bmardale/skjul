package pastes

import (
	"log/slog"

	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/bmardale/skjul/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *gin.RouterGroup, db *pgxpool.Pool, logger *slog.Logger, authSvc *auth.Service, s3Client *storage.S3Client) {
	queries := sqlc.New(db)
	svc := NewService(queries, db, s3Client)
	handler := NewHandler(svc, logger)

	r.GET("/pastes/:id", handler.GetPaste)
	r.GET("/pastes/:id/meta", handler.GetPasteMeta)
	r.POST("/pastes/:id/consume", handler.ConsumePaste)

	protected := r.Group("")
	protected.Use(auth.RequireAuth(authSvc))
	protected.POST("/pastes", handler.CreatePaste)
	protected.POST("/pastes/:id/attachments", handler.CreateAttachment)
	protected.GET("/pastes", handler.ListPastes)
	protected.DELETE("/pastes/:id", handler.DeletePaste)
}
