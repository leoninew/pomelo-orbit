package repositoryhandler

import (
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	repositorysvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       repositorysvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service repositorysvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
