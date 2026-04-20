package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// contextKey is a private type for context keys to prevent collisions
// with keys from other packages.
type contextKey string

// RequestIDKey is the key used to store/retrieve the request ID in context.
const RequestIDKey contextKey = "request_id"

// RequestID is a Gin middleware that assigns a unique ID to every request.
// All log lines for a request share this ID, making tracing easy in Grafana.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = generateID()
		}
		c.Set(string(RequestIDKey), id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// SetRequestID embeds a request ID into a Go context.
// Call this in the GraphQL handler before passing context to resolvers/clients.
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

// GetRequestID retrieves the request ID from a context. Returns "" if not set.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}