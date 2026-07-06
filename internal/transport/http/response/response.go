package response

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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

func DecodeJSON(r io.Reader, value any) error {
	if message, ok := value.(proto.Message); ok {
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		return protojson.UnmarshalOptions{DiscardUnknown: false}.Unmarshal(data, message)
	}
	return json.NewDecoder(r).Decode(value)
}

func protoMessage(value any) (proto.Message, bool) {
	if message, ok := value.(proto.Message); ok {
		return message, true
	}
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return nil, false
	}
	if reflected.Kind() == reflect.Pointer {
		return nil, false
	}
	copyValue := reflect.New(reflected.Type())
	copyValue.Elem().Set(reflected)
	message, ok := copyValue.Interface().(proto.Message)
	return message, ok
}

func JSON(logger *slog.Logger, w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if message, ok := protoMessage(value); ok {
		data, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(message)
		if err != nil {
			logger.Error("marshal proto response failed", "error", err)
			return
		}
		if _, err := w.Write(append(data, '\n')); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
			logger.Error("write response failed", "error", err)
		}
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		logger.Error("write response failed", "error", err)
	}
}

func Error(logger *slog.Logger, w http.ResponseWriter, status int, detail string) {
	JSON(logger, w, status, ErrorResp[string]{Detail: detail})
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
