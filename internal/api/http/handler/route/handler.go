package routehandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	routesvc "github.com/leoninew/pomelo-orbit/internal/application/route/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       routesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service routesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
