package bootstrap

import (
	"os"
	osuser "os/user"
	"runtime"
	"strings"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
)

func localEnvironmentDisplay() environmentdto.LocalDisplaySnapshot {
	snapshot := environmentdto.LocalDisplaySnapshot{}
	switch runtime.GOOS {
	case "linux", "windows":
		snapshot.Platform = runtime.GOOS
	}
	if host, err := os.Hostname(); err == nil {
		snapshot.Host = strings.TrimSpace(host)
	}
	if current, err := osuser.Current(); err == nil {
		snapshot.Username = sshUsernameHint(current.Username)
	}
	return snapshot
}

func sshUsernameHint(username string) string {
	username = strings.TrimSpace(username)
	if index := strings.LastIndex(username, `\`); index >= 0 && index+1 < len(username) {
		return username[index+1:]
	}
	return username
}
