package credentialhandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	credentialsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/credential/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       credentialsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service credentialsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
