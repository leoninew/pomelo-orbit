package gatewayhandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	gatewaysvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       gatewaysvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service gatewaysvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
