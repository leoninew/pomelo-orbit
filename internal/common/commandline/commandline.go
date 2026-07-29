package commandline

import (
	"strings"

	"github.com/mattn/go-shellwords"
)

// Parse tokenizes a command line without expanding environment variables or executing substitutions.
func Parse(input string) ([]string, error) {
	parser := shellwords.NewParser()
	parser.ParseEnv = false
	parser.ParseBacktick = false
	return parser.Parse(input)
}

// Format returns a shell-quoted command line that parses back to the supplied argv.
func Format(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		parts = append(parts, quote(arg))
	}
	return strings.Join(parts, " ")
}

func quote(arg string) string {
	if arg == "" {
		return "''"
	}
	for _, r := range arg {
		if !isBarewordRune(r) {
			return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
		}
	}
	return arg
}

func isBarewordRune(r rune) bool {
	return r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r >= '0' && r <= '9' ||
		strings.ContainsRune("_+-=,./:@%", r)
}
