package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowHeaders = "Authorization, Content-Type"
	corsAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	corsMaxAge       = "600"
)

func CORS(allowedOrigins []string, apiPathPrefixes []string) func(http.Handler) http.Handler {
	origins := parseCORSAllowedOrigins(allowedOrigins)
	return func(next http.Handler) http.Handler {
		if len(origins) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := normalizeCORSOrigin(r.Header.Get("Origin"))
			if !isAPIPath(r.URL.Path, apiPathPrefixes) || origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if origins[origin] {
				setCORSHeaders(w.Header(), origin)
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
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
