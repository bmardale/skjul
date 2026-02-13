package admin

import (
	"slices"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
)

func RequireAdmin(queries *sqlc.Queries, adminUsernames []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := auth.GetUserID(c)
		if !ok {
			apierr.ErrUnauthorized.Abort(c)
			return
		}

		user, err := queries.GetUserBasic(c.Request.Context(), userID)
		if err != nil {
			apierr.Forbidden("admin access required").Abort(c)
			return
		}

		if !slices.Contains(adminUsernames, user.Username) {
			apierr.Forbidden("admin access required").Abort(c)
			return
		}

		c.Next()
	}
}
