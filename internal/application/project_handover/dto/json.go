package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Encode serializes a package using the v1 wire format. Ownership fields are
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

// Decode accepts only the v1 document shape. It rejects transfer ownership
// fields before converting snake_case wire names to the existing business DTOs.
func Decode(data []byte) (Package, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return Package{}, fmt.Errorf("decode handover JSON: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Package{}, err
	}
	if _, ok := value.(map[string]any); !ok {
		return Package{}, fmt.Errorf("handover package must be a JSON object")
	}
	if err := rejectOwnershipFields(value, nil); err != nil {
		return Package{}, err
	}
	normalized, err := json.Marshal(normalizeForImport(value))
	if err != nil {
		return Package{}, fmt.Errorf("normalize handover package: %w", err)
	}
	strict := json.NewDecoder(bytes.NewReader(normalized))
	strict.DisallowUnknownFields()
	var item Package
	if err := strict.Decode(&item); err != nil {
		return Package{}, fmt.Errorf("decode handover package fields: %w", err)
	}
	if err := ensureEOF(strict); err != nil {
		return Package{}, err
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

func normalizeForImport(value any) any {
	switch current := value.(type) {
	case []any:
		result := make([]any, len(current))
		for index, item := range current {
			result[index] = normalizeForImport(item)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(current))
		for key, item := range current {
			result[camelCase(key)] = normalizeForImport(item)
		}
		return result
	default:
		return value
	}
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

func camelCase(value string) string {
	parts := strings.FieldsFunc(value, func(current rune) bool { return current == '_' || current == '-' || current == ' ' })
	if len(parts) == 0 {
		return value
	}
	for index := range parts {
		if index == 0 {
			continue
		}
		parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
	}
	return strings.Join(parts, "")
}
