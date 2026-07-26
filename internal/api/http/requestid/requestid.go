package requestid

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
)

const (
	ContextKey = "request_id"
	HeaderName = "X-Request-ID"
)

func New() string {
	return ulid.Make().String()
}

func FromContext(c *gin.Context) string {
	value, ok := c.Get(ContextKey)
	if !ok {
		return ""
	}
	requestId, _ := value.(string)
	return strings.TrimSpace(requestId)
}
