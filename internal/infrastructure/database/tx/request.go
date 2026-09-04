package tx

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

const unwrittenResponseSize = -1

// Middleware begins a request-scoped transaction for API writes. Successful
// responses stay buffered until the transaction commits.

func Middleware(db *sql.DB, skipPaths ...string) gin.HandlerFunc {
	skipped := make(map[string]struct{}, len(skipPaths))
	for _, path := range skipPaths {
		skipped[path] = struct{}{}
	}
	return func(c *gin.Context) {
		if !isWriteRequest(c.Request.Method) {
			c.Next()
			return
		}
		if skipPath(c.Request.URL.Path, skipped) {
			c.Next()
			return
		}

		ctx := WithDb(c.Request.Context(), db)
		sqlTx, err := db.BeginTx(ctx, nil)
		if err != nil {
			transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
			return
		}
		ctx = WithTx(ctx, sqlTx)
		c.Request = c.Request.WithContext(ctx)

		originalWriter := c.Writer
		originalHeaders := cloneHeader(originalWriter.Header())
		bufferedWriter := newBufferedResponseWriter(originalWriter)
		c.Writer = bufferedWriter

		committed := false
		defer func() {
			if recovered := recover(); recovered != nil {
				rollback(c, sqlTx)
				bufferedWriter.discard()
				restoreHeader(originalWriter.Header(), originalHeaders)
				c.Writer = originalWriter
				panic(recovered)
			}
			if !committed {
				rollback(c, sqlTx)
			}
			c.Writer = originalWriter
		}()

		c.Next()

		if c.IsAborted() || len(c.Errors) > 0 || bufferedWriter.Status() >= http.StatusBadRequest {
			bufferedWriter.flush()
			return
		}
		if err := sqlTx.Commit(); err != nil {
			bufferedWriter.discard()
			restoreHeader(originalWriter.Header(), originalHeaders)
			transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
			bufferedWriter.flush()
			return
		}
		committed = true
		bufferedWriter.flush()
	}
}

func skipPath(path string, skipped map[string]struct{}) bool {
	if _, ok := skipped[path]; ok {
		return true
	}
	for pattern := range skipped {
		if matchesPathTemplate(path, pattern) {
			return true
		}
	}
	return false
}

func matchesPathTemplate(path string, pattern string) bool {
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(pathSegments) != len(patternSegments) {
		return false
	}
	for index, segment := range patternSegments {
		if strings.HasPrefix(segment, ":") {
			if pathSegments[index] == "" {
				return false
			}
			continue
		}
		if pathSegments[index] != segment {
			return false
		}
	}
	return true
}
func isWriteRequest(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func rollback(c *gin.Context, transaction *sql.Tx) {
	if err := transaction.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		_ = c.Error(err)
	}
}

type bufferedResponseWriter struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
	size   int
}

func newBufferedResponseWriter(writer gin.ResponseWriter) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		ResponseWriter: writer,
		status:         writer.Status(),
		size:           unwrittenResponseSize,
	}
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if status > 0 && !w.Written() {
		w.status = status
	}
}

func (w *bufferedResponseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
	}
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	w.WriteHeaderNow()
	count, err := w.body.Write(data)
	w.size += count
	return count, err
}

func (w *bufferedResponseWriter) WriteString(value string) (int, error) {
	w.WriteHeaderNow()
	count, err := w.body.WriteString(value)
	w.size += count
	return count, err
}

func (w *bufferedResponseWriter) Status() int {
	return w.status
}

func (w *bufferedResponseWriter) Size() int {
	return w.size
}

func (w *bufferedResponseWriter) Written() bool {
	return w.size != unwrittenResponseSize
}

func (w *bufferedResponseWriter) Flush() {
	w.WriteHeaderNow()
}

func (w *bufferedResponseWriter) flush() {
	w.ResponseWriter.WriteHeader(w.status)
	w.ResponseWriter.WriteHeaderNow()
	if w.body.Len() == 0 {
		return
	}
	_, _ = w.ResponseWriter.Write(w.body.Bytes())
}

func (w *bufferedResponseWriter) discard() {
	w.body.Reset()
	w.status = http.StatusOK
	w.size = unwrittenResponseSize
}

func cloneHeader(source http.Header) http.Header {
	copy := make(http.Header, len(source))
	for key, values := range source {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}

func restoreHeader(destination http.Header, source http.Header) {
	for key := range destination {
		delete(destination, key)
	}
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}
