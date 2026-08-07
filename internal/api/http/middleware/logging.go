package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/requestid"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
)

const (
	RequestIdKey        = requestid.ContextKey
	RequestIdHeader     = requestid.HeaderName
	TruncatedBodySuffix = "..."
)

type LogRequestConfig struct {
	Enabled           bool
	RequestBodyLimit  int
	ResponseBodyLimit int
	SkipAssetEnabled  bool
}

func RequestId() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestId := strings.TrimSpace(c.GetHeader(RequestIdHeader))
		if requestId == "" {
			requestId = idutil.NewId()
		}
		c.Set(RequestIdKey, requestId)
		c.Request = c.Request.WithContext(requestid.WithContext(c.Request.Context(), requestId))
		c.Writer.Header().Set(RequestIdHeader, requestId)
		c.Next()
	}
}

func RequestIdFromContext(c *gin.Context) string {
	return requestid.FromGinContext(c)
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
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return func(c *gin.Context) {
		startedAt := time.Now()
		requestAttrs := requestLogAttrs(c)
		requestBody, bodyErr := readRequestBodyForLog(c.Request, cfg.RequestBodyLimit)

		startedAttrs := append([]any{}, requestAttrs...)
		if requestBody != "" {
			startedAttrs = append(startedAttrs, "request_body", requestBody)
		}
		if bodyErr != nil {
			startedAttrs = append(startedAttrs, "body_read_error", bodyErr.Error())
		}
		delayStartedLog := cfg.SkipAssetEnabled && isSkippableAssetPath(c.Request.URL.Path)
		if !delayStartedLog {
			logger.Info("request started", startedAttrs...)
		}

		bodyWriter := &bodyLogWriter{ResponseWriter: c.Writer, responseBodyLimit: cfg.ResponseBodyLimit}
		c.Writer = bodyWriter
		c.Next()

		status := c.Writer.Status()
		if status >= http.StatusInternalServerError {
			if lastError := c.Errors.Last(); lastError != nil {
				failureAttrs := append([]any{}, requestAttrs...)
				failureAttrs = append(failureAttrs, "status", status, "error", lastError.Err)
				logger.Error("request failed", failureAttrs...)
			}
		}
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
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		logger.Error("http panic recovered", append(requestLogAttrs(c), "panic", recovered)...)
		if c.Writer.Written() {
			c.Abort()
			return
		}
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", fmt.Errorf("panic: %v", recovered)))
	})
}

func requestLogAttrs(c *gin.Context) []any {
	r := c.Request
	return []any{
		"method", r.Method,
		"path", r.URL.Path,
		"uri", r.URL.RequestURI(),
		"request_id", RequestIdFromContext(c),
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	}
}

func shouldSkipRequestLog(r *http.Request, status int, cfg LogRequestConfig) bool {
	return cfg.SkipAssetEnabled && isSkippableAssetPath(r.URL.Path) && isSuccessfulAssetStatus(status)
}

func isSkippableAssetPath(requestPath string) bool {
	requestPath = path.Clean("/" + requestPath)
	return strings.HasPrefix(requestPath, "/assets/")
}

func isSuccessfulAssetStatus(status int) bool {
	return (status >= http.StatusOK && status < http.StatusMultipleChoices) || status == http.StatusNotModified
}

func readRequestBodyForLog(r *http.Request, limit int) (string, error) {
	if limit <= 0 || !isJSONContentType(r.Header.Get("Content-Type")) || r.Body == nil {
		return "", nil
	}
	body := r.Body
	loggedBytes, err := io.ReadAll(io.LimitReader(body, int64(limit)+1))
	r.Body = &prefixReadCloser{reader: io.MultiReader(bytes.NewReader(loggedBytes), body), closer: body}
	if err != nil {
		return "", err
	}
	if len(loggedBytes) == 0 {
		return "", nil
	}
	return truncateLogBody(loggedBytes, limit), nil
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
	responseBodyLimit int
	body              bytes.Buffer
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
	if w.responseBodyLimit <= 0 || w.body.Len() == 0 {
		return ""
	}
	return truncateLogBody(w.body.Bytes(), w.responseBodyLimit)
}

func (w *bodyLogWriter) captureBody(data []byte) {
	if w.responseBodyLimit <= 0 || !isJSONContentType(w.Header().Get("Content-Type")) || len(data) == 0 {
		return
	}
	limit := w.responseBodyLimit + 1
	if w.body.Len() >= limit {
		return
	}
	remaining := limit - w.body.Len()
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.body.Write(data)
}
