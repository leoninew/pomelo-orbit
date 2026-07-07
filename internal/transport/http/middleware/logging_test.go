package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const testBodyMaxBytes = 32

func TestLogRequestIncludesMetadata(t *testing.T) {
	entries, _ := runLoggedRequest(t, http.MethodGet, "/api/test?x=1", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogValue(t, started, "level", "INFO")
	assertLogValue(t, started, "method", http.MethodGet)
	assertLogValue(t, started, "path", "/api/test")
	assertLogValue(t, started, "uri", "/api/test?x=1")
	assertLogMissing(t, started, "query")
	if started["request_id"] == "" {
		t.Fatal("expected started request_id")
	}
	if started["remote_addr"] == "" {
		t.Fatal("expected started remote_addr")
	}
	assertLogValue(t, started, "user_agent", "test-agent")
	assertLogMissing(t, started, "status")
	assertLogMissing(t, started, "bytes")
	assertLogMissing(t, started, "duration_ms")
	assertLogMissing(t, started, "request_body")
	assertLogMissing(t, started, "response_body")

	assertLogValue(t, completed, "level", "INFO")
	assertLogValue(t, completed, "method", http.MethodGet)
	assertLogValue(t, completed, "path", "/api/test")
	assertLogValue(t, completed, "uri", "/api/test?x=1")
	assertLogMissing(t, completed, "query")
	assertLogNumber(t, completed, "status", http.StatusCreated)
	assertLogNumber(t, completed, "bytes", len(`{"ok":true}`))
	if _, ok := completed["duration_ms"]; !ok {
		t.Fatal("expected duration_ms")
	}
	if completed["request_id"] == "" {
		t.Fatal("expected completed request_id")
	}
	if completed["remote_addr"] == "" {
		t.Fatal("expected completed remote_addr")
	}
	assertLogValue(t, completed, "user_agent", "test-agent")
	assertLogValue(t, completed, "response_body", `{"ok":true}`)
}

func TestLogRequestKeepsInfoLevelForErrorStatus(t *testing.T) {
	entries, _ := runLoggedRequest(t, http.MethodGet, "/api/error", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, http.StatusInternalServerError, map[string]string{"detail": "boom"})
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogValue(t, started, "level", "INFO")
	assertLogValue(t, completed, "level", "INFO")
	assertLogNumber(t, completed, "status", http.StatusInternalServerError)
}

func TestLogRequestSkipsAssets200WhenEnabled(t *testing.T) {
	cfg := LogRequestConfig{BodyEnabled: true, BodyMaxBytes: testBodyMaxBytes, SkipAssets200Enabled: true}
	cases := []string{
		"/assets/app.js",
		"/assets/app.css?v=1",
		"/assets/page.html",
	}
	for _, target := range cases {
		content, recorder := runLoggedRequestContentWithConfig(t, cfg, http.MethodGet, target, "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("asset"))
		}))
		if content != "" {
			t.Fatalf("expected no log entries for %s, got %s", target, content)
		}
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200 for %s, got %d", target, recorder.Code)
		}
	}
}

func TestLogRequestKeepsNonSkippedAssetLogs(t *testing.T) {
	cfg := LogRequestConfig{BodyEnabled: true, BodyMaxBytes: testBodyMaxBytes, SkipAssets200Enabled: true}
	cases := []struct {
		name   string
		target string
		status int
	}{
		{name: "asset js not found", target: "/assets/app.js", status: http.StatusNotFound},
		{name: "asset js not modified", target: "/assets/app.js", status: http.StatusNotModified},
		{name: "asset png ok", target: "/assets/app.png", status: http.StatusOK},
		{name: "asset prefix mismatch", target: "/assets-old/app.js", status: http.StatusOK},
		{name: "asserts typo", target: "/asserts/app.js", status: http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries, _ := runLoggedRequestWithConfig(t, cfg, http.MethodGet, tc.target, "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			_, completed := assertStartedAndCompleted(t, entries)
			assertLogNumber(t, completed, "status", tc.status)
		})
	}
}

