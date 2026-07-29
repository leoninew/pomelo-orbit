package pipelinehandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	pipelinesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       pipelinesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service pipelinesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
