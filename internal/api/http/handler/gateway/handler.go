package gatewayhandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	gatewaysvc "github.com/leoninew/pomelo-orbit/internal/application/gateway/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       gatewaysvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service gatewaysvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
