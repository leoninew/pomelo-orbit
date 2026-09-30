package logstream

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	logdto "github.com/leoninew/pomelo-orbit/internal/application/logstream/dto"
	logport "github.com/leoninew/pomelo-orbit/internal/application/logstream/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

const (
	ChunkSize       = 32 * 1024
	PollInterval    = 250 * time.Millisecond
	RecheckInterval = 2 * time.Second
	DrainInterval   = 3 * time.Second
)

type Subscription struct {
	SourceId string
	Status   string
	Run      func(context.Context, logdto.Emit) error
	Close    func() error
}

func SourceId(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

type fileCursor struct {
	Source string `json:"s"`
	Offset string `json:"o"`
}

func EncodeCursor(source string, offset int64) string {
	data, _ := json.Marshal(fileCursor{Source: source, Offset: strconv.FormatInt(offset, 10)})
	return base64.RawURLEncoding.EncodeToString(data)
}

func DecodeCursor(value, source string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	var cursor fileCursor
	if err != nil || len(data) > 512 || json.Unmarshal(data, &cursor) != nil {
		return 0, apperror.New(apperror.KindValidation, "Invalid log cursor")
	}
	if cursor.Source != source {
		return 0, apperror.New(apperror.KindConflict, "Log source changed; reopen logs")
	}
	offset, err := strconv.ParseInt(cursor.Offset, 10, 64)
	if err != nil || offset < 0 {
		return 0, apperror.New(apperror.KindValidation, "Invalid log cursor offset")
	}
	return offset, nil
}

// State also rechecks authorization and source identity; a terminal task needs a final drain.
type State func(context.Context) (string, error)

func File(source, cursor, initialStatus string, reader logport.Reader, state State) (Subscription, error) {
	offset, err := DecodeCursor(cursor, source)
	if err != nil {
		_ = reader.Close()
		return Subscription{}, err
	}
	return Subscription{SourceId: source, Status: initialStatus, Close: reader.Close, Run: func(ctx context.Context, emit logdto.Emit) error {
		currentStatus := initialStatus
		lastCheck := time.Time{}
		quietSince := time.Now()
		terminalSince := time.Time{}
		waiting := false
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			if time.Since(lastCheck) >= RecheckInterval {
				var err error
				currentStatus, err = state(ctx)
				if err != nil {
					return err
				}
				lastCheck = time.Now()
				if status.WorkStatusIsComplete(currentStatus) && terminalSince.IsZero() {
					terminalSince = time.Now()
				}
			}
			data, exists, err := reader.Read(ctx, offset, ChunkSize)
			if err != nil {
				return apperror.Wrap(apperror.KindUnavailable, "Failed to read log stream", err)
			}
			if len(data) > 0 {
				offset += int64(len(data))
				quietSince = time.Now()
				waiting = false
				if err := emit(logdto.Event{Type: "chunk", SourceId: source, Cursor: EncodeCursor(source, offset), Data: data}); err != nil {
					return err
				}
				continue
			}
			if status.WorkStatusIsComplete(currentStatus) && time.Since(quietSince) >= DrainInterval && time.Since(terminalSince) >= DrainInterval {
				return emit(logdto.Event{Type: "complete", SourceId: source, Cursor: EncodeCursor(source, offset), Status: currentStatus})
			}
			if !exists && !waiting {
				if err := emit(logdto.Event{Type: "waiting", SourceId: source}); err != nil {
					return err
				}
				waiting = true
			}
			if err := Wait(ctx, PollInterval); err != nil {
				return err
			}
		}
	}}, nil
}

func Wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
