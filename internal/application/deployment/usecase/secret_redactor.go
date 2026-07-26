package deploymentsvc

import (
	"bytes"
	"io"
	"regexp"
	"sort"
	"strings"
)

const redactedSecret = "[REDACTED]"

// SecretRedactor limits the lifetime and exposure of deployment-only values.
type SecretRedactor struct {
	values []string
	keys   []string
}

func NewSecretRedactor(values map[string]string) SecretRedactor {
	keys := make([]string, 0, len(values))
	uniqueValues := make(map[string]struct{}, len(values))
	for key, value := range values {
		keys = append(keys, key)
		if value != "" {
			uniqueValues[value] = struct{}{}
		}
	}
	sort.Strings(keys)
	items := make([]string, 0, len(uniqueValues))
	for value := range uniqueValues {
		items = append(items, value)
	}
	sort.Slice(items, func(i, j int) bool { return len(items[i]) > len(items[j]) })
	return SecretRedactor{keys: keys, values: items}
}

func (r SecretRedactor) RedactText(text string) string {
	for _, value := range r.values {
		text = strings.ReplaceAll(text, value, redactedSecret)
	}
	for _, key := range r.keys {
		pattern := regexp.MustCompile(`(?m)(^|[\s,{\[])` + regexp.QuoteMeta(key) + `=([^\s,}\]]*)`)
		text = pattern.ReplaceAllString(text, "${1}"+key+"="+redactedSecret)
	}
	return text
}

func (r SecretRedactor) RedactMap(value map[string]any) map[string]any {
	result := make(map[string]any, len(value))
	for key, item := range value {
		if containsSecretKey(r.keys, key) {
			result[key] = redactedSecret
			continue
		}
		switch typed := item.(type) {
		case string:
			result[key] = r.RedactText(typed)
		case map[string]any:
			result[key] = r.RedactMap(typed)
		case []any:
			result[key] = r.RedactSlice(typed)
		default:
			result[key] = item
		}
	}
	return result
}

func (r SecretRedactor) RedactSlice(value []any) []any {
	result := make([]any, 0, len(value))
	for _, item := range value {
		switch typed := item.(type) {
		case string:
			result = append(result, r.RedactText(typed))
		case map[string]any:
			result = append(result, r.RedactMap(typed))
		case []any:
			result = append(result, r.RedactSlice(typed))
		default:
			result = append(result, item)
		}
	}
	return result
}

func containsSecretKey(keys []string, key string) bool {
	for _, candidate := range keys {
		if candidate == key {
			return true
		}
	}
	return false
}

type secretRedactingWriter struct {
	destination io.Writer
	redactor    SecretRedactor
	buffer      bytes.Buffer
}

func (w *secretRedactingWriter) Write(data []byte) (int, error) {
	return w.buffer.Write(data)
}

func (w *secretRedactingWriter) Flush() error {
	_, err := io.WriteString(w.destination, w.redactor.RedactText(w.buffer.String()))
	return err
}
