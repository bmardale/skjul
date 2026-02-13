package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/bmardale/skjul/internal/apierr"
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
			apierr.ErrUnauthorized.Abort(c)
			return
		}

		userID, sessionID, err := service.GetUserIDFromSession(c.Request.Context(), token)
		if err != nil {
			apierr.ErrUnauthorized.Abort(c)
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
