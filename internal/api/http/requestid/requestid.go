package requestid

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextKey = "request_id"
	HeaderName = "X-Request-Id"
)

type contextKey struct{}

var requestIdContextKey contextKey

func WithContext(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, requestIdContextKey, strings.TrimSpace(requestId))
}

func FromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIdContextKey).(string)
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
