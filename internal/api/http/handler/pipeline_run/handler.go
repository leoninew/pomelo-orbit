package pipelinerunhandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	pipelinerunsvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       pipelinerunsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service pipelinerunsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
