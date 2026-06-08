package templatex

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/flosch/pongo2/v6"
)

var (
	defaultFilterCallPattern  = regexp.MustCompile(`\|\s*(default|d)\(\s*([^)]*?)\s*\)`)
	defaultFilterColonPattern = regexp.MustCompile(`\|\s*(default|d)\s*:\s*([^|}\n]+)`)
	variablePattern           = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.]*)\s*\}\}`)
)

func init() {
	_ = pongo2.RegisterFilter("d", jinjaDefaultFilter)
	_ = pongo2.RegisterFilter("default", jinjaDefaultFilter)
	pongo2.SetAutoescape(false)
}

func Render(input string, values map[string]any) (string, error) {
	converted := convertJinjaSyntax(input)
	if missing := firstMissingVariable(converted, values); missing != "" {
		return "", fmt.Errorf("template render failed: variable %s is undefined", missing)
	}
	tmpl, err := pongo2.FromString(converted)
	if err != nil {
		return "", fmt.Errorf("template syntax error: %w", err)
	}

	out, err := tmpl.Execute(pongo2.Context(values))
	if err != nil {
		return "", fmt.Errorf("template render failed: %w", err)
	}
	return out, nil
}

func convertJinjaSyntax(input string) string {
	converted := defaultFilterCallPattern.ReplaceAllString(input, `|$1:$2`)
	converted = defaultFilterColonPattern.ReplaceAllString(converted, `|$1:$2`)
	return converted
}

func firstMissingVariable(input string, values map[string]any) string {
	for _, match := range variablePattern.FindAllStringSubmatch(input, -1) {
		if len(match) != 2 {
			continue
		}
		name := match[1]
		if strings.Contains(input, "for "+name+" in ") {
			continue
		}
		if !hasPath(values, strings.Split(name, ".")) {
			return name
		}
	}
	return ""
}

func hasPath(values any, parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	current, ok := values.(map[string]any)
	if !ok {
		return false
	}
	value, ok := current[parts[0]]
	if !ok {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	return hasPath(value, parts[1:])
}

func jinjaDefaultFilter(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if !hasValue(in) {
		return param, nil
	}
	return in, nil
}

func hasValue(value *pongo2.Value) bool {
	if value == nil || value.IsNil() {
		return false
	}
	if value.IsString() {
		return strings.TrimSpace(value.String()) != ""
	}
	return true
}
