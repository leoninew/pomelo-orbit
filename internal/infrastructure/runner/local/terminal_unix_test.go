//go:build !windows

package localrunner

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func terminalChildCommand(pidPath string) string {
	return `sh -c 'echo $$ > "$1"; exec sleep 60' sh '` + strings.ReplaceAll(pidPath, "'", "'\"'\"'") + `'`
}

func terminalTestProcessAlive(pid int) bool {
	if errors.Is(unix.Kill(pid, 0), unix.ESRCH) {
		return false
	}
	// Container PID 1 may not reap promptly; a zombie has already terminated.
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err == nil {
		if index := strings.LastIndex(string(data), ") "); index >= 0 && strings.HasPrefix(string(data[index+2:]), "Z ") {
			return false
		}
	}
	return true
}
