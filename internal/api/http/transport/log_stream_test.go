package transport

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	logdto "github.com/leoninew/pomelo-orbit/internal/application/logstream/dto"
	logstream "github.com/leoninew/pomelo-orbit/internal/application/logstream/usecase"
)

func TestLogStreamFlushesAndCancelsSourceWhenClientCloses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stopped := make(chan struct{})
	router := gin.New()
	router.GET("/stream", func(c *gin.Context) {
		WriteLogStream(c, logstream.Subscription{SourceId: "source", Close: func() error { return nil }, Run: func(ctx context.Context, emit logdto.Emit) error {
			defer close(stopped)
			if err := emit(logdto.Event{Type: "chunk", SourceId: "source", Data: []byte("live"), Cursor: "cursor"}); err != nil {
				return err
			}
			<-ctx.Done()
			return ctx.Err()
		}})
	})
	server := httptest.NewServer(router)
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(server.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	if response.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatal("proxy buffering was not disabled")
	}
	reader := bufio.NewReader(response.Body)
	first, err := reader.ReadString('\n')
	if err != nil || !strings.Contains(first, `"type":"ready"`) {
		t.Fatalf("first event = %s, %v", first, err)
	}
	_, _ = reader.ReadString('\n')
	chunk, err := reader.ReadString('\n')
	if err != nil || !strings.Contains(chunk, `"data_base64":"bGl2ZQ=="`) {
		t.Fatalf("chunk = %s, %v", chunk, err)
	}
	_ = response.Body.Close()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("client close did not cancel source")
	}
}

func TestLogStreamEmitsStructuredReadError(t *testing.T) {
	router := gin.New()
	router.GET("/stream", func(c *gin.Context) {
		WriteLogStream(c, logstream.Subscription{Close: func() error { return nil }, Run: func(context.Context, logdto.Emit) error { return io.ErrUnexpectedEOF }})
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if !strings.Contains(recorder.Body.String(), `"type":"error"`) || !strings.Contains(recorder.Body.String(), `"retryable":true`) {
		t.Fatalf("error stream = %s", recorder.Body.String())
	}
}
