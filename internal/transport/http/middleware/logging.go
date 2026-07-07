package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDKey        = "request_id"
	RequestIDHeader     = "X-Request-Id"
	TruncatedBodySuffix = "..."
)

type LogRequestConfig struct {
	BodyEnabled          bool
	BodyMaxBytes         int
	SkipAssets200Enabled bool
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(RequestIDKey, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)
		c.Next()
	}
}

func RequestIDFromContext(c *gin.Context) string {
	value, ok := c.Get(RequestIDKey)
	if !ok {
		return ""
	}
	requestID, _ := value.(string)
	return requestID
}

func RealIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if realIP := strings.TrimSpace(parts[0]); realIP != "" {
				c.Request.RemoteAddr = realIP
			}
		} else if realIP := strings.TrimSpace(c.GetHeader("X-Real-IP")); realIP != "" {
			c.Request.RemoteAddr = realIP
		}
		c.Next()
	}
}

func LogRequest(logger *slog.Logger, cfg LogRequestConfig) gin.HandlerFunc {
	if cfg.BodyEnabled && cfg.BodyMaxBytes <= 0 {
		panic("http body max bytes must be positive")
	}
	return func(c *gin.Context) {
		startedAt := time.Now()
		requestAttrs := requestLogAttrs(c)
		requestBody, bodyErr := readRequestBodyForLog(c.Request, cfg)

		startedAttrs := append([]any{}, requestAttrs...)
		if requestBody != "" {
			startedAttrs = append(startedAttrs, "request_body", requestBody)
		}
		if bodyErr != nil {
			startedAttrs = append(startedAttrs, "body_read_error", bodyErr.Error())
		}
		delayStartedLog := cfg.SkipAssets200Enabled && isSkippableAssetPath(c.Request.URL.Path)
		if !delayStartedLog {
			logger.Info("request started", startedAttrs...)
		}

		bodyWriter := &bodyLogWriter{ResponseWriter: c.Writer, cfg: cfg}
		c.Writer = bodyWriter
		c.Next()

		status := c.Writer.Status()
		if shouldSkipRequestLog(c.Request, status, cfg) {
			return
		}
		if delayStartedLog {
			logger.Info("request started", startedAttrs...)
		}

		completedAttrs := append([]any{}, requestAttrs...)
		completedAttrs = append(completedAttrs,
			"status", status,
			"bytes", c.Writer.Size(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
		if responseBody := bodyWriter.Body(); responseBody != "" {
			completedAttrs = append(completedAttrs, "response_body", responseBody)
		}
		logger.Info("request completed", completedAttrs...)
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	_ = logger
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

func requestLogAttrs(c *gin.Context) []any {
	r := c.Request
	return []any{
		"method", r.Method,
		"path", r.URL.Path,
		"uri", r.URL.RequestURI(),
		"request_id", RequestIDFromContext(c),
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

type bodyLogWriter struct {
	gin.ResponseWriter
	cfg  LogRequestConfig
	body bytes.Buffer
}

func (w *bodyLogWriter) Write(data []byte) (int, error) {
	w.captureBody(data)
	return w.ResponseWriter.Write(data)
}

func (w *bodyLogWriter) WriteString(data string) (int, error) {
	w.captureBody([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *bodyLogWriter) Body() string {
	if !w.cfg.BodyEnabled || w.body.Len() == 0 {
		return ""
	}
	return truncateLogBody(w.body.Bytes(), w.cfg.BodyMaxBytes)
}

func (w *bodyLogWriter) captureBody(data []byte) {
	if !w.cfg.BodyEnabled || !isJSONContentType(w.Header().Get("Content-Type")) || len(data) == 0 {
		return
	}
	limit := w.cfg.BodyMaxBytes + 1
	if w.body.Len() >= limit {
		return
	}
	remaining := limit - w.body.Len()
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.body.Write(data)
}
