package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

const (
	corsAllowHeaders = "Authorization, Content-Type"
	corsAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	corsMaxAge       = "600"
)

func CORS(allowedOrigins []string, apiPathPrefixes []string) gin.HandlerFunc {
	origins := parseCORSAllowedOrigins(allowedOrigins)
	return func(c *gin.Context) {
		if len(origins) == 0 {
			c.Next()
			return
		}

		origin := normalizeCORSOrigin(c.GetHeader("Origin"))
		if !isAPIPath(c.Request.URL.Path, apiPathPrefixes) || origin == "" {
			c.Next()
			return
		}

		if origins[origin] {
			setCORSHeaders(c.Writer.Header(), origin)
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

func parseCORSAllowedOrigins(values []string) map[string]bool {
	origins := make(map[string]bool)
	for _, item := range values {
		origin := normalizeCORSOrigin(item)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

func isAPIPath(path string, prefixes []string) bool {
	return hasAPIPathPrefix(path, prefixes)
}

func hasAPIPathPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func normalizeCORSOrigin(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func setCORSHeaders(header http.Header, origin string) {
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	header.Set("Access-Control-Allow-Methods", corsAllowMethods)
	header.Set("Access-Control-Max-Age", corsMaxAge)
	header.Add("Vary", "Origin")
}
