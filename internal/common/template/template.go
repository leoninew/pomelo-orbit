package templatex

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/osteele/liquid"
)

var (
	outputTagPattern         = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.]*)([^}]*)\}\}`)
	conditionVariablePattern = regexp.MustCompile(`\{%\s*(?:if|elsif|unless)\s+([A-Za-z_][A-Za-z0-9_.]*)`)
	forVariablePattern       = regexp.MustCompile(`\{%\s*for\s+([A-Za-z_][A-Za-z0-9_]*)\s+in\s+([A-Za-z_][A-Za-z0-9_.]*)`)
	liquidOutputTagPattern   = regexp.MustCompile(`(?s)\{\{(.*?)\}\}`)
	shellDefaultPattern      = regexp.MustCompile(`\$\{[A-Za-z][A-Za-z0-9_]*\s*(?::-|:=|-)`)
	variableNamePattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
)

// VariableReference is a simple Liquid output variable and its optional
// literal default filter.
type VariableReference struct {
	Name       string
	HasDefault bool
	Default    string
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

// IsVariableName reports whether name can be used as a simple Liquid output
// variable by the configuration APIs.
func IsVariableName(name string) bool {
	return variableNamePattern.MatchString(name)
}

// ExtractVariableReferences finds simple Liquid output variables used by a
// full Liquid template. It recognizes the optional default filter only when
// its argument is a quoted literal. Other valid Liquid expressions are left to
// Render and are not treated as configurable variable references.
func ExtractVariableReferences(input string) ([]VariableReference, error) {
	if shellDefaultPattern.MatchString(input) {
		return nil, fmt.Errorf("unsupported shell-style variable default; use {{ NAME | default: \"value\" }}")
	}

	result := make([]VariableReference, 0)
	for _, match := range liquidOutputTagPattern.FindAllStringSubmatch(input, -1) {
		if len(match) != 2 {
			continue
		}
		parts, err := splitLiquidFilters(strings.TrimSpace(match[1]))
		if err != nil {
			return nil, err
		}
		name := strings.TrimSpace(parts[0])
		if !IsVariableName(name) {
			continue
		}
		switch len(parts) {
		case 1:
			result = append(result, VariableReference{Name: name})
		case 2:
			value, ok, err := parseDefaultLiteral(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid variable %q: %w", name, err)
			}
			if ok {
				result = append(result, VariableReference{Name: name, HasDefault: true, Default: value})
			}
		}
	}
	return result, nil
}

// ValidateVariableExpressions validates the configurable-variable subset of a
// full Liquid template without changing Render's broader Liquid support.
func ValidateVariableExpressions(input string) error {
	_, err := ExtractVariableReferences(input)
	return err
}

// ValidateLiteralValue rejects Liquid delimiters in values that must remain
// literal, such as a configured variable default.
func ValidateLiteralValue(input string) error {
	if strings.Contains(input, "{{") || strings.Contains(input, "}}") || strings.Contains(input, "{%") || strings.Contains(input, "%}") {
		return fmt.Errorf("literal value cannot contain Liquid syntax")
	}
	return nil
}

// ResolveNestedValues recursively expands simple {{ NAME }} references in
// named values. roots are immutable, already-resolved values; templates and
// roots must not use the same name. Full Liquid syntax is intentionally not
// accepted here so callers get deterministic dependency validation.
func ResolveNestedValues(templates map[string]string, roots map[string]string) (map[string]string, error) {
	for name := range templates {
		if _, exists := roots[name]; exists {
			return nil, fmt.Errorf("nested variable %s is declared as both a template and root", name)
		}
	}

	resolved := make(map[string]string, len(templates))
	visiting := make(map[string]bool, len(templates))
	stack := make([]string, 0, len(templates))
	var resolve func(string) (string, error)
	resolve = func(name string) (string, error) {
		if value, exists := roots[name]; exists {
			return value, nil
		}
		if value, exists := resolved[name]; exists {
			return value, nil
		}
		if visiting[name] {
			start := 0
			for index, item := range stack {
				if item == name {
					start = index
					break
				}
			}
			cycle := append(append([]string{}, stack[start:]...), name)
			return "", fmt.Errorf("nested variable cycle: %s", strings.Join(cycle, " -> "))
		}
		input, exists := templates[name]
		if !exists {
			return "", fmt.Errorf("nested variable %s references unknown variable %s", currentNestedVariable(stack), name)
		}

		segments, err := nestedValueSegments(input)
		if err != nil {
			return "", fmt.Errorf("invalid nested value for variable %s: %w", name, err)
		}
		visiting[name] = true
		stack = append(stack, name)
		defer func() {
			stack = stack[:len(stack)-1]
			visiting[name] = false
		}()

		var output strings.Builder
		for _, segment := range segments {
			if segment.reference == "" {
				output.WriteString(segment.literal)
				continue
			}
			value, err := resolve(segment.reference)
			if err != nil {
				return "", err
			}
			output.WriteString(value)
		}
		value := output.String()
		resolved[name] = value
		return value, nil
	}

	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := resolve(name); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

type nestedValueSegment struct {
	literal   string
	reference string
}

func nestedValueSegments(input string) ([]nestedValueSegment, error) {
	if strings.Contains(input, "{%") || strings.Contains(input, "%}") {
		return nil, fmt.Errorf("nested Liquid values only support simple {{ NAME }} references")
	}

	segments := make([]nestedValueSegment, 0)
	for position := 0; position < len(input); {
		openOffset := strings.Index(input[position:], "{{")
		closeOffset := strings.Index(input[position:], "}}")
		if closeOffset >= 0 && (openOffset < 0 || closeOffset < openOffset) {
			return nil, fmt.Errorf("nested Liquid value contains an incomplete output tag")
		}
		if openOffset < 0 {
			segments = append(segments, nestedValueSegment{literal: input[position:]})
			break
		}
		open := position + openOffset
		if open > position {
			segments = append(segments, nestedValueSegment{literal: input[position:open]})
		}
		close := strings.Index(input[open+2:], "}}")
		if close < 0 {
			return nil, fmt.Errorf("nested Liquid value contains an incomplete output tag")
		}
		end := open + 2 + close
		name := strings.TrimSpace(input[open+2 : end])
		if !IsVariableName(name) {
			return nil, fmt.Errorf("nested Liquid values only support simple {{ NAME }} references")
		}
		segments = append(segments, nestedValueSegment{reference: name})
		position = end + 2
	}
	return segments, nil
}

func currentNestedVariable(stack []string) string {
	if len(stack) == 0 {
		return "<root>"
	}
	return stack[len(stack)-1]
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

func parseDefaultLiteral(expression string) (string, bool, error) {
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
