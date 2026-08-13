package applicationhandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	servicesvc "github.com/leoninew/pomelo-orbit/internal/application/service/usecase"
)

type Handler struct {
	logger         *slog.Logger
	service        applicationsvc.Service
	runtimeService servicesvc.Service
	authenticator  security.Authenticator
}

func New(logger *slog.Logger, service applicationsvc.Service, runtimeService servicesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, runtimeService: runtimeService, authenticator: authenticator}
}
