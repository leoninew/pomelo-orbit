package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/requestid"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

type ErrorResp struct {
	Code      string `json:"code"`
	Error     string `json:"error"`
	RequestId string `json:"requestId"`
}

func WriteStatusError(c *gin.Context, status int, message string) {
	WriteError(c, apperror.New(kindForHTTPStatus(status), message))
}

func WriteError(c *gin.Context, err error) {
	classification := apperror.Classify(err)
	status := httpStatus(err)
	if err != nil && status >= http.StatusInternalServerError {
		_ = c.Error(err)
	}
	if status == http.StatusUnauthorized {
		c.Header("WWW-Authenticate", "Bearer")
	}
	c.AbortWithStatusJSON(status, ErrorResp{
		Code:      classification.Code,
		Error:     classification.Message,
		RequestId: requestid.FromGinContext(c),
	})
}

func httpStatus(err error) int {
	appErr, ok := apperror.As(err)
	if !ok {
		return http.StatusInternalServerError
	}
	return statusForKind(appErr.Kind)
}

func statusForKind(kind apperror.Kind) int {
	switch kind {
	case apperror.KindValidation:
		return http.StatusBadRequest
	case apperror.KindUnauthorized:
		return http.StatusUnauthorized
	case apperror.KindForbidden:
		return http.StatusForbidden
	case apperror.KindNotFound:
		return http.StatusNotFound
	case apperror.KindConflict:
		return http.StatusConflict
	case apperror.KindMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case apperror.KindRateLimited:
		return http.StatusTooManyRequests
	case apperror.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func kindForHTTPStatus(status int) apperror.Kind {
	switch status {
	case http.StatusBadRequest:
		return apperror.KindValidation
	case http.StatusUnauthorized:
		return apperror.KindUnauthorized
	case http.StatusForbidden:
		return apperror.KindForbidden
	case http.StatusNotFound:
		return apperror.KindNotFound
	case http.StatusConflict:
		return apperror.KindConflict
	case http.StatusMethodNotAllowed:
		return apperror.KindMethodNotAllowed
	case http.StatusTooManyRequests:
		return apperror.KindRateLimited
	case http.StatusServiceUnavailable:
		return apperror.KindUnavailable
	default:
		return apperror.KindInternal
	}
}
