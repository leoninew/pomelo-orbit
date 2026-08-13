package credentialhandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       credentialsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service credentialsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
