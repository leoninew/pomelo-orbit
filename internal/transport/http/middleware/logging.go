package middleware

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const TruncatedBodySuffix = "..."

type LogRequestConfig struct {
	BodyEnabled          bool
	BodyMaxBytes         int
	SkipAssets200Enabled bool
}

func LogRequest(logger *slog.Logger, cfg LogRequestConfig) func(http.Handler) http.Handler {
	if cfg.BodyEnabled && cfg.BodyMaxBytes <= 0 {
		panic("http body max bytes must be positive")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			requestAttrs := requestLogAttrs(r)
			requestBody, bodyErr := readRequestBodyForLog(r, cfg)

			startedAttrs := append([]any{}, requestAttrs...)
			if requestBody != "" {
				startedAttrs = append(startedAttrs, "request_body", requestBody)
			}
			if bodyErr != nil {
				startedAttrs = append(startedAttrs, "body_read_error", bodyErr.Error())
			}
			delayStartedLog := cfg.SkipAssets200Enabled && isSkippableAssetPath(r.URL.Path)
			if !delayStartedLog {
				logger.Info("request started", startedAttrs...)
			}

			responseWriter := newLoggingResponseWriter(w, cfg)
			next.ServeHTTP(responseWriter, r)
			if shouldSkipRequestLog(r, responseWriter.Status(), cfg) {
				return
			}
			if delayStartedLog {
				logger.Info("request started", startedAttrs...)
			}

			completedAttrs := append([]any{}, requestAttrs...)
			completedAttrs = append(completedAttrs,
				"status", responseWriter.Status(),
				"bytes", responseWriter.BytesWritten(),
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
			if responseBody := responseWriter.Body(); responseBody != "" {
				completedAttrs = append(completedAttrs, "response_body", responseBody)
			}
			logger.Info("request completed", completedAttrs...)
		})
	}
}

func requestLogAttrs(r *http.Request) []any {
	return []any{
		"method", r.Method,
		"path", r.URL.Path,
		"uri", r.URL.RequestURI(),
		"request_id", chimiddleware.GetReqID(r.Context()),
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	}
}

func shouldSkipRequestLog(r *http.Request, status int, cfg LogRequestConfig) bool {
	return cfg.SkipAssets200Enabled && status == http.StatusOK && isSkippableAssetPath(r.URL.Path)
}

func isSkippableAssetPath(requestPath string) bool {
	requestPath = path.Clean("/" + requestPath)
	if !strings.HasPrefix(requestPath, "/assets/") {
		return false
	}
	switch strings.ToLower(path.Ext(requestPath)) {
	case ".js", ".css", ".html":
		return true
	default:
		return false
	}
}

func readRequestBodyForLog(r *http.Request, cfg LogRequestConfig) (string, error) {
	if !cfg.BodyEnabled || !isJSONContentType(r.Header.Get("Content-Type")) || r.Body == nil {
		return "", nil
	}
	body := r.Body
	loggedBytes, err := io.ReadAll(io.LimitReader(body, int64(cfg.BodyMaxBytes)+1))
	r.Body = &prefixReadCloser{reader: io.MultiReader(bytes.NewReader(loggedBytes), body), closer: body}
	if err != nil {
		return "", err
	}
	if len(loggedBytes) == 0 {
		return "", nil
	}
	return truncateLogBody(loggedBytes, cfg.BodyMaxBytes), nil
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func truncateLogBody(value []byte, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return string(value)
	}
	value = value[:limit]
	for len(value) > 0 && !utf8.Valid(value) {
		value = value[:len(value)-1]
	}
	return string(value) + TruncatedBodySuffix
}

type prefixReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func (r *prefixReadCloser) Read(data []byte) (int, error) {
	return r.reader.Read(data)
}

func (r *prefixReadCloser) Close() error {
	return r.closer.Close()
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status       int
	bytes        int
	body         bytes.Buffer
	bodyEnabled  bool
	bodyMaxBytes int
}

func newLoggingResponseWriter(w http.ResponseWriter, cfg LogRequestConfig) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w, bodyEnabled: cfg.BodyEnabled, bodyMaxBytes: cfg.BodyMaxBytes}
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.captureBody(data)
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func (w *loggingResponseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *loggingResponseWriter) BytesWritten() int {
	return w.bytes
}

func (w *loggingResponseWriter) Body() string {
	if !w.bodyEnabled || w.body.Len() == 0 {
		return ""
	}
	return truncateLogBody(w.body.Bytes(), w.bodyMaxBytes)
}

func (w *loggingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	if w.status == 0 {
		w.status = http.StatusSwitchingProtocols
	}
	return hijacker.Hijack()
}

func (w *loggingResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *loggingResponseWriter) captureBody(data []byte) {
	if !w.bodyEnabled || !isJSONContentType(w.Header().Get("Content-Type")) || len(data) == 0 {
		return
	}
	limit := w.bodyMaxBytes + 1
	if w.body.Len() >= limit {
		return
	}
	remaining := limit - w.body.Len()
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.body.Write(data)
}

var _ http.Flusher = (*loggingResponseWriter)(nil)
var _ http.Hijacker = (*loggingResponseWriter)(nil)
var _ http.Pusher = (*loggingResponseWriter)(nil)
var _ interface{ Unwrap() http.ResponseWriter } = (*loggingResponseWriter)(nil)
