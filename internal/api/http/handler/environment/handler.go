package environmenthandler

import (
	"context"
	"log/slog"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
)

type TerminalApplication interface {
	IssueTicket(context.Context, string, string) (string, error)
	RedeemTicket(context.Context, string, string) (environmentsvc.TerminalGrant, func(), error)
	ConfirmConnection(context.Context, environmentsvc.TerminalGrant, string) (environmentsvc.TerminalGrant, error)
	ValidateSession(context.Context, environmentsvc.TerminalGrant) error
	MonitorSession(context.Context, environmentsvc.TerminalGrant, func() time.Time) string
	RuntimeUsername(environmentsvc.TerminalGrant) string
}

type Handler struct {
	logger        *slog.Logger
	service       environmentsvc.Service
	authenticator security.Authenticator
	terminal      TerminalApplication
	runner        environmentport.TerminalRunner
	origins       []string
}

func (h Handler) WithTerminal(terminal TerminalApplication, runner environmentport.TerminalRunner, origins []string) Handler {
	h.terminal = terminal
	h.runner = runner
	h.origins = origins
	return h
}

func New(logger *slog.Logger, service environmentsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
