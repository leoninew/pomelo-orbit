package response

import (
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
)

type ErrorResp[T any] struct {
	Detail T `json:"detail"`
}

func ProtoJSON(c *gin.Context, status int, message proto.Message) {
	c.Render(status, codec.ProtoJSON{Message: message})
}

func Error[T any](c *gin.Context, status int, detail T) {
	c.JSON(status, ErrorResp[T]{Detail: detail})
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
