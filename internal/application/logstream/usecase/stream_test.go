package logstream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	logdto "github.com/leoninew/pomelo-orbit/internal/application/logstream/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
)

type readerStub struct {
	mu     sync.Mutex
	data   []byte
	exists bool
}

func (r *readerStub) Read(_ context.Context, offset int64, limit int) ([]byte, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if offset >= int64(len(r.data)) {
		return nil, r.exists, nil
	}
	return append([]byte(nil), r.data[offset:min(int64(len(r.data)), offset+int64(limit))]...), r.exists, nil
}
func (*readerStub) Close() error { return nil }

func TestFileStreamsBeforeCompletionAndDrainsCancellationOutput(t *testing.T) {
	reader := &readerStub{}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	stream, err := File("source", "", status.WorkStatusCanceled, reader, func(context.Context) (string, error) { return status.WorkStatusCanceled, nil })
	if err != nil {
		t.Fatal(err)
	}
	var output []byte
	var kinds []string
	started := time.Now()
	first := true
	err = stream.Run(ctx, func(event logdto.Event) error {
		kinds = append(kinds, event.Type)
		output = append(output, event.Data...)
		if event.Type == "waiting" && first {
			first = false
			reader.mu.Lock()
			reader.exists = true
			reader.data = []byte("first\n")
			reader.mu.Unlock()
			go func() {
				timer := time.NewTimer(time.Second)
				defer timer.Stop()
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
				reader.mu.Lock()
				reader.data = append(reader.data, []byte("last\n")...)
				reader.mu.Unlock()
			}()
		}
		if event.Type == "complete" && event.Status != status.WorkStatusCanceled {
			t.Fatalf("terminal status = %s", event.Status)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "first\nlast\n" || kinds[0] != "waiting" || kinds[len(kinds)-1] != "complete" {
		t.Fatalf("events = %v, output = %q", kinds, output)
	}
	if time.Since(started) < DrainInterval {
		t.Fatal("closed before cancellation output drain")
	}
}

func TestFileContinuesReceivingWhileTaskRunsAndCanResume(t *testing.T) {
	reader := &readerStub{data: []byte("oldnew"), exists: true}
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := File("source", EncodeCursor("source", 3), status.WorkStatusRunning, reader, func(context.Context) (string, error) { return status.WorkStatusRunning, nil })
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Run(ctx, func(event logdto.Event) error {
		if event.Type != "chunk" || string(event.Data) != "new" {
			t.Fatalf("event = %+v", event)
		}
		offset, err := DecodeCursor(event.Cursor, "source")
		if err != nil || offset != 6 {
			t.Fatalf("offset = %d, err = %v", offset, err)
		}
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stream cancellation = %v", err)
	}
}

func TestCursorBindsSourceAndKeepsInt64Precision(t *testing.T) {
	const offset int64 = 9_007_199_254_740_993
	cursor := EncodeCursor("source", offset)
	decoded, err := DecodeCursor(cursor, "source")
	if err != nil || decoded != offset {
		t.Fatalf("cursor offset = %d, err = %v", decoded, err)
	}
	if _, err := DecodeCursor(cursor, "another"); err == nil {
		t.Fatal("accepted another resource cursor")
	}
}
