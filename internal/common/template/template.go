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
	pipelineVariableName     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	pipelineOutputTagPattern = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)
	shellDefaultPattern      = regexp.MustCompile(`\$\{[A-Za-z][A-Za-z0-9_]*\s*(?::-|:=|-)`)
)

// VariableReference is the subset of Liquid output expressions supported by
// pipeline stage variables. The default is positional: it belongs to the
// exact stage expression where it appears rather than to the global variable.
type VariableReference struct {
	Name       string
	HasDefault bool
	Default    string
}

// ExtractPipelineVariableReferences extracts the supported pipeline variable
// expressions from a stage field. General Liquid rendering remains available
// through Render; this function only defines the pipeline variable contract.
func ExtractPipelineVariableReferences(input string) ([]VariableReference, error) {
	if shellDefaultPattern.MatchString(input) {
		return nil, fmt.Errorf("unsupported shell-style variable default; use {{ NAME | default: \"value\" }}")
	}

	result := make([]VariableReference, 0)
	for _, match := range pipelineOutputTagPattern.FindAllStringSubmatch(input, -1) {
		if len(match) != 2 {
			continue
		}
		expression := strings.TrimSpace(match[1])
		parts, err := splitLiquidFilters(expression)
		if err != nil {
			return nil, fmt.Errorf("invalid pipeline variable %q: %w", expression, err)
		}
		name := strings.TrimSpace(parts[0])
		if !pipelineVariableName.MatchString(name) {
			continue
		}
		switch len(parts) {
		case 1:
			result = append(result, VariableReference{Name: name})
		case 2:
			value, ok, err := parseDefaultFilter(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid pipeline variable %q: %w", expression, err)
			}
			if !ok {
				continue
			}
			result = append(result, VariableReference{Name: name, HasDefault: true, Default: value})
		default:
			continue
		}
	}
	return result, nil
}

func splitLiquidFilters(expression string) ([]string, error) {
	parts := make([]string, 0, 2)
	var current strings.Builder
	var quote rune
	escaped := false
	for _, character := range expression {
		if quote != 0 {
			current.WriteRune(character)
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		switch character {
		case '\'', '"':
			quote = character
			current.WriteRune(character)
		case '|':
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(character)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted literal")
	}
	parts = append(parts, current.String())
	return parts, nil
}

func parseDefaultFilter(expression string) (string, bool, error) {
	expression = strings.TrimSpace(expression)
	if !strings.HasPrefix(expression, "default") {
		return "", false, nil
	}
	remainder := strings.TrimSpace(strings.TrimPrefix(expression, "default"))
	if remainder == "" || !strings.HasPrefix(remainder, ":") {
		return "", false, fmt.Errorf("default filter must use a quoted literal")
	}
	literal := strings.TrimSpace(strings.TrimPrefix(remainder, ":"))
	if len(literal) < 2 || (literal[0] != '\'' && literal[0] != '"') || literal[len(literal)-1] != literal[0] {
		return "", false, fmt.Errorf("default filter must use a quoted literal")
	}
	return literal[1 : len(literal)-1], true, nil
}

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
