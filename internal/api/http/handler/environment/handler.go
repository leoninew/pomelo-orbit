package environmenthandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       environmentsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service environmentsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
