package dialoguehandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	dialoguesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       dialoguesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service dialoguesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
