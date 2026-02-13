package admin

import (
	"net/http"
	"slices"

	"github.com/bmardale/skjul/internal/auth"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
)

func RequireAdmin(queries *sqlc.Queries, adminUsernames []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := auth.GetUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing or invalid session",
			})
			c.Abort()
			return
		}

		user, err := queries.GetUserBasic(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    "FORBIDDEN",
				"message": "admin access required",
			})
			c.Abort()
			return
		}

		if !slices.Contains(adminUsernames, user.Username) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    "FORBIDDEN",
				"message": "admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
