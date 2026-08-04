package requestid

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextKey = "request_id"
	HeaderName = "X-Request-ID"
)

func FromContext(c *gin.Context) string {
	value, ok := c.Get(ContextKey)
	if !ok {
		return ""
	}
	requestId, _ := value.(string)
	return strings.TrimSpace(requestId)
}
