package transport

import (
	"time"

	"google.golang.org/protobuf/types/known/structpb"
)

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
	result := make([]*T, 0, len(items))
	for i := range items {
		result = append(result, &items[i])
	}
	return result
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
