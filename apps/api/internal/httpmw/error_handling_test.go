package httpmw

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/gin-gonic/gin"
)

func TestErrorHandling_LogsOnlyFor5xx(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	router := gin.New()
	router.Use(ErrorHandling(logger))

	router.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.GET("/bad", func(c *gin.Context) {
		apierr.BadRequest("invalid").Respond(c)
	})
	router.GET("/boom", func(c *gin.Context) {
		apierr.Internal(c, errors.New("db down"), "failed", "test_boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-Request-Id"); got == "" {
		t.Fatalf("expected request id header on 200 response")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/bad", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	logsBefore500 := logBuffer.String()

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/boom", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	logs := logBuffer.String()
	if strings.Contains(logsBefore500, "request failed") {
		t.Fatalf("did not expect 2xx/4xx requests to be logged as failures")
	}
	if !strings.Contains(logs, "\"msg\":\"request failed\"") {
		t.Fatalf("expected a centralized error log entry for 5xx response")
	}
	if !strings.Contains(logs, "\"op\":\"test_boom\"") {
		t.Fatalf("expected op field in 5xx log entry")
	}
	if !strings.Contains(logs, "\"error\":\"db down\"") {
		t.Fatalf("expected cause error in 5xx log entry")
	}
}

func TestErrorHandling_RecoversPanicAndLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	router := gin.New()
	router.Use(ErrorHandling(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if body["code"] != apierr.CodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR code, got %#v", body["code"])
	}

	logs := logBuffer.String()
	if !strings.Contains(logs, "\"msg\":\"request failed\"") {
		t.Fatalf("expected panic to produce centralized error log")
	}
	if !strings.Contains(logs, "\"op\":\"panic_recovered\"") {
		t.Fatalf("expected panic op in log")
	}
	if !strings.Contains(logs, "\"stack\":") {
		t.Fatalf("expected panic stack in log")
	}
}

func TestErrorHandling_RecoversPanicAfterWriteAndLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	router := gin.New()
	router.Use(ErrorHandling(logger))
	router.GET("/panic-after-write", func(c *gin.Context) {
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString("partial")
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic-after-write", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status to stay 200 after partial write, got %d", rec.Code)
	}

	logs := logBuffer.String()
	if !strings.Contains(logs, "\"msg\":\"request failed\"") {
		t.Fatalf("expected panic-after-write to produce centralized error log")
	}
	if !strings.Contains(logs, "\"op\":\"panic_recovered\"") {
		t.Fatalf("expected panic op in log")
	}
	if !strings.Contains(logs, "\"status\":200") {
		t.Fatalf("expected log status to reflect already-written response")
	}
}
