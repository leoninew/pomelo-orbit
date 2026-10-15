package clihandler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	"github.com/leoninew/pomelo-orbit/internal/api/http/terminal"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	cliv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/cli"
)

type AccessApplication interface {
	List(context.Context, string) ([]environmentdto.ProjectSummary, error)
	AcquireTerminal(context.Context, string, string) (environmentsvc.TerminalGrant, func(), error)
}
type CommandApplication interface {
	Execute(context.Context, string, environmentdto.CommandInput) (environmentdto.CommandResult, error)
}

type Handler struct {
	auth     security.Authenticator
	access   AccessApplication
	commands CommandApplication
	terminal terminal.Application
	runner   environmentport.TerminalRunner
	logger   *slog.Logger
}

func New(auth security.Authenticator, access AccessApplication, commands CommandApplication, application terminal.Application, runner environmentport.TerminalRunner, logger *slog.Logger) Handler {
	return Handler{auth: auth, access: access, commands: commands, terminal: application, runner: runner, logger: logger}
}

func (h Handler) List(c *gin.Context) {
	user, ok := h.auth.CurrentPATUser(c)
	if !ok {
		return
	}
	items, err := h.access.List(c.Request.Context(), user.Id)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	response := &cliv1.ProjectListResp{Projects: make([]*cliv1.ProjectResp, 0, len(items))}
	for _, p := range items {
		response.Projects = append(response.Projects, &cliv1.ProjectResp{Id: p.Id, Name: p.Name, Code: p.Code, IsActive: p.IsActive, EnvironmentId: p.EnvironmentId, TargetType: p.TargetType, Platform: p.Platform, Host: p.Host, Port: int32(p.Port), Username: p.Username})
	}
	c.Header("Cache-Control", "no-store")
	transport.WriteProtoJSON(c, http.StatusOK, response)
}

func (h Handler) Exec(c *gin.Context) {
	user, ok := h.auth.CurrentPATUser(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
	var req cliv1.ExecReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid command request (maximum 128 KiB)")
		return
	}
	result, err := h.commands.Execute(c.Request.Context(), user.Id, environmentdto.CommandInput{ProjectCode: req.ProjectCode, Argv: req.Argv, Workdir: req.Workdir, Stdin: req.Stdin, TimeoutSeconds: req.TimeoutSeconds})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	transport.WriteProtoJSON(c, http.StatusOK, &cliv1.ExecResp{ProjectId: result.ProjectId, ProjectCode: result.ProjectCode, EnvironmentId: result.EnvironmentId, TargetRevision: result.TargetRevision, State: result.State, ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr, DurationMs: result.DurationMs, OutputTruncated: result.OutputTruncated})
}

func (h Handler) Terminal(c *gin.Context) {
	user, ok := h.auth.CurrentPATUser(c)
	if !ok {
		return
	}
	if strings.TrimSpace(c.GetHeader("Sec-WebSocket-Protocol")) != terminal.Protocol || c.GetHeader("Origin") != "" {
		transport.WriteStatusError(c, http.StatusBadRequest, "CLI terminal requires orbit-terminal-v1 without Origin")
		return
	}
	grant, release, err := h.access.AcquireTerminal(c.Request.Context(), user.Id, c.Query("project_code"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	defer release()
	c.Header("Cache-Control", "no-store")
	terminal.Serve(c, grant, h.terminal, h.runner, h.logger, nil)
}
