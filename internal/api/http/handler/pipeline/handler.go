package pipelinehandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	pipelinesvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       pipelinesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service pipelinesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
