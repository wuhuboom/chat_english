package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go-fly-muti/models"
)

// GetHealth reports whether the HTTP process is alive.
func GetHealth(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetReadiness reports whether the service can reach its database.
func GetReadiness(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if models.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	sqlDB := models.DB.DB()
	if sqlDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
