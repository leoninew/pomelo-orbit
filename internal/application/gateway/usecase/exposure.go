package gatewaysvc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	exposeAccessLocal  = "local"
	exposeAccessPublic = "public"
)

type exposeOccupancy struct {
	local          map[int]string
	publicPlainTCP map[int]string
	publicTLSTCP   map[string]string
	publicTCP      map[int]struct{}
	tlsMode        string
}

func (o *exposeOccupancy) publicTCPListens() []int {
	if o == nil {
		return nil
	}
	ports := make([]int, 0, len(o.publicTCP))
	for port := range o.publicTCP {
		ports = append(ports, port)
	}
	return normalizeTCPListens(ports)
}

// ActivePublicTCPListens returns active standard-application TCP listens for gateway compilation.
func (s Service) ActivePublicTCPListens(ctx context.Context, excludeApplicationId string) ([]int, error) {
	occupancy, err := s.buildExposeOccupancy(ctx, excludeApplicationId)
	if err != nil {
		return nil, err
	}
	return occupancy.publicTCPListens(), nil
}

// ValidateDeploymentExposureConflicts validates an application's candidate exposes against active services.
func (s Service) ValidateDeploymentExposureConflicts(ctx context.Context, app model.Application, exposes []model.VersionExpose, gateway *model.GatewayConfig) error {
	occupancy, err := s.buildExposeOccupancy(ctx, app.Id)
	if err != nil {
		return err
	}
	return validateDeploymentExposureConflicts(occupancy, app, exposes, gateway)
}

func (s Service) buildExposeOccupancy(ctx context.Context, excludeApplicationId string) (*exposeOccupancy, error) {
	tlsMode := "none"
	if cfg, err := s.config.ResolveActiveGatewayConfig(ctx); err == nil {
		tlsMode = strings.ToLower(strings.TrimSpace(cfg.TLSMode))
		if tlsMode == "" {
			tlsMode = "none"
		}
	}
	occupancy := &exposeOccupancy{
		local:          map[int]string{},
		publicPlainTCP: map[int]string{},
		publicTLSTCP:   map[string]string{},
		publicTCP:      map[int]struct{}{},
		tlsMode:        tlsMode,
	}
	apps, err := s.application.ListApplications(ctx, nil, 1, 10000, "", status.ApplicationKindStandard)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list applications for conflict check", err)
	}
	var gateway *model.GatewayConfig
	if cfg, err := s.config.ResolveActiveGatewayConfig(ctx); err == nil {
		gateway = &cfg
	}
	for _, app := range apps.Items {
		if excludeApplicationId != "" && app.Id == excludeApplicationId {
			continue
		}
		services, err := s.service.ListServicesByApplication(ctx, app.Id)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services for conflict check", err)
		}
		for _, service := range services {
			if !isActiveServiceStatus(service.Status) {
				continue
			}
			exposes, err := s.application.VersionExposesByVersion(ctx, service.VersionId)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindInternal, "Failed to list exposes for conflict check", err)
			}
			owner := app.Code + "/" + service.Id
			if err := occupancy.addExposes(app, exposes, gateway, owner); err != nil {
				return nil, err
			}
		}
	}
	return occupancy, nil
}

func (o *exposeOccupancy) addExposes(app model.Application, exposes []model.VersionExpose, gateway *model.GatewayConfig, owner string) error {
	for _, expose := range exposes {
		listen := effectiveListen(expose)
		access := exposeAccessOf(expose)
		protocol := strings.ToLower(strings.TrimSpace(expose.Protocol))
		switch access {
		case exposeAccessLocal:
			if previous, ok := o.local[listen]; ok {
				return apperror.New(apperror.KindValidation, fmt.Sprintf("local listen port %d already used by %s (conflict with %s)", listen, previous, owner))
			}
			o.local[listen] = owner
		case exposeAccessPublic:
			if protocol != "tcp" {
				continue
			}
			o.publicTCP[listen] = struct{}{}
			if o.tlsMode == "none" {
				if previous, ok := o.publicPlainTCP[listen]; ok {
					return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP listen port %d already used by %s (conflict with %s)", listen, previous, owner))
				}
				o.publicPlainTCP[listen] = owner
				continue
			}
			host := ""
			if gateway != nil {
				if value, err := deriveHost(gateway, app.Code); err == nil {
					host = value
				}
			}
			key := fmt.Sprintf("%d|%s", listen, host)
			if previous, ok := o.publicTLSTCP[key]; ok {
				return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP %s already used by %s (conflict with %s)", key, previous, owner))
			}
			o.publicTLSTCP[key] = owner
		}
	}
	return nil
}

func validateDeploymentExposureConflicts(occupancy *exposeOccupancy, app model.Application, exposes []model.VersionExpose, gateway *model.GatewayConfig) error {
	if occupancy == nil {
		occupancy = &exposeOccupancy{
			local:          map[int]string{},
			publicPlainTCP: map[int]string{},
			publicTLSTCP:   map[string]string{},
			publicTCP:      map[int]struct{}{},
			tlsMode:        "none",
		}
	}
	local := copyIntStringMap(occupancy.local)
	plainTCP := copyIntStringMap(occupancy.publicPlainTCP)
	tlsTCP := copyStringStringMap(occupancy.publicTLSTCP)
	tlsMode := occupancy.tlsMode
	if gateway != nil {
		if mode := strings.ToLower(strings.TrimSpace(gateway.TLSMode)); mode != "" {
			tlsMode = mode
		}
	}
	owner := app.Code + "/deploy"
	for _, expose := range exposes {
		listen := effectiveListen(expose)
		access := exposeAccessOf(expose)
		protocol := strings.ToLower(strings.TrimSpace(expose.Protocol))
		switch access {
		case exposeAccessLocal:
			if previous, ok := local[listen]; ok {
				return apperror.New(apperror.KindValidation, fmt.Sprintf("local listen port %d already used by %s", listen, previous))
			}
			local[listen] = owner
		case exposeAccessPublic:
			if protocol != "tcp" {
				continue
			}
			if tlsMode == "none" {
				if previous, ok := plainTCP[listen]; ok {
					return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP listen port %d already used by %s", listen, previous))
				}
				plainTCP[listen] = owner
				continue
			}
			host := ""
			if gateway != nil {
				if value, err := deriveHost(gateway, app.Code); err == nil {
					host = value
				}
			}
			key := fmt.Sprintf("%d|%s", listen, host)
			if previous, ok := tlsTCP[key]; ok {
				return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP %s already used by %s", key, previous))
			}
			tlsTCP[key] = owner
		}
	}
	return nil
}

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

func exposeAccessOf(expose model.VersionExpose) string {
	access := strings.ToLower(strings.TrimSpace(expose.Access))
	if access == "" {
		return exposeAccessPublic
	}
	return access
}

func effectiveListen(expose model.VersionExpose) int {
	if expose.ListenPort != nil && *expose.ListenPort > 0 {
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

func copyIntStringMap(input map[int]string) map[int]string {
	out := make(map[int]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func copyStringStringMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
