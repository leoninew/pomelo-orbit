package response

import (
	"net/http"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/requestid"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
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
	WriteError(c, apperror.NewForHTTPStatus(status, message))
}

func WriteError(c *gin.Context, err error) {
	classification := apperror.Classify(err)
	if err != nil && classification.StatusCode >= 500 {
		_ = c.Error(err)
	}
	if classification.StatusCode == http.StatusUnauthorized {
		c.Header("WWW-Authenticate", "Bearer")
	}
	c.AbortWithStatusJSON(classification.StatusCode, ErrorResp{
		Code:      classification.Code,
		Error:     classification.Message,
		RequestId: requestid.FromContext(c),
	})
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
