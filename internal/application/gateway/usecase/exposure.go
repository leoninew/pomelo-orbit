package gatewaysvc

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	exposeAccessLocal  = "local"
	exposeAccessPublic = "public"
)

func normalizeTCPListens(listens []int) []int {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(listens))
	for _, port := range listens {
		if port < 1 || port > 65535 || port == 80 || port == 443 || port == 8080 {
			continue
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		out = append(out, port)
	}
	sort.Ints(out)
	return out
}

func exposeAccessOf(expose model.ServiceExpose) string {
	return strings.ToLower(strings.TrimSpace(expose.Access))
}

func effectiveListen(expose model.ServiceExpose) int {
	if expose.ListenPort != nil {
		return *expose.ListenPort
	}
	return expose.ContainerPort
}

func tcpEntrypointName(listen int) string {
	return "tcp" + strconv.Itoa(listen)
}

func runtimeName(appCode string, component string) (string, error) {
	raw := strings.ToLower(strings.TrimSpace(appCode) + "-" + strings.TrimSpace(component))
	var builder strings.Builder
	previousDash := false
	for _, char := range raw {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			builder.WriteRune(char)
			previousDash = false
		case char == '-' || char == '_' || unicode.IsSpace(char):
			if !previousDash && builder.Len() > 0 {
				builder.WriteByte('-')
				previousDash = true
			}
		default:
			if !previousDash && builder.Len() > 0 {
				builder.WriteByte('-')
				previousDash = true
			}
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "", fmt.Errorf("invalid runtime name for app=%q component=%q", appCode, component)
	}
	return out, nil
}

func deriveHost(gateway *model.GatewayConfig, appCode string) (string, error) {
	if gateway == nil {
		return "", fmt.Errorf("gateway config required for host derivation")
	}
	baseDomain := strings.TrimSpace(gateway.BaseDomain)
	if baseDomain == "" {
		return "", fmt.Errorf("gateway base_domain is required for host derivation")
	}
	code := strings.TrimSpace(appCode)
	if code == "" {
		return "", fmt.Errorf("application code is required for host derivation")
	}
	return code + "." + baseDomain, nil
}
