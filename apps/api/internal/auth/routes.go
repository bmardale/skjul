package auth

import (
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegisterRoutesOpts struct {
	DB             *pgxpool.Pool
	InvSvc         InvitationsService
	AdminUsernames []string
}

func RegisterRoutes(r *gin.RouterGroup, db *pgxpool.Pool) *Service {
	return RegisterRoutesWithOpts(r, RegisterRoutesOpts{DB: db})
}

func RegisterRoutesWithOpts(r *gin.RouterGroup, opts RegisterRoutesOpts) *Service {
	queries := sqlc.New(opts.DB)
	svc := NewService(queries, opts.DB)
	var handler *Handler
	if opts.InvSvc != nil && opts.DB != nil {
		handler = NewHandlerWithInvitations(svc, opts.InvSvc, opts.DB, opts.AdminUsernames)
	} else {
		handler = NewHandler(svc, opts.AdminUsernames)
	}

	r.POST("/auth/register", handler.Register)
	r.POST("/auth/login/challenge", handler.LoginChallenge)
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/logout", handler.Logout)

	protected := r.Group("")
	protected.Use(RequireAuth(svc))
	protected.GET("/me", handler.Me)
	protected.GET("/sessions", handler.ListSessions)
	protected.DELETE("/sessions/:id", handler.DeleteSession)
	protected.DELETE("/me", handler.DeleteAccount)

	return svc
}
