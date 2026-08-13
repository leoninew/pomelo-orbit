package deploymenthandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       deploymentsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service deploymentsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
