package app

import (
	"net/http"

	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/pastes"
	"github.com/gin-gonic/gin"
)

func (a *App) setupRoutes() {
	a.router.NoRoute(a.noRoute)

	v1 := a.router.Group("/api/v1")
	v1.GET("/health", a.healthCheck)

	authSvc := auth.RegisterRoutes(v1, a.db, a.logger)
	pastes.RegisterRoutes(v1, a.db, a.logger, authSvc, a.s3Client)
}

func (a *App) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (a *App) noRoute(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}
