package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func TestLogRequestIncludesMetadata(t *testing.T) {
	entry, _ := runLoggedRequest(t, http.MethodGet, "/api/test?x=1", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	assertLogValue(t, entry, "msg", "request completed")
	assertLogValue(t, entry, "level", "INFO")
	assertLogValue(t, entry, "method", http.MethodGet)
	assertLogValue(t, entry, "path", "/api/test")
	assertLogValue(t, entry, "uri", "/api/test?x=1")
	assertLogNumber(t, entry, "status", http.StatusCreated)
	assertLogNumber(t, entry, "bytes", len(`{"ok":true}`))
	if _, ok := entry["duration_ms"]; !ok {
		t.Fatal("expected duration_ms")
	}
	if entry["request_id"] == "" {
		t.Fatal("expected request_id")
	}
	if entry["remote_addr"] == "" {
		t.Fatal("expected remote_addr")
	}
	assertLogValue(t, entry, "user_agent", "test-agent")
}

func TestLogRequestKeepsInfoLevelForErrorStatus(t *testing.T) {
	entry, _ := runLoggedRequest(t, http.MethodGet, "/api/error", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "boom"})
	}))

	assertLogValue(t, entry, "level", "INFO")
	assertLogNumber(t, entry, "status", http.StatusInternalServerError)
}

func TestLogRequestRecordsJSONRequestBody(t *testing.T) {
	body := `{"name":"demo"}`
	var handlerBody string
	entry, _ := runLoggedRequest(t, http.MethodPost, "/api/test", "application/json", body, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		handlerBody = buf.String()
		writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	}))

	assertLogValue(t, entry, "request_body", body)
	if handlerBody != body {
		t.Fatalf("expected handler body %q, got %q", body, handlerBody)
	}
}

func TestLogRequestSkipsNonJSONRequestBody(t *testing.T) {
	entry, _ := runLoggedRequest(t, http.MethodPost, "/api/test", "text/plain", "plain text", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	if _, ok := entry["request_body"]; ok {
		t.Fatalf("did not expect request_body: %+v", entry)
	}
}

func TestLogRequestRecordsJSONResponseBody(t *testing.T) {
	responseBody := `{"status":"ok"}`
	entry, recorder := runLoggedRequest(t, http.MethodGet, "/api/test", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(responseBody))
	}))

	assertLogValue(t, entry, "response_body", responseBody)
	if recorder.Body.String() != responseBody {
		t.Fatalf("expected response body %q, got %q", responseBody, recorder.Body.String())
	}
}

func TestLogRequestSkipsNonJSONResponseBody(t *testing.T) {
	entry, _ := runLoggedRequest(t, http.MethodGet, "/api/test", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain text"))
	}))

	if _, ok := entry["response_body"]; ok {
		t.Fatalf("did not expect response_body: %+v", entry)
	}
}

func TestLogRequestTruncatesResponseBody(t *testing.T) {
	responseBody := `{"value":"` + strings.Repeat("好", maxResponseBodyLogLength+10) + `"}`
	entry, _ := runLoggedRequest(t, http.MethodGet, "/api/test", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))

	loggedBody, ok := entry["response_body"].(string)
	if !ok {
		t.Fatalf("expected response_body string: %+v", entry)
	}
	if !strings.HasSuffix(loggedBody, "...") {
		t.Fatalf("expected truncated response body, got %q", loggedBody)
	}
	if len([]rune(loggedBody)) != maxResponseBodyLogLength+3 {
		t.Fatalf("unexpected truncated length: %d", len([]rune(loggedBody)))
	}
}

func TestLogRequestDoesNotTruncateRequestBody(t *testing.T) {
	requestBody := `{"value":"` + strings.Repeat("好", maxResponseBodyLogLength+10) + `"}`
	entry, _ := runLoggedRequest(t, http.MethodPost, "/api/test", "application/json", requestBody, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	assertLogValue(t, entry, "request_body", requestBody)
}

func TestLogRequestLogsRecoveredPanicAsInfo(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	server := Server{logger: logger}
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(server.logRequest)
	router.Use(middleware.Recoverer)
	router.Get("/api/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", recorder.Code)
	}
	entry := decodeLogEntry(t, logBuffer.String())
	assertLogValue(t, entry, "level", "INFO")
	assertLogNumber(t, entry, "status", http.StatusInternalServerError)
}

func runLoggedRequest(t *testing.T, method string, target string, contentType string, body string, handler http.Handler) (map[string]any, *httptest.ResponseRecorder) {
	t.Helper()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	server := Server{logger: logger}
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(server.logRequest)
	router.Handle("/*", handler)

	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("User-Agent", "test-agent")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return decodeLogEntry(t, logBuffer.String()), recorder
}

func decodeLogEntry(t *testing.T, content string) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected log entry")
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("decode log entry: %v\n%s", err, content)
	}
	return entry
}

func assertLogValue(t *testing.T, entry map[string]any, key string, want string) {
	t.Helper()
	if got, ok := entry[key].(string); !ok || got != want {
		t.Fatalf("expected %s=%q, got %#v", key, want, entry[key])
	}
}

func assertLogNumber(t *testing.T, entry map[string]any, key string, want int) {
	t.Helper()
	got, ok := entry[key].(float64)
	if !ok || int(got) != want {
		t.Fatalf("expected %s=%d, got %#v", key, want, entry[key])
	}
}
