package admin

import (
	"log/slog"

	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *gin.RouterGroup, db *pgxpool.Pool, authSvc *auth.Service, adminUsernames []string, logger *slog.Logger) {
	queries := sqlc.New(db)
	handler := NewHandler(queries, logger)

	admin := r.Group("/admin")
	admin.Use(auth.RequireAuth(authSvc))
	admin.Use(RequireAdmin(queries, adminUsernames))
	admin.GET("/users", handler.ListUsers)
	admin.GET("/users/:id", handler.GetUser)
	admin.DELETE("/users/:id", handler.DeleteUser)
	admin.PATCH("/users/:id/invite-quota", handler.UpdateInviteQuota)
}
