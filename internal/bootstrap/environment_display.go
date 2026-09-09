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
		snapshot.Username = strings.TrimSpace(current.Username)
	}
	return snapshot
}
