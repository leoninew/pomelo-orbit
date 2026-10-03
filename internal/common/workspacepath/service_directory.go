package workspacepath

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var driveAbsolute = regexp.MustCompile(`^[A-Za-z]:/`)

// NormalizeServiceDirectory validates paths using the execution target's platform.
func NormalizeServiceDirectory(value, platform string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 2048 || strings.ContainsAny(value, "\x00\r\n") {
		return "", fmt.Errorf("deployment_directory must be an absolute path or ~/ path")
	}
	if platform == "" {
		platform = runtime.GOOS
	}
	if platform == "windows" {
		value = strings.ReplaceAll(value, "\\", "/")
	}
	home := IsHomeWorkspaceRoot(value)
	absolute := strings.HasPrefix(value, "/")
	if platform == "windows" {
		absolute = driveAbsolute.MatchString(value) || strings.HasPrefix(value, "//")
	}
	if !home && !absolute {
		return "", fmt.Errorf("deployment_directory must be an absolute path or ~/ path for %s", platform)
	}
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("deployment_directory contains an invalid separator")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", fmt.Errorf("deployment_directory cannot contain parent segments")
		}
	}
	if strings.HasPrefix(value, "//") {
		value = "//" + strings.TrimPrefix(path.Clean(value), "/")
	} else {
		value = path.Clean(value)
	}
	if value == "/" || platform == "windows" && len(value) == 2 && value[1] == ':' {
		return "", fmt.Errorf("deployment_directory cannot be a filesystem root")
	}
	return value, nil
}

func DefaultServiceDirectory(root, code string) string {
	return strings.TrimRight(strings.ReplaceAll(root, "\\", "/"), "/") + "/deployment/" + code
}

func OverlappingDirectories(a, b, platform string) bool {
	a, b = strings.TrimRight(filepath.ToSlash(a), "/"), strings.TrimRight(filepath.ToSlash(b), "/")
	if platform == "windows" {
		a, b = strings.ToLower(a), strings.ToLower(b)
	}
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
