package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bmardale/skjul/internal/config"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func testConfig(requests int, window time.Duration) config.RateLimitConfig {
	lc := config.LimitConfig{Requests: requests, Window: window, Burst: requests}
	return config.RateLimitConfig{
		Enabled:         true,
		Register:        lc,
		LoginChallenge:  lc,
		Login:           lc,
		PasteGet:        lc,
		EntryTTL:        time.Hour,
		CleanupInterval: time.Hour,
	}
}

func setupRouter(cfg config.RateLimitConfig) *gin.Engine {
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(SensitivePaths(cfg))
	v1.POST("/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	v1.POST("/auth/register", func(c *gin.Context) { c.Status(http.StatusOK) })
	v1.POST("/auth/login/challenge", func(c *gin.Context) { c.Status(http.StatusOK) })
	v1.GET("/pastes/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	v1.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestRateLimitBlocksAfterBurst(t *testing.T) {
	router := setupRouter(testConfig(2, time.Hour))

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestRateLimitHeadersOnSuccess(t *testing.T) {
	router := setupRouter(testConfig(5, time.Hour))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if v := w.Header().Get("X-RateLimit-Limit"); v != "5" {
		t.Fatalf("expected X-RateLimit-Limit=5, got %q", v)
	}
	if v := w.Header().Get("X-RateLimit-Remaining"); v == "" {
		t.Fatal("expected X-RateLimit-Remaining header to be set")
	}
}

func TestRateLimitHeadersOn429(t *testing.T) {
	router := setupRouter(testConfig(1, time.Hour))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if v := w.Header().Get("X-RateLimit-Remaining"); v != "0" {
		t.Fatalf("expected X-RateLimit-Remaining=0, got %q", v)
	}
	if v := w.Header().Get("Retry-After"); v == "" {
		t.Fatal("expected Retry-After header on 429")
	}
}

func TestUnmatchedPathNotRateLimited(t *testing.T) {
	router := setupRouter(testConfig(1, time.Hour))

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
		if v := w.Header().Get("X-RateLimit-Limit"); v != "" {
			t.Fatalf("expected no rate limit header on unmatched path, got %q", v)
		}
	}
}

func TestDisabledRateLimit(t *testing.T) {
	cfg := testConfig(1, time.Hour)
	cfg.Enabled = false
	router := setupRouter(cfg)

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200 when disabled, got %d", i+1, w.Code)
		}
	}
}

func TestRulesAreIsolated(t *testing.T) {
	router := setupRouter(testConfig(1, time.Hour))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/register", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register should have its own bucket, got %d", w.Code)
	}
}

func TestDifferentIPsHaveSeparateBuckets(t *testing.T) {
	router := setupRouter(testConfig(1, time.Hour))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	req.RemoteAddr = "1.1.1.1:1234"
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", nil)
	req.RemoteAddr = "2.2.2.2:1234"
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("different IP should have its own bucket, got %d", w.Code)
	}
}

func TestPasteGetIsRateLimited(t *testing.T) {
	router := setupRouter(testConfig(1, time.Hour))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/pastes/abc123", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/pastes/def456", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for paste_get after burst, got %d", w.Code)
	}
}
