package envfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/hashicorp/go-envparse"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Overrides are literal values: interpolation must not change saved secrets.
func Decode(content []byte) (map[string]string, error) {
	content = bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})
	values, err := envparse.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("parse ENV assignments: %w", err)
	}
	for key := range values {
		if !keyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid ENV key %s", key)
		}
	}
	return values, nil
}

func Encode(values map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		if !keyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid ENV key %s", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	for _, key := range keys {
		content.WriteString(key)
		content.WriteByte('=')
		if err := encoder.Encode(values[key]); err != nil {
			return nil, fmt.Errorf("encode ENV value for %s: %w", key, err)
		}
	}
	return content.Bytes(), nil
}
