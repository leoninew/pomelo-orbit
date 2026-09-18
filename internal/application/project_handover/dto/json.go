package dto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

type decodeError struct {
	message string
	err     error
}

func (e decodeError) Error() string {
	return e.err.Error()
}

func (e decodeError) Unwrap() error {
	return e.err
}

// DecodeErrorMessage returns a safe, actionable summary of a package decoding failure.
func DecodeErrorMessage(err error) string {
	var decodeErr decodeError
	if errors.As(err, &decodeErr) {
		return decodeErr.message
	}
	return "Invalid handover package"
}

func newDecodeError(message string, err error) error {
	return decodeError{message: message, err: err}
}

// Encode serializes a package using the v2 wire format. Ownership fields are
// deliberately not transferable: the import target supplies every project ID.
func Encode(item Package) ([]byte, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("marshal handover package: %w", err)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("normalize handover package: %w", err)
	}
	return json.Marshal(normalizeForExport(value))
}

// Decode accepts the current wire document shape. It rejects transfer ownership
// fields before converting snake_case wire names to the existing business DTOs.
func Decode(data []byte) (Package, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return Package{}, newDecodeError("Handover package is not valid JSON", fmt.Errorf("decode handover JSON: %w", err))
	}
	if err := ensureEOF(decoder); err != nil {
		return Package{}, newDecodeError("Handover package must contain exactly one JSON document", err)
	}
	if _, ok := value.(map[string]any); !ok {
		return Package{}, newDecodeError("Handover package must be a JSON object", fmt.Errorf("handover package must be a JSON object"))
	}
	if err := rejectOwnershipFields(value, nil); err != nil {
		return Package{}, newDecodeError("Handover package must not contain source project ownership fields", err)
	}
	normalized, err := json.Marshal(normalizeForImport(value, reflect.TypeOf(Package{})))
	if err != nil {
		return Package{}, newDecodeError("Handover package has invalid content", fmt.Errorf("normalize handover package: %w", err))
	}
	strict := json.NewDecoder(bytes.NewReader(normalized))
	strict.DisallowUnknownFields()
	var item Package
	if err := strict.Decode(&item); err != nil {
		return Package{}, newDecodeError("Handover package contains unsupported or invalid fields", fmt.Errorf("decode handover package fields: %w", err))
	}
	if err := ensureEOF(strict); err != nil {
		return Package{}, newDecodeError("Handover package must contain exactly one JSON document", err)
	}
	return item, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("handover package must contain one JSON document")
		}
		return fmt.Errorf("decode handover JSON: %w", err)
	}
	return nil
}

func normalizeForExport(value any) any {
	switch current := value.(type) {
	case []any:
		result := make([]any, len(current))
		for index, item := range current {
			result[index] = normalizeForExport(item)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(current))
		for key, item := range current {
			normalizedKey := snakeCase(key)
			if isProjectOwnershipField(normalizedKey) {
				continue
			}
			result[normalizedKey] = normalizeForExport(item)
		}
		return result
	default:
		return value
	}
}

// normalizeForImport maps the snake_case wire document to the actual JSON key
// for the target Go type. This preserves explicit json tags while converting
// untagged fields to their Go names for strict decoding.
func normalizeForImport(value any, target reflect.Type) any {
	switch current := value.(type) {
	case []any:
		target = indirectType(target)
		if target.Kind() == reflect.Array || target.Kind() == reflect.Slice {
			target = target.Elem()
		}
		result := make([]any, len(current))
		for index, item := range current {
			result[index] = normalizeForImport(item, target)
		}
		return result
	case map[string]any:
		target = indirectType(target)
		if target.Kind() == reflect.Map {
			result := make(map[string]any, len(current))
			for key, item := range current {
				result[key] = normalizeForImport(item, target.Elem())
			}
			return result
		}
		if target.Kind() != reflect.Struct {
			return current
		}
		result := make(map[string]any, len(current))
		for key, item := range current {
			field, ok := fieldForWireKey(target, key)
			if !ok {
				// Keep unexpected fields intact so the strict decoder rejects them.
				result[key] = item
				continue
			}
			result[jsonFieldName(field)] = normalizeForImport(item, field.Type)
		}
		return result
	default:
		return value
	}
}

func indirectType(target reflect.Type) reflect.Type {
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	return target
}

func fieldForWireKey(target reflect.Type, key string) (reflect.StructField, bool) {
	for index := 0; index < target.NumField(); index++ {
		field := target.Field(index)
		if field.PkgPath != "" {
			continue
		}
		name := jsonFieldName(field)
		if name == "-" {
			continue
		}
		wireName := snakeCase(field.Name)
		if explicitName := explicitJSONFieldName(field); explicitName != "" {
			wireName = explicitName
		}
		if key == wireName {
			return field, true
		}
	}
	return reflect.StructField{}, false
}

func jsonFieldName(field reflect.StructField) string {
	if name := explicitJSONFieldName(field); name != "" {
		return name
	}
	return field.Name
}

func explicitJSONFieldName(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	return name
}

func rejectOwnershipFields(value any, path []string) error {
	switch current := value.(type) {
	case []any:
		for _, item := range current {
			if err := rejectOwnershipFields(item, path); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, item := range current {
			normalizedKey := snakeCase(key)
			if isProjectOwnershipField(normalizedKey) || (len(path) == 1 && path[0] == "project" && normalizedKey == "id") {
				return fmt.Errorf("handover package must not include %s", strings.Join(append(path, normalizedKey), "."))
			}
			if err := rejectOwnershipFields(item, append(path, normalizedKey)); err != nil {
				return err
			}
		}
	}
	return nil
}

func isProjectOwnershipField(key string) bool {
	return strings.EqualFold(strings.ReplaceAll(key, "_", ""), "projectid")
}

func snakeCase(value string) string {
	var result strings.Builder
	for index, current := range value {
		if current >= 'A' && current <= 'Z' {
			if index > 0 {
				previous := rune(value[index-1])
				nextLower := index+1 < len(value) && value[index+1] >= 'a' && value[index+1] <= 'z'
				if (previous >= 'a' && previous <= 'z') || (previous >= 'A' && previous <= 'Z' && nextLower) {
					result.WriteByte('_')
				}
			}
			result.WriteRune(current + ('a' - 'A'))
			continue
		}
		if current == '-' || current == ' ' {
			result.WriteByte('_')
			continue
		}
		result.WriteRune(current)
	}
	return result.String()
}
