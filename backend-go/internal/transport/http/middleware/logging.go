package middleware

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const MaxResponseBodyLogLength = 4096

func LogRequest(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			requestBody, bodyErr := readRequestBodyForLog(r)
			responseWriter := newLoggingResponseWriter(w)

			next.ServeHTTP(responseWriter, r)

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"uri", r.URL.RequestURI(),
				"status", responseWriter.Status(),
				"bytes", responseWriter.BytesWritten(),
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"request_id", chimiddleware.GetReqID(r.Context()),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			}
			if requestBody != "" {
				attrs = append(attrs, "request_body", requestBody)
			}
			if responseBody := responseWriter.Body(); responseBody != "" {
				attrs = append(attrs, "response_body", responseBody)
			}
			if bodyErr != nil {
				attrs = append(attrs, "body_read_error", bodyErr.Error())
			}

			logger.Info("request completed", attrs...)
		})
	}
}

func readRequestBodyForLog(r *http.Request) (string, error) {
	if !isJSONContentType(r.Header.Get("Content-Type")) || r.Body == nil {
		return "", nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) == 0 {
		return "", nil
	}
	return string(body), nil
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func truncateLogBody(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
	body   bytes.Buffer
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w}
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
	if w.body.Len() == 0 {
		return ""
	}
	return truncateLogBody(w.body.String(), MaxResponseBodyLogLength)
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
	return hijacker.Hijack()
}

func (w *loggingResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *loggingResponseWriter) captureBody(data []byte) {
	if !isJSONContentType(w.Header().Get("Content-Type")) || len(data) == 0 {
		return
	}
	limit := MaxResponseBodyLogLength * 4
	if w.body.Len() >= limit {
		return
	}
	remaining := limit - w.body.Len()
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.body.Write(data)
}
