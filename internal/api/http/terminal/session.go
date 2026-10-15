package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
)

const Protocol = "orbit-terminal-v1"
const terminalProtocol = Protocol

type Application interface {
	ConfirmConnection(context.Context, environmentsvc.TerminalGrant, string) (environmentsvc.TerminalGrant, error)
	ValidateSession(context.Context, environmentsvc.TerminalGrant) error
	MonitorSession(context.Context, environmentsvc.TerminalGrant, func() time.Time) string
	RuntimeUsername(environmentsvc.TerminalGrant) string
}

type Control struct {
	Type string `json:"type"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Code int    `json:"code,omitempty"`
}

type terminalMessage struct {
	typeOf websocket.MessageType
	data   []byte
}

func Serve(c *gin.Context, grant environmentsvc.TerminalGrant, application Application, runner environmentport.TerminalRunner, logger *slog.Logger, origins []string) {
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		Subprotocols: []string{terminalProtocol}, OriginPatterns: origins,
	})
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(64 * 1024)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	events := make(chan string, 8)
	sessionTimer := time.AfterFunc(environmentsvc.TerminalSessionLimit, func() { events <- "session_limit" })
	defer sessionTimer.Stop()
	messages := make(chan terminalMessage, 64)
	// Keep reading while SSH connects or waits for input, so closing the browser
	// also cancels a blocked SSH handshake or write.
	go func() {
		for {
			messageType, data, readErr := conn.Read(ctx)
			if readErr != nil {
				events <- "connection_closed"
				cancel()
				return
			}
			select {
			case messages <- terminalMessage{typeOf: messageType, data: data}:
			case <-ctx.Done():
				return
			default:
				events <- "input_overflow"
				cancel()
				return
			}
		}
	}()
	connectTimer := time.AfterFunc(environmentsvc.TerminalConnectLimit, cancel)
	defer connectTimer.Stop()
	session, err := runner.StartTerminal(ctx, grant.Target, 80, 24)
	if err != nil {
		_ = writeTerminalControl(ctx, conn, Control{Type: "connection_failed"})
		_ = conn.Close(websocket.StatusInternalError, "connection_failed")
		return
	}
	defer func() { _ = session.Close() }()
	fingerprint := ""
	if metadata, ok := session.(environmentport.TerminalHostKey); ok {
		fingerprint = metadata.HostKeyFingerprint()
	}
	grant, err = application.ConfirmConnection(ctx, grant, fingerprint)
	if err != nil {
		_ = writeTerminalControl(ctx, conn, Control{Type: "connection_failed"})
		_ = conn.Close(websocket.StatusInternalError, "connection_failed")
		return
	}
	connectTimer.Stop()
	if err := writeTerminalControl(ctx, conn, Control{Type: "ready"}); err != nil {
		return
	}
	started := time.Now()
	logger.Info("environment terminal connected", "actor_id", grant.ActorId, "project_id", grant.ProjectId,
		"environment_id", grant.Target.Environment.Id, "target_type", grant.Target.Environment.TargetType,
		"runtime_username", application.RuntimeUsername(grant))
	var lastActivity atomic.Int64
	lastActivity.Store(started.UnixNano())
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		buffer := make([]byte, 16*1024)
		for {
			n, readErr := session.Output().Read(buffer)
			if n > 0 {
				if err := conn.Write(ctx, websocket.MessageBinary, buffer[:n]); err != nil {
					events <- "connection_closed"
					return
				}
				lastActivity.Store(time.Now().UnixNano())
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					events <- "output_failed"
				}
				return
			}
		}
	}()
	go func() {
		code, waitErr := session.Wait()
		<-outputDone
		if waitErr != nil {
			code = 1
		}
		_ = writeTerminalControl(ctx, conn, Control{Type: "exit", Code: code})
		events <- "remote_exit"
	}()
	go func() {
		for {
			var message terminalMessage
			select {
			case message = <-messages:
			case <-ctx.Done():
				return
			}
			lastActivity.Store(time.Now().UnixNano())
			if message.typeOf == websocket.MessageBinary {
				checkCtx, checkCancel := context.WithTimeout(ctx, 5*time.Second)
				checkErr := application.ValidateSession(checkCtx, grant)
				checkCancel()
				if checkErr != nil {
					events <- "access_or_target_changed"
					return
				}
				if _, err := session.Input().Write(message.data); err != nil {
					events <- "input_failed"
					return
				}
				continue
			}
			var control Control
			if message.typeOf != websocket.MessageText || json.Unmarshal(message.data, &control) != nil ||
				control.Type != "resize" || control.Cols < 1 || control.Cols > 500 || control.Rows < 1 || control.Rows > 200 {
				events <- "invalid_message"
				return
			}
			if err := session.Resize(control.Cols, control.Rows); err != nil {
				events <- "resize_failed"
				return
			}
		}
	}()
	go func() {
		reason := application.MonitorSession(ctx, grant, func() time.Time {
			return time.Unix(0, lastActivity.Load())
		})
		if reason != "" {
			select {
			case events <- reason:
			case <-ctx.Done():
			}
		}
	}()
	reason := "session_limit"
	select {
	case reason = <-events:
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			reason = "server_shutdown"
			select {
			case reason = <-events:
			default:
			}
		}
	}
	if reason == "idle_timeout" || reason == "access_or_target_changed" || reason == "session_limit" {
		notifyCtx, notifyCancel := context.WithTimeout(ctx, time.Second)
		_ = writeTerminalControl(notifyCtx, conn, Control{Type: reason})
		notifyCancel()
	}
	if ctx.Err() == nil {
		_ = conn.Close(websocket.StatusNormalClosure, reason)
	}
	cancel()
	_ = session.Close()
	logger.Info("environment terminal disconnected", "actor_id", grant.ActorId, "project_id", grant.ProjectId,
		"environment_id", grant.Target.Environment.Id, "target_type", grant.Target.Environment.TargetType,
		"runtime_username", application.RuntimeUsername(grant),
		"duration_ms", time.Since(started).Milliseconds(), "reason", reason)
}

func writeTerminalControl(ctx context.Context, conn *websocket.Conn, control Control) error {
	data, err := json.Marshal(control)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}
