package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	middleware "auth_service/internal/transport/http/middleware_http"
	loggerpkg "auth_service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type databasePinger interface {
	PingContext(context.Context) error
}

type HealthController struct {
	database databasePinger
	service  string
	version  string
	logger   loggerpkg.Logger
}

type healthCheck struct {
	ComponentID   string `json:"componentId"`
	ComponentType string `json:"componentType"`
	Status        string `json:"status"`
	Output        string `json:"output,omitempty"`
}

type healthResponse struct {
	Status      string                   `json:"status"`
	ServiceID   string                   `json:"serviceId"`
	Description string                   `json:"description"`
	Version     string                   `json:"version,omitempty"`
	Checks      map[string][]healthCheck `json:"checks"`
}

func NewHealthController(database databasePinger, service, version string, logger loggerpkg.Logger) *HealthController {
	return &HealthController{
		database: database,
		service:  service,
		version:  version,
		logger:   logger,
	}
}

func (c *HealthController) Route(rg *gin.RouterGroup) {
	rg.GET("/health", c.Check)
}

func (c *HealthController) Check(ctx *gin.Context) {
	response := healthResponse{
		Status:      "pass",
		ServiceID:   c.service,
		Description: c.service + " health check",
		Version:     c.version,
		Checks: map[string][]healthCheck{
			"database": {{
				ComponentID:   "database",
				ComponentType: "datastore",
				Status:        "pass",
			}},
		},
	}

	pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
	defer cancel()
	if err := c.database.PingContext(pingCtx); err != nil {
		correlationID := middleware.GetCorrelationID(ctx)
		c.logger.Error(correlationID, "", "anonymous", fmt.Sprintf("health check database ping failed: %v", err))
		response.Status = "fail"
		response.Checks["database"][0].Status = "fail"
		response.Checks["database"][0].Output = "database is unavailable"
		ctx.JSON(http.StatusServiceUnavailable, response)
		return
	}

	ctx.JSON(http.StatusOK, response)
}
