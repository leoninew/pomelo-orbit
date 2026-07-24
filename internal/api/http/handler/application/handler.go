package applicationhandler

import (
	"log/slog"
	"net/http"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	applicationsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/usecase"
	servicesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger         *slog.Logger
	service        applicationsvc.Service
	runtimeService servicesvc.Service
	authenticator  security.Authenticator
}

func New(logger *slog.Logger, service applicationsvc.Service, runtimeService servicesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, runtimeService: runtimeService, authenticator: authenticator}
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("application request failed", "error", err)
	}
	transportresponse.Error(c, apperror.StatusCode(err), err.Error())
}
