package cdsvc

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	exposeAccessLocal  = "local"
	exposeAccessPublic = "public"
)

func exposeAccessOf(e model.VersionExpose) string {
	access := strings.ToLower(strings.TrimSpace(e.Access))
	if access == "" {
		return exposeAccessPublic
	}
	return access
}

func effectiveListen(e model.VersionExpose) int {
	if e.ListenPort != nil && *e.ListenPort > 0 {
		return *e.ListenPort
	}
	return e.ContainerPort
}

func tcpEntrypointName(listen int) string {
	return "tcp" + strconv.Itoa(listen)
}

// runtimeName builds cluster DNS / container_name: {app_code}-{component}.
func runtimeName(appCode, component string) (string, error) {
	raw := strings.ToLower(strings.TrimSpace(appCode) + "-" + strings.TrimSpace(component))
	var b strings.Builder
	prevDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "", fmt.Errorf("invalid runtime name for app=%q component=%q", appCode, component)
	}
	return out, nil
}

func publicTCPListens(exposes []model.VersionExpose) []int {
	seen := map[int]struct{}{}
	var out []int
	for _, e := range exposes {
		if exposeAccessOf(e) != exposeAccessPublic {
			continue
		}
		if strings.ToLower(strings.TrimSpace(e.Protocol)) != "tcp" {
			continue
		}
		listen := effectiveListen(e)
		if _, ok := seen[listen]; ok {
			continue
		}
		seen[listen] = struct{}{}
		out = append(out, listen)
	}
	// sort ascending for stable compile
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
