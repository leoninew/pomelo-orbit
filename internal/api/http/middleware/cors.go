package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

const (
	corsAllowHeaders = "Authorization, Content-Type"
	corsAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	corsMaxAge       = "600"
)

func Cors(allowedOrigins []string, apiPathPrefixes []string) gin.HandlerFunc {
	origins := parseCorsAllowedOrigins(allowedOrigins)
	return func(c *gin.Context) {
		if len(origins) == 0 {
			c.Next()
			return
		}

		origin := normalizeCorsOrigin(c.GetHeader("Origin"))
		if !isApiPath(c.Request.URL.Path, apiPathPrefixes) || origin == "" {
			c.Next()
			return
		}

		if origins[origin] {
			setCorsHeaders(c.Writer.Header(), origin)
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		if c.Request.Method == http.MethodOptions {
			transportresponse.WriteError(c, apperror.New(apperror.KindForbidden, "CORS origin is not allowed."))
			return
		}
		c.Next()
	}
}

func parseCorsAllowedOrigins(values []string) map[string]bool {
	origins := make(map[string]bool)
	for _, item := range values {
		origin := normalizeCorsOrigin(item)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

func isApiPath(path string, prefixes []string) bool {
	return hasApiPathPrefix(path, prefixes)
}

func hasApiPathPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func normalizeCorsOrigin(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func setCorsHeaders(header http.Header, origin string) {
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	header.Set("Access-Control-Allow-Methods", corsAllowMethods)
	header.Set("Access-Control-Max-Age", corsMaxAge)
	header.Add("Vary", "Origin")
}
