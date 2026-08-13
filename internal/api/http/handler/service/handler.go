package servicehandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	servicesvc "github.com/leoninew/pomelo-orbit/internal/application/service/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       servicesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service servicesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
