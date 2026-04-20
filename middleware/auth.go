package middleware

import (
	"net/http"

	"data-hub/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// APIKeyAuth validates the X-Api-Key header on every request.
// Returns HTTP 403 if the key is missing or wrong.
func APIKeyAuth(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Api-Key")
		if key == "" || key != cfg.XAPIKey {
			logger.Warn("auth rejected",
				zap.String("request_id", c.GetString(string(RequestIDKey))),
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "unauthorized",
				"message": "invalid or missing API key",
			})
			return
		}
		c.Next()
	}
}