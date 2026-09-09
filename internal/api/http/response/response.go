package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/requestid"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/leoninew/pomelo-orbit/internal/api/http/codec"
)

type ErrorResp struct {
	Code      string `json:"code"`
	Error     string `json:"error"`
	RequestId string `json:"requestId"`
}

func ProtoJSON(c *gin.Context, status int, message proto.Message) {
	c.Render(status, codec.ProtoJSON{Message: message})
}

func WriteStatusError(c *gin.Context, status int, message string) {
	WriteError(c, apperror.New(kindForHTTPStatus(status), message))
}

func WriteError(c *gin.Context, err error) {
	classification := apperror.Classify(err)
	status := HTTPStatus(err)
	if err != nil && status >= 500 {
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

func HTTPStatus(err error) int {
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

func PageCount(total int, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	return (total + perPage - 1) / perPage
}

func Ptrs[T any](items []T) []*T {
	if items == nil {
		return nil
	}
	resp := make([]*T, 0, len(items))
	for i := range items {
		resp = append(resp, &items[i])
	}
	return resp
}

func ProtoValue(value any) *structpb.Value {
	converted, err := structpb.NewValue(value)
	if err != nil {
		return structpb.NewNullValue()
	}
	return converted
}

func NativeValue(value *structpb.Value) any {
	if value == nil {
		return nil
	}
	return value.AsInterface()
}

func OptionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func OptionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func FormatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func FormatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := FormatTime(*value)
	return &formatted
}
