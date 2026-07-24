package pipelinehandler

import (
	"log/slog"
	"net/http"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	pipelinesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger        *slog.Logger
	service       pipelinesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service pipelinesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("pipeline request failed", "error", err)
	}
	transportresponse.Error(c, apperror.StatusCode(err), err.Error())
}
