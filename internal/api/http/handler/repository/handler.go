package repositoryhandler

import (
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	repositorysvc "github.com/leoninew/pomelo-orbit/internal/application/repository/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       repositorysvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service repositorysvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}
