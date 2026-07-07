package response

import (
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/codec"
)

type ErrorResp[T any] struct {
	Detail T `json:"detail"`
}

func PageCount(total int, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	return (total + perPage - 1) / perPage
}

func DecodeJSON(c *gin.Context, value any) error {
	return DecodeJSONReader(c.Request.Body, value)
}

func DecodeJSONReader(r io.Reader, value any) error {
	if message, ok := value.(proto.Message); ok {
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		return codec.UnmarshalProtoJSON(data, message)
	}
	return json.NewDecoder(r).Decode(value)
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

func QueryInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func QueryProjectId(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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
