package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"

	"github.com/leoninew/pomelo-orbit/internal/api/http/requestid"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
)

const (
	RequestIdKey        = requestid.ContextKey
	RequestIdHeader     = requestid.HeaderName
	TruncatedBodySuffix = "..."
	privateKeyMask      = "******"
)

var privateKeyJSONField = regexp.MustCompile(`(?is)("(?:deployment_ssh_private_key|deployment_ssh_key_passphrase|private_key|private_key_passphrase)"\s*:\s*)("(?:\\.|[^"\\])*"?)`)

type LogRequestConfig struct {
	Enabled           bool
	RequestBodyLimit  int
	ResponseBodyLimit int
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
		logger.Info("request started", startedAttrs...)

		bodyWriter := &bodyLogWriter{ResponseWriter: c.Writer, responseBodyLimit: cfg.ResponseBodyLimit}
		c.Writer = bodyWriter
		c.Next()

		status := c.Writer.Status()
		if status >= http.StatusBadRequest {
			if lastError := c.Errors.Last(); lastError != nil {
				failureAttrs := append([]any{}, requestAttrs...)
				failureAttrs = append(failureAttrs, "status", status, "code", apperror.Classify(lastError.Err).Code, "error", lastError.Err)
				level := slog.LevelWarn
				if status >= http.StatusInternalServerError {
					level = slog.LevelError
				}
				logger.Log(c.Request.Context(), level, "request failed", failureAttrs...)
			}
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
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", fmt.Errorf("panic: %v", recovered)))
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
	return formatJSONLogBody(loggedBytes, limit), nil
}

func maskPrivateKeyJSONLogBody(value string) string {
	return privateKeyJSONField.ReplaceAllStringFunc(value, func(field string) string {
		matches := privateKeyJSONField.FindStringSubmatch(field)
		masked := privateKeyMask
		var secret string
		// A truncated JSON string has no captured suffix to reveal.
		if err := json.Unmarshal([]byte(matches[2]), &secret); err == nil {
			characters := []rune(secret)
			if len(characters) > 4 {
				masked += string(characters[len(characters)-4:])
			}
		}
		encoded, _ := json.Marshal(masked)
		return matches[1] + string(encoded)
	})
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func formatJSONLogBody(value []byte, limit int) string {
	if limit <= 0 {
		return ""
	}
	suffix := ""
	if len(value) > limit {
		value = value[:limit]
		for len(value) > 0 && !utf8.Valid(value) {
			value = value[:len(value)-1]
		}
		suffix = TruncatedBodySuffix
	}
	return maskPrivateKeyJSONLogBody(string(value)) + suffix
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

func (w *bodyLogWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

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
	return formatJSONLogBody(w.body.Bytes(), w.responseBodyLimit)
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
