package repositoryhandler

import (
	"log/slog"
	"net/http"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	repositorysvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger        *slog.Logger
	service       repositorysvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service repositorysvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("repository request failed", "error", err)
	}
	transportresponse.WriteError(c, err)
}
