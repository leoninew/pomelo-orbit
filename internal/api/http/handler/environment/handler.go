package environmenthandler

import (
	"log/slog"
	"os"
	osuser "os/user"
	"runtime"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       environmentsvc.Service
	authenticator security.Authenticator
	localTarget   localTargetInfo
}

func New(logger *slog.Logger, service environmentsvc.Service, authenticator security.Authenticator, localWorkspaceRoot string) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator, localTarget: resolveLocalTargetInfo(localWorkspaceRoot)}
}

type localTargetInfo struct {
	WorkspaceRoot string
	Platform      string
	Host          string
	Username      string
}

func resolveLocalTargetInfo(workspaceRoot string) localTargetInfo {
	result := localTargetInfo{WorkspaceRoot: workspaceRoot}
	switch runtime.GOOS {
	case "linux", "windows":
		result.Platform = runtime.GOOS
	}
	if host, err := os.Hostname(); err == nil {
		result.Host = strings.TrimSpace(host)
	}
	if current, err := osuser.Current(); err == nil {
		result.Username = strings.TrimSpace(current.Username)
	}
	return result
}
