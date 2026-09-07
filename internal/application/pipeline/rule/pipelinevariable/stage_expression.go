package pipelinevariable

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	stageOutputTagPattern = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)
	shellDefaultPattern   = regexp.MustCompile(`\$\{[A-Za-z][A-Za-z0-9_]*\s*(?::-|:=|-)`)
)

type stageVariableReference struct {
	Name       string
	HasDefault bool
	Default    string
}

// ValidateStageVariableExpressions enforces the variable expression contract
// accepted by Pipeline stages before rendering them.
func ValidateStageVariableExpressions(input string) error {
	_, err := extractStageVariableReferences(input)
	return err
}

func extractStageVariableReferences(input string) ([]stageVariableReference, error) {
	if shellDefaultPattern.MatchString(input) {
		return nil, fmt.Errorf("unsupported shell-style variable default; use {{ NAME | default: \"value\" }}")
	}

	result := make([]stageVariableReference, 0)
	for _, match := range stageOutputTagPattern.FindAllStringSubmatch(input, -1) {
		if len(match) != 2 {
			continue
		}
		expression := strings.TrimSpace(match[1])
		parts, err := splitStageLiquidFilters(expression)
		if err != nil {
			return nil, fmt.Errorf("invalid pipeline variable %q: %w", expression, err)
		}
		name := strings.TrimSpace(parts[0])
		if !pipelineVariableNamePattern.MatchString(name) {
			continue
		}
		switch len(parts) {
		case 1:
			result = append(result, stageVariableReference{Name: name})
		case 2:
			value, ok, err := parseStageDefaultFilter(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid pipeline variable %q: %w", expression, err)
			}
			if !ok {
				continue
			}
			result = append(result, stageVariableReference{Name: name, HasDefault: true, Default: value})
		}
	}
	return result, nil
}

func splitStageLiquidFilters(expression string) ([]string, error) {
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

func parseStageDefaultFilter(expression string) (string, bool, error) {
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
