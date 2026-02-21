package invitations

import (
	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/config"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *gin.RouterGroup, db *pgxpool.Pool, authSvc *auth.Service, cfg config.InvitationsConfig) {
	queries := sqlc.New(db)
	svc := NewService(queries, db, cfg)
	handler := NewHandler(svc)

	protected := r.Group("")
	protected.Use(auth.RequireAuth(authSvc))
	protected.POST("/invitations", handler.GenerateInvite)
	protected.GET("/invitations", handler.ListInvites)
}
