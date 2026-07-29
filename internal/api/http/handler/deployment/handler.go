package deploymenthandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	deploymentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       deploymentsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service deploymentsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
