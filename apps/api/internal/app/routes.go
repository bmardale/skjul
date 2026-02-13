package app

import (
	"net/http"

	"github.com/bmardale/skjul/internal/admin"
	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/bmardale/skjul/internal/invitations"
	"github.com/bmardale/skjul/internal/pastes"
	"github.com/gin-gonic/gin"
)

func (a *App) setupRoutes() {
	a.router.NoRoute(a.noRoute)

	v1 := a.router.Group("/api/v1")
	v1.GET("/health", a.healthCheck)
	v1.GET("/config", a.publicConfig)

	queries := sqlc.New(a.db)
	invSvc := invitations.NewService(queries, a.db, a.config.Invitations)

	authSvc := auth.RegisterRoutesWithOpts(v1, auth.RegisterRoutesOpts{
		DB:             a.db,
		Logger:         a.logger,
		InvSvc:         invSvc,
		AdminUsernames: a.config.Admin.Admins,
	})
	pastes.RegisterRoutes(v1, a.db, a.logger, authSvc, a.s3Client)
	invitations.RegisterRoutes(v1, a.db, authSvc, a.config.Invitations, a.logger)
	admin.RegisterRoutes(v1, a.db, authSvc, a.config.Admin.Admins, a.logger)
}

func (a *App) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (a *App) publicConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"require_invite_code": a.config.Invitations.RequireInviteCode,
	})
}

func (a *App) noRoute(c *gin.Context) {
	apierr.ErrNotFound.Respond(c)
}
