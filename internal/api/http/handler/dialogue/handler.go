package dialoguehandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	dialoguesvc "github.com/leoninew/pomelo-orbit/internal/application/dialogue/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       dialoguesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service dialoguesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
