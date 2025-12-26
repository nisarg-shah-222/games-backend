package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheckResponse represents the health check response
type HealthCheckResponse struct {
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthCheck handles GET /api/v1/health-check
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthCheckResponse{
		Message:   "Service is healthy",
		Status:    "ok",
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}
