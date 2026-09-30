package environmenthandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       environmentsvc.Service
	authenticator security.Authenticator
	terminal      *environmentsvc.TerminalService
	runner        environmentport.TerminalRunner
	origins       []string
}

func (h Handler) WithTerminal(terminal *environmentsvc.TerminalService, runner environmentport.TerminalRunner, origins []string) Handler {
	h.terminal = terminal
	h.runner = runner
	h.origins = origins
	return h
}

func New(logger *slog.Logger, service environmentsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
