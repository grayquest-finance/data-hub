package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"data-hub/config"

	"github.com/gin-gonic/gin"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
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

// APIKeyAuthAllowIntrospection skips API key auth for introspection-only
// queries, so clients can discover the schema without credentials.
// All other queries still require a valid X-Api-Key.
func APIKeyAuthAllowIntrospection(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	authed := APIKeyAuth(cfg, logger)
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err == nil {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		var req struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(body, &req)

		if isIntrospectionOnly(req.Query) {
			c.Next()
			return
		}
		authed(c)
	}
}

func isIntrospectionOnly(q string) bool {
	if strings.TrimSpace(q) == "" {
		return false
	}
	doc, err := parser.ParseQuery(&ast.Source{Input: q})
	if err != nil {
		return false
	}
	for _, op := range doc.Operations {
		for _, sel := range op.SelectionSet {
			field, ok := sel.(*ast.Field)
			if !ok {
				return false
			}
			if !strings.HasPrefix(field.Name, "__") {
				return false
			}
		}
	}
	return true
}