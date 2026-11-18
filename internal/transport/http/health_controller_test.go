package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	loggerpkg "auth_service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type healthDatabasePinger struct {
	err error
}

func (p healthDatabasePinger) PingContext(context.Context) error {
	return p.err
}

type healthTestLogger struct{}

func (healthTestLogger) Info(string, any, string, string, ...slog.Attr)  {}
func (healthTestLogger) Warn(string, any, string, string, ...slog.Attr)  {}
func (healthTestLogger) Error(string, any, string, string, ...slog.Attr) {}
func (healthTestLogger) NewCorrelationID() string                        { return "test-correlation-id" }

var _ loggerpkg.Logger = healthTestLogger{}

func TestHealthControllerRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		pingError  error
		statusCode int
	}{
		{name: "database is available", statusCode: http.StatusOK},
		{name: "database is unavailable", pingError: errors.New("database down"), statusCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			controller := NewHealthController(
				healthDatabasePinger{err: tt.pingError},
				"auth-service",
				"1.0.0",
				healthTestLogger{},
			)
			controller.Route(router.Group("/api/v1"))

			request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.statusCode {
				t.Fatalf("GET /api/v1/health status = %d, want %d", response.Code, tt.statusCode)
			}
		})
	}
}
