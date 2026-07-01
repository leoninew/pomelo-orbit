package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSNoopWhenNotConfigured(t *testing.T) {
	recorder := serveCORSRequest(t, nil, http.MethodGet, "/api/health", "https://orbit.preflite.cn")

	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no cors origin header, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCORSAllowsSingleOrigin(t *testing.T) {
	recorder := serveCORSRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodGet, "/api/health", "https://orbit.preflite.cn")

	assertCORSHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCORSAllowsMultipleOrigins(t *testing.T) {
	recorder := serveCORSRequest(t, []string{"https://orbit.preflite.cn", "https://preview.preflite.cn"}, http.MethodGet, "/api/health", "https://preview.preflite.cn")

	assertCORSHeaders(t, recorder, "https://preview.preflite.cn")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCORSNormalizesConfiguredTrailingSlash(t *testing.T) {
	recorder := serveCORSRequest(t, []string{" https://orbit.preflite.cn/ "}, http.MethodGet, "/api/health", "https://orbit.preflite.cn")

	assertCORSHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCORSAllowedPreflightReturnsNoContent(t *testing.T) {
	recorder := serveCORSRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/api/health", "https://orbit.preflite.cn")

	assertCORSHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
}

func TestCORSDeniedPreflightReturnsForbidden(t *testing.T) {
	recorder := serveCORSRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/api/health", "https://evil.example.test")

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no cors origin header, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSIgnoresNonAPIPaths(t *testing.T) {
	recorder := serveCORSRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/ci/repository", "https://orbit.preflite.cn")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected downstream status 200, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no cors origin header, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSIgnoresRequestsWithoutOrigin(t *testing.T) {
	recorder := serveCORSRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/api/health", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected downstream status 200, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no cors origin header, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func serveCORSRequest(t *testing.T, allowedOrigins []string, method string, path string, origin string) *httptest.ResponseRecorder {
	t.Helper()
	handler := CORS(allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest(method, path, nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func assertCORSHeaders(t *testing.T, recorder *httptest.ResponseRecorder, origin string) {
	t.Helper()
	if recorder.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("unexpected allow origin: %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if recorder.Header().Get("Access-Control-Allow-Headers") != corsAllowHeaders {
		t.Fatalf("unexpected allow headers: %s", recorder.Header().Get("Access-Control-Allow-Headers"))
	}
	if recorder.Header().Get("Access-Control-Allow-Methods") != corsAllowMethods {
		t.Fatalf("unexpected allow methods: %s", recorder.Header().Get("Access-Control-Allow-Methods"))
	}
	if recorder.Header().Get("Vary") != "Origin" {
		t.Fatalf("unexpected vary header: %s", recorder.Header().Get("Vary"))
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("expected no credentials header, got %s", recorder.Header().Get("Access-Control-Allow-Credentials"))
	}
}
