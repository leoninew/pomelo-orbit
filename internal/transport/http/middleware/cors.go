package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowHeaders = "Authorization, Content-Type"
	corsAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
)

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := parseCORSAllowedOrigins(allowedOrigins)
	return func(next http.Handler) http.Handler {
		if len(origins) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := normalizeCORSOrigin(r.Header.Get("Origin"))
			if !strings.HasPrefix(r.URL.Path, "/api/") || origin == "" {
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

func normalizeCORSOrigin(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func setCORSHeaders(header http.Header, origin string) {
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	header.Set("Access-Control-Allow-Methods", corsAllowMethods)
	header.Add("Vary", "Origin")
}
