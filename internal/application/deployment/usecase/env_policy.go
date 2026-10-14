package deploymentsvc

import "regexp"

var (
	envPlaceholderRequired        = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)
	envPlaceholderDefault         = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}$`)
	envPlaceholderRequiredMessage = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*):\?[^}]*\}$`)
)

type placeholderNeed struct {
	Required   bool
	Default    string
	HasDefault bool
}

func parsePlaceholder(value string) (string, placeholderNeed, bool) {
	if m := envPlaceholderRequiredMessage.FindStringSubmatch(value); len(m) == 2 {
		return m[1], placeholderNeed{Required: true}, true
	}
	if m := envPlaceholderRequired.FindStringSubmatch(value); len(m) == 2 {
		return m[1], placeholderNeed{Required: true}, true
	}
	if m := envPlaceholderDefault.FindStringSubmatch(value); len(m) == 3 {
		return m[1], placeholderNeed{HasDefault: true, Default: m[2]}, true
	}
	return "", placeholderNeed{}, false
}
