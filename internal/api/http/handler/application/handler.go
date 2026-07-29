package applicationhandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	applicationsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/usecase"
	servicesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/usecase"
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
