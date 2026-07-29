package pipelinerunhandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	pipelinerunsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       pipelinerunsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service pipelinerunsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
