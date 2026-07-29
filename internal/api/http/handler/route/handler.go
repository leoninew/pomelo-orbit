package routehandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	routesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/route/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       routesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service routesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