func TestLogRequestKeepsAssets200WhenSkipDisabled(t *testing.T) {
	cfg := LogRequestConfig{BodyEnabled: true, BodyMaxBytes: testBodyMaxBytes, SkipAssets200Enabled: false}
	entries, _ := runLoggedRequestWithConfig(t, cfg, http.MethodGet, "/assets/app.js", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	_, completed := assertStartedAndCompleted(t, entries)
	assertLogNumber(t, completed, "status", http.StatusOK)
}

func TestLogRequestSkipsBodiesWhenDisabled(t *testing.T) {
	body := `{"name":"demo"}`
	entries, _ := runLoggedRequestWithConfig(t, LogRequestConfig{BodyEnabled: false, BodyMaxBytes: testBodyMaxBytes}, http.MethodPost, "/api/test", "application/json", body, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogMissing(t, started, "request_body")
	assertLogMissing(t, started, "response_body")
	assertLogMissing(t, completed, "request_body")
	assertLogMissing(t, completed, "response_body")
}

func TestLogRequestRecordsJSONRequestBody(t *testing.T) {
	body := `{"name":"demo"}`
	var handlerBody string
	entries, _ := runLoggedRequest(t, http.MethodPost, "/api/test", "application/json", body, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		handlerBody = buf.String()
		writeTestJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogValue(t, started, "request_body", body)
	assertLogMissing(t, started, "response_body")
	assertLogMissing(t, completed, "request_body")
	if handlerBody != body {
		t.Fatalf("expected handler body %q, got %q", body, handlerBody)
	}
}

func TestLogRequestSkipsNonJSONRequestBody(t *testing.T) {
	entries, _ := runLoggedRequest(t, http.MethodPost, "/api/test", "text/plain", "plain text", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogMissing(t, started, "request_body")
	assertLogMissing(t, completed, "request_body")
}

func TestLogRequestRecordsJSONResponseBody(t *testing.T) {
	responseBody := `{"status":"ok"}`
	entries, recorder := runLoggedRequest(t, http.MethodGet, "/api/test", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(responseBody))
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogMissing(t, started, "response_body")
	assertLogMissing(t, completed, "request_body")
	assertLogValue(t, completed, "response_body", responseBody)
	if recorder.Body.String() != responseBody {
		t.Fatalf("expected response body %q, got %q", responseBody, recorder.Body.String())
	}
}

func TestLogRequestSplitsRequestAndResponseBodies(t *testing.T) {
	requestBody := `{"name":"demo"}`
	responseBody := `{"status":"ok"}`
	entries, _ := runLoggedRequest(t, http.MethodPost, "/api/test", "application/json", requestBody, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogValue(t, started, "request_body", requestBody)
	assertLogMissing(t, started, "response_body")
	assertLogMissing(t, completed, "request_body")
	assertLogValue(t, completed, "response_body", responseBody)
}

func TestLogRequestSkipsNonJSONResponseBody(t *testing.T) {
	entries, _ := runLoggedRequest(t, http.MethodGet, "/api/test", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain text"))
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	assertLogMissing(t, started, "response_body")
	assertLogMissing(t, completed, "response_body")
}

func TestLogRequestTruncatesResponseBodyByConfiguredBytes(t *testing.T) {
	responseBody := `{"value":"` + strings.Repeat("好", testBodyMaxBytes) + `"}`
	entries, _ := runLoggedRequest(t, http.MethodGet, "/api/test", "", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	_, completed := assertStartedAndCompleted(t, entries)

	loggedBody, ok := completed["response_body"].(string)
	if !ok {
		t.Fatalf("expected response_body string: %+v", completed)
	}
	assertTruncatedBody(t, loggedBody)
}

func TestLogRequestTruncatesRequestBodyByConfiguredBytesAndRestoresBody(t *testing.T) {
	requestBody := `{"value":"` + strings.Repeat("好", testBodyMaxBytes) + `"}`
	var handlerBody string
	entries, _ := runLoggedRequest(t, http.MethodPost, "/api/test", "application/json", requestBody, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		handlerBody = buf.String()
		w.WriteHeader(http.StatusNoContent)
	}))
	started, completed := assertStartedAndCompleted(t, entries)

	loggedBody, ok := started["request_body"].(string)
	if !ok {
		t.Fatalf("expected request_body string: %+v", started)
	}
	assertLogMissing(t, completed, "request_body")
	assertTruncatedBody(t, loggedBody)
	if handlerBody != requestBody {
		t.Fatalf("expected handler body %q, got %q", requestBody, handlerBody)
	}
}

func TestLogRequestLogsRecoveredPanicAsInfo(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(LogRequest(logger, testLogRequestConfig()))
	router.Use(chimiddleware.Recoverer)
	router.Get("/api/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", recorder.Code)
	}
	entries := decodeLogEntries(t, logBuffer.String())
	started, completed := assertStartedAndCompleted(t, entries)
	assertLogValue(t, started, "level", "INFO")
	assertLogValue(t, completed, "level", "INFO")
	assertLogNumber(t, completed, "status", http.StatusInternalServerError)
}

func TestLoggingResponseWriterExposesOptionalInterfaces(t *testing.T) {
	writer := newLoggingResponseWriter(httptest.NewRecorder(), testLogRequestConfig())
	if _, ok := any(writer).(http.Flusher); !ok {
		t.Fatal("expected http.Flusher")
	}
	if _, ok := any(writer).(http.Hijacker); !ok {
		t.Fatal("expected http.Hijacker")
	}
	if _, ok := any(writer).(http.Pusher); !ok {
		t.Fatal("expected http.Pusher")
	}
	if got := writer.Unwrap(); got == nil {
		t.Fatal("expected Unwrap to return underlying writer")
	}
}

func TestLoggingResponseWriterHijackSetsSwitchingProtocols(t *testing.T) {
	underlying := &hijackableResponseWriter{}
	writer := newLoggingResponseWriter(underlying, testLogRequestConfig())
	if _, _, err := writer.Hijack(); err != nil {
		t.Fatalf("Hijack() error = %v", err)
	}
	if got := writer.Status(); got != http.StatusSwitchingProtocols {
		t.Fatalf("expected status %d, got %d", http.StatusSwitchingProtocols, got)
	}
}

func TestLoggingResponseWriterUnwrap(t *testing.T) {
	underlying := httptest.NewRecorder()
	writer := newLoggingResponseWriter(underlying, testLogRequestConfig())
	if got := writer.Unwrap(); got != underlying {
		t.Fatalf("expected Unwrap to return underlying writer, got %#v", got)
	}
}

func runLoggedRequest(t *testing.T, method string, target string, contentType string, body string, handler http.Handler) ([]map[string]any, *httptest.ResponseRecorder) {
	t.Helper()
	return runLoggedRequestWithConfig(t, testLogRequestConfig(), method, target, contentType, body, handler)
}

func runLoggedRequestWithConfig(t *testing.T, cfg LogRequestConfig, method string, target string, contentType string, body string, handler http.Handler) ([]map[string]any, *httptest.ResponseRecorder) {
	t.Helper()
	content, recorder := runLoggedRequestContentWithConfig(t, cfg, method, target, contentType, body, handler)
	return decodeLogEntries(t, content), recorder
}

func runLoggedRequestContentWithConfig(t *testing.T, cfg LogRequestConfig, method string, target string, contentType string, body string, handler http.Handler) (string, *httptest.ResponseRecorder) {
	t.Helper()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(LogRequest(logger, cfg))
	router.Handle("/*", handler)

	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("User-Agent", "test-agent")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return logBuffer.String(), recorder
}

func testLogRequestConfig() LogRequestConfig {
	return LogRequestConfig{BodyEnabled: true, BodyMaxBytes: testBodyMaxBytes}
}

func decodeLogEntries(t *testing.T, content string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected log entry")
	}
	entries := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("decode log entry: %v\n%s", err, content)
		}
		entries = append(entries, entry)
	}
	return entries
}

func assertStartedAndCompleted(t *testing.T, entries []map[string]any) (map[string]any, map[string]any) {
	t.Helper()
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d: %+v", len(entries), entries)
	}
	started := entries[0]
	completed := entries[1]
	assertLogValue(t, started, "msg", "request started")
	assertLogValue(t, completed, "msg", "request completed")
	if started["request_id"] != completed["request_id"] {
		t.Fatalf("expected matching request_id, got started=%#v completed=%#v", started["request_id"], completed["request_id"])
	}
	return started, completed
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

func assertLogMissing(t *testing.T, entry map[string]any, key string) {
	t.Helper()
	if _, ok := entry[key]; ok {
		t.Fatalf("did not expect %s: %+v", key, entry)
	}
}

func assertTruncatedBody(t *testing.T, body string) {
	t.Helper()
	if !strings.HasSuffix(body, "...") {
		t.Fatalf("expected truncated body, got %q", body)
	}
	if !utf8.ValidString(body) {
		t.Fatalf("expected valid utf-8 body, got %q", body)
	}
	withoutSuffix := strings.TrimSuffix(body, "...")
	if len([]byte(withoutSuffix)) > testBodyMaxBytes {
		t.Fatalf("expected body within %d bytes, got %d", testBodyMaxBytes, len([]byte(withoutSuffix)))
	}
}

func writeTestJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type hijackableResponseWriter struct {
	header http.Header
	status int
}

func (w *hijackableResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *hijackableResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *hijackableResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(data), nil
}

func (w *hijackableResponseWriter) Flush() {}

func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func (w *hijackableResponseWriter) Push(string, *http.PushOptions) error {
	return nil
}
