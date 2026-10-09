package config

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const EnvPrefix = "POMELO_ORBIT_"

type Field struct {
	Key         string
	Path        string
	EnvName     string
	Type        string
	Description string
	index       []int
}

var configFields = sync.OnceValue(func() []Field {
	return discoverFields(reflect.TypeFor[Config](), nil, nil)
})

func Fields() []Field {
	return append([]Field(nil), configFields()...)
}

func discoverFields(kind reflect.Type, path []string, index []int) []Field {
	var fields []Field
	for position := range kind.NumField() {
		field := kind.Field(position)
		name := strings.Split(field.Tag.Get("mapstructure"), ",")[0]
		if name == "-" || !field.IsExported() {
			continue
		}
		if name == "" {
			panic("configuration field requires a mapstructure tag: " + field.Name)
		}
		fieldPath := append(append([]string(nil), path...), name)
		fieldIndex := append(append([]int(nil), index...), position)
		if field.Type.Kind() == reflect.Struct {
			fields = append(fields, discoverFields(field.Type, fieldPath, fieldIndex)...)
			continue
		}
		valueType := ""
		switch {
		case field.Type == reflect.TypeFor[time.Duration]():
			valueType = "duration"
		case field.Type.Kind() == reflect.String:
			valueType = "string"
		case field.Type.Kind() == reflect.Bool:
			valueType = "boolean"
		case field.Type.Kind() == reflect.Int:
			valueType = "integer"
		case field.Type == reflect.TypeFor[[]string]():
			valueType = "string_list"
		default:
			panic("unsupported configuration field type: " + field.Name)
		}
		key := strings.Join(fieldPath, "__")
		fields = append(fields, Field{Key: key, Path: strings.Join(fieldPath, "."), EnvName: EnvPrefix + strings.ToUpper(key), Type: valueType, Description: field.Tag.Get("description"), index: fieldIndex})
	}
	return fields
}

func (f Field) Value(cfg Config) any {
	value := reflect.ValueOf(cfg).FieldByIndex(f.index).Interface()
	if duration, ok := value.(time.Duration); ok {
		return duration.String()
	}
	if values, ok := value.([]string); ok {
		return append([]string{}, values...)
	}
	return value
}

func (f Field) ParseEnv(raw string) (any, error) {
	switch f.Type {
	case "string":
		return raw, nil
	case "boolean":
		return strconv.ParseBool(raw)
	case "integer":
		return strconv.Atoi(raw)
	case "duration":
		return time.ParseDuration(raw)
	case "string_list":
		if raw == "" {
			return []string{}, nil
		}
		reader := csv.NewReader(strings.NewReader(raw))
		reader.FieldsPerRecord = -1
		records, err := reader.ReadAll()
		if err != nil || len(records) != 1 {
			return nil, errors.New("expected one comma-separated string list")
		}
		return records[0], nil
	default:
		return nil, errors.New("unsupported configuration type")
	}
}

func (f Field) Encode(value any) (string, error) {
	invalid := fmt.Errorf("%s requires %s", f.Path, f.Type)
	switch f.Type {
	case "string", "duration":
		text, ok := value.(string)
		if !ok {
			return "", invalid
		}
		if _, err := f.ParseEnv(text); err != nil {
			return "", invalid
		}
		return text, nil
	case "boolean":
		boolean, ok := value.(bool)
		if !ok {
			return "", invalid
		}
		return strconv.FormatBool(boolean), nil
	case "integer":
		switch number := value.(type) {
		case int:
			return strconv.Itoa(number), nil
		case float64:
			if math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number || math.Abs(number) > 9007199254740991 {
				return "", invalid
			}
			text := strconv.FormatFloat(number, 'f', 0, 64)
			if _, err := strconv.Atoi(text); err != nil {
				return "", invalid
			}
			return text, nil
		}
	case "string_list":
		var values []string
		switch list := value.(type) {
		case []string:
			values = list
		case []any:
			for _, entry := range list {
				text, ok := entry.(string)
				if !ok {
					return "", invalid
				}
				values = append(values, text)
			}
		default:
			return "", invalid
		}
		if len(values) == 0 {
			return "", nil
		}
		if len(values) == 1 && values[0] == "" {
			return `""`, nil
		}
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		if err := writer.Write(values); err != nil {
			return "", invalid
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return "", invalid
		}
		return strings.TrimSuffix(buffer.String(), "\n"), nil
	}
	return "", invalid
}
