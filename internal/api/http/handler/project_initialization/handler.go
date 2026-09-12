package projectinitializationhandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	projectinitializationsvc "github.com/leoninew/pomelo-orbit/internal/application/project_initialization/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       projectinitializationsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service projectinitializationsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
