package servicehandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	servicesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       servicesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service servicesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
