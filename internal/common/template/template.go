package templatex

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/osteele/liquid"
)

var (
	outputTagPattern         = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.]*)([^}]*)\}\}`)
	conditionVariablePattern = regexp.MustCompile(`\{%\s*(?:if|elsif|unless)\s+([A-Za-z_][A-Za-z0-9_.]*)`)
	forVariablePattern       = regexp.MustCompile(`\{%\s*for\s+([A-Za-z_][A-Za-z0-9_]*)\s+in\s+([A-Za-z_][A-Za-z0-9_.]*)`)
)

func Render(input string, values map[string]any) (string, error) {
	if missing := firstMissingVariable(input, values); missing != "" {
		return "", fmt.Errorf("template render failed: variable %s is undefined", missing)
	}

	engine := liquid.NewEngine()
	out, err := engine.ParseAndRenderString(input, values)
	if err != nil {
		return "", fmt.Errorf("template render failed: %w", err)
	}
	return out, nil
}

func firstMissingVariable(input string, values map[string]any) string {
	localVariables := loopVariables(input)
	for _, name := range templateVariables(input) {
		if localVariables[name] {
			continue
		}
		if !hasPath(values, strings.Split(name, ".")) {
			return name
		}
	}
	return ""
}

func templateVariables(input string) []string {
	seen := map[string]bool{}
	variables := make([]string, 0)
	for _, match := range outputTagPattern.FindAllStringSubmatch(input, -1) {
		if len(match) != 3 || seen[match[1]] || hasDefaultFilter(match[2]) {
			continue
		}
		seen[match[1]] = true
		variables = append(variables, match[1])
	}
	for _, match := range conditionVariablePattern.FindAllStringSubmatch(input, -1) {
		if len(match) != 2 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		variables = append(variables, match[1])
	}
	for _, match := range forVariablePattern.FindAllStringSubmatch(input, -1) {
		if len(match) != 3 || seen[match[2]] {
			continue
		}
		seen[match[2]] = true
		variables = append(variables, match[2])
	}
	return variables
}

func hasDefaultFilter(expression string) bool {
	for _, part := range strings.Split(expression, "|") {
		if strings.HasPrefix(strings.TrimSpace(part), "default:") {
			return true
		}
	}
	return false
}

func loopVariables(input string) map[string]bool {
	variables := map[string]bool{}
	for _, match := range forVariablePattern.FindAllStringSubmatch(input, -1) {
		if len(match) == 3 {
			variables[match[1]] = true
		}
	}
	return variables
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
