package requestid

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextKey = "request_id"
	HeaderName = "X-Request-ID"
)

type contextKey struct{}

var requestIDContextKey contextKey

func WithContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, strings.TrimSpace(requestID))
}

func FromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey).(string)
	return strings.TrimSpace(value)
}

func FromGinContext(c *gin.Context) string {
	value, ok := c.Get(ContextKey)
	if !ok {
		return FromContext(c.Request.Context())
	}
	requestId, _ := value.(string)
	return strings.TrimSpace(requestId)
}
