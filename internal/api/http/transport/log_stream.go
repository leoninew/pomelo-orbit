package transport

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/requestid"
	logdto "github.com/leoninew/pomelo-orbit/internal/application/logstream/dto"
	logstream "github.com/leoninew/pomelo-orbit/internal/application/logstream/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
)

const logHeartbeatInterval = 15 * time.Second
const logWriteTimeout = 10 * time.Second

func WriteLogStream(c *gin.Context, subscription logstream.Subscription) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	defer func() { _ = subscription.Close() }()
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	controller := http.NewResponseController(c.Writer)
	write := func(data string) error {
		_ = controller.SetWriteDeadline(time.Now().Add(logWriteTimeout))
		defer func() { _ = controller.SetWriteDeadline(time.Time{}) }()
		if _, err := fmt.Fprint(c.Writer, data); err != nil {
			return err
		}
		return controller.Flush()
	}
	writeEvent := func(event *commonv1.LogStreamEvent) error {
		data, err := MarshalProtoJSON(event)
		if err != nil {
			return err
		}
		return write("data: " + string(data) + "\n\n")
	}
	if err := writeEvent(&commonv1.LogStreamEvent{Type: "ready", SourceId: subscription.SourceId, Status: subscription.Status}); err != nil {
		return
	}
	events := make(chan logdto.Event, 1)
	done := make(chan error, 1)
	go func() {
		done <- subscription.Run(ctx, func(event logdto.Event) error {
			select {
			case events <- event:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		close(events)
	}()
	defer func() { cancel(); _ = subscription.Close(); <-done }()
	ticker := time.NewTicker(logHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := write(": heartbeat\n\n"); err != nil {
				return
			}
		case event, ok := <-events:
			if !ok {
				err := <-done
				done <- err
				if err != nil && ctx.Err() == nil {
					classification := apperror.Classify(err)
					status := httpStatus(err)
					if status >= 500 {
						_ = c.Error(err)
					}
					_ = writeEvent(&commonv1.LogStreamEvent{Type: "error", Message: classification.Message, Code: classification.Code, RequestId: requestid.FromGinContext(c), HttpStatus: int32(status), Retryable: status == 429 || status >= 500})
				}
				return
			}
			if err := writeEvent(&commonv1.LogStreamEvent{Type: event.Type, SourceId: event.SourceId, Cursor: event.Cursor, DataBase64: base64.StdEncoding.EncodeToString(event.Data), Status: event.Status, Message: event.Message}); err != nil {
				return
			}
		}
	}
}
