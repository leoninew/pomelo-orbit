package response

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"testing"
)

type failingResponseWriter struct {
	header http.Header
	status int
	err    error
}

func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *failingResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestJSONLogsWriteFailureThroughDefaultLogger(t *testing.T) {
	oldDefault := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(oldDefault)
	})

	var logBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logBuffer, nil)))

	writer := &failingResponseWriter{err: errors.New("boom")}
	JSON(slog.Default(), writer, http.StatusAccepted, map[string]string{"ok": "true"})

	if got := writer.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("expected content type, got %q", got)
	}
	if writer.status != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, writer.status)
	}

	var entry map[string]any
	if err := json.Unmarshal(logBuffer.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v\n%s", err, logBuffer.String())
	}
	if entry["msg"] != "write response failed" {
		t.Fatalf("expected write failure log, got %+v", entry)
	}
	if got := entry["error"]; got == nil {
		t.Fatalf("expected error field, got %+v", entry)
	}
}
