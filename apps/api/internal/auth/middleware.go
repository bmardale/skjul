package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	SessionCookieName = "session_token"
	UserIDKey         = "user_id"
	SessionIDKey      = "session_id"
)

func RequireAuth(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(SessionCookieName)
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, errorResponse{
				Code:    "UNAUTHORIZED",
				Message: "missing or invalid session",
			})
			c.Abort()
			return
		}

		userID, sessionID, err := service.GetUserIDFromSession(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, errorResponse{
				Code:    "UNAUTHORIZED",
				Message: "missing or invalid session",
			})
			c.Abort()
			return
		}

		c.Set(UserIDKey, userID)
		c.Set(SessionIDKey, sessionID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(UserIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func GetSessionID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(SessionIDKey)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}
