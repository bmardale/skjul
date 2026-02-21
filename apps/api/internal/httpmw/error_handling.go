package httpmw

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDHeader = "X-Request-Id"
	requestIDKey    = "request_id"
)

func ErrorHandling(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(requestIDKey, requestID)
		c.Header(requestIDHeader, requestID)

		defer func() {
			if recovered := recover(); recovered != nil {
				panicErr := fmt.Errorf("panic: %v", recovered)
				apierr.RecordWithStack(c, panicErr, "panic_recovered", string(debug.Stack()))
				if !c.Writer.Written() {
					apierr.ErrInternal.Abort(c)
				} else {
					c.Abort()
				}
			}

			status := c.Writer.Status()
			if status < http.StatusInternalServerError {
				return
			}

			route := c.FullPath()
			if route == "" {
				route = c.Request.URL.Path
			}

			attrs := []any{
				"request_id", requestID,
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"route", route,
				"status", status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"client_ip", c.ClientIP(),
			}

			if userID, ok := c.Get(auth.UserIDKey); ok {
				attrs = append(attrs, "user_id", toLogValue(userID))
			}
			if sessionID, ok := c.Get(auth.SessionIDKey); ok {
				attrs = append(attrs, "session_id", toLogValue(sessionID))
			}

			if report, ok := apierr.GetReport(c); ok {
				if report.Op != "" {
					attrs = append(attrs, "op", report.Op)
				}
				if report.Cause != nil {
					attrs = append(attrs, "error", report.Cause)
				}
				if report.Stack != "" {
					attrs = append(attrs, "stack", report.Stack)
				}
			}

			logger.Error("request failed", attrs...)
		}()

		c.Next()
	}
}

func toLogValue(v any) any {
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}
	return v
}
