package deploymentsvc

import (
	"strconv"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	exposeAccessLocal  = "local"
	exposeAccessPublic = "public"
)

func effectiveListen(e model.ServiceExpose) int {
	if e.ListenPort != nil {
		return *e.ListenPort
	}
	return e.ContainerPort
}

func tcpEntrypointName(listen int) string {
	return "tcp" + strconv.Itoa(listen)
}

// runtimeName builds cluster DNS / container_name: {app_code}-{component}.
func runtimeName(appCode, component string) string {
	return appCode + "-" + component
}
