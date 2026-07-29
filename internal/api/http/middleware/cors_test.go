package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/requestid"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
)

func TestCorsNoopWhenNotConfigured(t *testing.T) {
	recorder := serveCorsRequest(t, nil, http.MethodGet, "/api/health", "https://orbit.preflite.cn")

	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no cors origin header, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCorsAllowsSingleOrigin(t *testing.T) {
	recorder := serveCorsRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodGet, "/api/health", "https://orbit.preflite.cn")

	assertCorsHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCorsAllowsMultipleOrigins(t *testing.T) {
	recorder := serveCorsRequest(t, []string{"https://orbit.preflite.cn", "https://preview.preflite.cn"}, http.MethodGet, "/api/health", "https://preview.preflite.cn")

	assertCorsHeaders(t, recorder, "https://preview.preflite.cn")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCorsNormalizesConfiguredTrailingSlash(t *testing.T) {
	recorder := serveCorsRequest(t, []string{" https://orbit.preflite.cn/ "}, http.MethodGet, "/api/health", "https://orbit.preflite.cn")

	assertCorsHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCorsAllowedPreflightReturnsNoContent(t *testing.T) {
	recorder := serveCorsRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/api/health", "https://orbit.preflite.cn")

	assertCorsHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
}

func TestCorsAllowsApiRootPath(t *testing.T) {
	recorder := serveCorsRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/api", "https://orbit.preflite.cn")

	assertCorsHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
}

func TestCorsUsesConfiguredApiPathPrefixes(t *testing.T) {
	recorder := serveCorsRequestWithPrefixes(t, []string{"https://orbit.preflite.cn"}, []string{"/api", "/graphql"}, http.MethodOptions, "/graphql", "https://orbit.preflite.cn")

	assertCorsHeaders(t, recorder, "https://orbit.preflite.cn")
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
}

func TestCorsDeniedPreflightReturnsForbidden(t *testing.T) {
	recorder := serveCorsRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/api/health", "https://evil.example.test")

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no cors origin header, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	var response transportresponse.ErrorResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != "forbidden" || response.RequestId == "" || recorder.Header().Get(requestid.HeaderName) != response.RequestId {
		t.Fatalf("unexpected forbidden response: %+v", response)
	}
}

func TestCorsIgnoresNonApiPaths(t *testing.T) {
	paths := []string{"/repository", "/apix/health"}
	for _, path := range paths {
		recorder := serveCorsRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, path, "https://orbit.preflite.cn")

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected downstream status 200 for %s, got %d", path, recorder.Code)
		}
		if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("expected no cors origin header for %s, got %s", path, recorder.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestCorsIgnoresRequestsWithoutOrigin(t *testing.T) {
	recorder := serveCorsRequest(t, []string{"https://orbit.preflite.cn"}, http.MethodOptions, "/api/health", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected downstream status 200, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no cors origin header, got %s", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestHasApiPathPrefixUsesConfiguredPrefixes(t *testing.T) {
	prefixes := []string{"/api", "/graphql", "/report-api"}
	cases := []struct {
		path string
		want bool
	}{
		{path: "/api", want: true},
		{path: "/api/health", want: true},
		{path: "/graphql", want: true},
		{path: "/graphql/query", want: true},
		{path: "/report-api/v1", want: true},
		{path: "/apix", want: false},
		{path: "/graphqlx", want: false},
		{path: "/report-api-v2", want: false},
	}
	for _, tc := range cases {
		if got := hasApiPathPrefix(tc.path, prefixes); got != tc.want {
			t.Fatalf("hasAPIPathPrefix(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func serveCorsRequest(t *testing.T, allowedOrigins []string, method string, path string, origin string) *httptest.ResponseRecorder {
	t.Helper()
	return serveCorsRequestWithPrefixes(t, allowedOrigins, []string{"/api"}, method, path, origin)
}

func serveCorsRequestWithPrefixes(t *testing.T, allowedOrigins []string, apiPathPrefixes []string, method string, path string, origin string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestId())
	router.Use(Cors(allowedOrigins, apiPathPrefixes))
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(method, path, nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertCorsHeaders(t *testing.T, recorder *httptest.ResponseRecorder, origin string) {
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
	if recorder.Header().Get("Access-Control-Max-Age") != corsMaxAge {
		t.Fatalf("unexpected max age: %s", recorder.Header().Get("Access-Control-Max-Age"))
	}
	if recorder.Header().Get("Vary") != "Origin" {
		t.Fatalf("unexpected vary header: %s", recorder.Header().Get("Vary"))
	}
	if recorder.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("expected no credentials header, got %s", recorder.Header().Get("Access-Control-Allow-Credentials"))
	}
}
