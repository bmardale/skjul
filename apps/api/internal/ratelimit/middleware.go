package ratelimit

import (
	"math"
	"strconv"
	"time"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/config"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rule struct {
	name   string
	method string
	path   string
	limit  rate.Limit
	burst  int
}

func toRule(name, method, path string, cfg config.LimitConfig) rule {
	r := rate.Every(cfg.Window / time.Duration(cfg.Requests))
	burst := cfg.Burst
	if burst <= 0 {
		burst = cfg.Requests
	}
	return rule{name: name, method: method, path: path, limit: r, burst: burst}
}

func SensitivePaths(cfg config.RateLimitConfig) gin.HandlerFunc {
	store := NewStore(cfg.EntryTTL, cfg.CleanupInterval)

	rules := []rule{
		toRule("register", "POST", "/api/v1/auth/register", cfg.Register),
		toRule("login_challenge", "POST", "/api/v1/auth/login/challenge", cfg.LoginChallenge),
		toRule("login", "POST", "/api/v1/auth/login", cfg.Login),
		toRule("paste_get", "GET", "/api/v1/pastes/:id", cfg.PasteGet),
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		method := c.Request.Method
		fullPath := c.FullPath()

		var matched *rule
		for i := range rules {
			if rules[i].method == method && rules[i].path == fullPath {
				matched = &rules[i]
				break
			}
		}
		if matched == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		limiter := store.Limiter(matched.name, ip, matched.limit, matched.burst)

		c.Header("X-RateLimit-Limit", strconv.Itoa(matched.burst))

		if !limiter.Allow() {
			retryAfter := int(math.Ceil(1.0 / float64(matched.limit)))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			apierr.ErrRateLimited.Abort(c)
			return
		}

		c.Header("X-RateLimit-Remaining", strconv.Itoa(int(limiter.Tokens())))
		c.Next()
	}
}
