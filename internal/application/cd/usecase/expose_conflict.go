package cdsvc

import (
	"context"
	"fmt"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// exposeOccupancy tracks listen ports occupied by active services.
type exposeOccupancy struct {
	// localListen -> owner key
	local map[int]string
	// public plain TCP listen -> owner (tls_mode=none path)
	publicPlainTCP map[int]string
	// public TLS TCP: listen|host -> owner
	publicTLSTCP map[string]string
	// all public TCP listens (plain or TLS)
	publicTCP map[int]struct{}
	tlsMode   string
}

func (o *exposeOccupancy) PublicTCPListens() []int {
	if o == nil {
		return nil
	}
	out := make([]int, 0, len(o.publicTCP))
	for p := range o.publicTCP {
		out = append(out, p)
	}
	return normalizeTCPListens(out)
}

func (s Service) buildExposeOccupancy(ctx context.Context, excludeApplicationId string) (*exposeOccupancy, error) {
	if s.store == nil {
		return &exposeOccupancy{
			local:          map[int]string{},
			publicPlainTCP: map[int]string{},
			publicTLSTCP:   map[string]string{},
			publicTCP:      map[int]struct{}{},
		}, nil
	}
	tlsMode := "none"
	if cfg, err := s.store.ResolveActiveGatewayConfig(ctx); err == nil {
		tlsMode = strings.ToLower(strings.TrimSpace(cfg.TLSMode))
		if tlsMode == "" {
			tlsMode = "none"
		}
	}
	occ := &exposeOccupancy{
		local:          map[int]string{},
		publicPlainTCP: map[int]string{},
		publicTLSTCP:   map[string]string{},
		publicTCP:      map[int]struct{}{},
		tlsMode:        tlsMode,
	}
	apps, err := s.store.ListApplications(ctx, nil, 1, 10000, "", status.ApplicationKindStandard)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list applications for conflict check", err)
	}
	var gateway *model.GatewayConfig
	if cfg, err := s.store.ResolveActiveGatewayConfig(ctx); err == nil {
		gateway = &cfg
	}
	for _, app := range apps.Items {
		if excludeApplicationId != "" && app.Id == excludeApplicationId {
			continue
		}
		services, err := s.store.ListServicesByApplication(ctx, app.Id)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services for conflict check", err)
		}
		for _, svc := range services {
			if !isActiveServiceStatus(svc.Status) {
				continue
			}
			exposes, err := s.store.VersionExposesByVersion(ctx, svc.VersionId)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindInternal, "Failed to list exposes for conflict check", err)
			}
			owner := app.Code + "/" + svc.Id
			if err := occ.addExposes(app, exposes, gateway, owner); err != nil {
				return nil, err
			}
		}
	}
	return occ, nil
}

func (o *exposeOccupancy) addExposes(app model.Application, exposes []model.VersionExpose, gateway *model.GatewayConfig, owner string) error {
	for _, expose := range exposes {
		listen := effectiveListen(expose)
		access := exposeAccessOf(expose)
		protocol := strings.ToLower(strings.TrimSpace(expose.Protocol))
		switch access {
		case exposeAccessLocal:
			if prev, ok := o.local[listen]; ok {
				return apperror.New(apperror.KindValidation, fmt.Sprintf("local listen port %d already used by %s (conflict with %s)", listen, prev, owner))
			}
			o.local[listen] = owner
		case exposeAccessPublic:
			if protocol != "tcp" {
				continue
			}
			o.publicTCP[listen] = struct{}{}
			if o.tlsMode == "none" {
				if prev, ok := o.publicPlainTCP[listen]; ok {
					return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP listen port %d already used by %s (conflict with %s)", listen, prev, owner))
				}
				o.publicPlainTCP[listen] = owner
			} else {
				host := ""
				if gateway != nil {
					if h, err := deriveHost(gateway, app.Code); err == nil {
						host = h
					}
				}
				key := fmt.Sprintf("%d|%s", listen, host)
				if prev, ok := o.publicTLSTCP[key]; ok {
					return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP %s already used by %s (conflict with %s)", key, prev, owner))
				}
				o.publicTLSTCP[key] = owner
			}
		}
	}
	return nil
}

// validateDeployExposeConflicts checks candidate exposes against occupancy (self app already excluded).
func validateDeployExposeConflicts(occ *exposeOccupancy, app model.Application, exposes []model.VersionExpose, gateway *model.GatewayConfig) error {
	if occ == nil {
		occ = &exposeOccupancy{
			local:          map[int]string{},
			publicPlainTCP: map[int]string{},
			publicTLSTCP:   map[string]string{},
			publicTCP:      map[int]struct{}{},
			tlsMode:        "none",
		}
	}
	// clone maps so we can simulate adding candidate without mutating global
	local := copyIntStringMap(occ.local)
	plain := copyIntStringMap(occ.publicPlainTCP)
	tlsMap := copyStringStringMap(occ.publicTLSTCP)
	tlsMode := occ.tlsMode
	if gateway != nil {
		m := strings.ToLower(strings.TrimSpace(gateway.TLSMode))
		if m != "" {
			tlsMode = m
		}
	}
	owner := app.Code + "/deploy"
	for _, expose := range exposes {
		listen := effectiveListen(expose)
		access := exposeAccessOf(expose)
		protocol := strings.ToLower(strings.TrimSpace(expose.Protocol))
		switch access {
		case exposeAccessLocal:
			if prev, ok := local[listen]; ok {
				return apperror.New(apperror.KindValidation, fmt.Sprintf("local listen port %d already used by %s", listen, prev))
			}
			local[listen] = owner
		case exposeAccessPublic:
			if protocol != "tcp" {
				continue
			}
			if tlsMode == "none" {
				if prev, ok := plain[listen]; ok {
					return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP listen port %d already used by %s", listen, prev))
				}
				plain[listen] = owner
			} else {
				host := ""
				if gateway != nil {
					if h, err := deriveHost(gateway, app.Code); err == nil {
						host = h
					}
				}
				key := fmt.Sprintf("%d|%s", listen, host)
				if prev, ok := tlsMap[key]; ok {
					return apperror.New(apperror.KindValidation, fmt.Sprintf("public TCP %s already used by %s", key, prev))
				}
				tlsMap[key] = owner
			}
		}
	}
	return nil
}

func copyIntStringMap(in map[int]string) map[int]string {
	out := make(map[int]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyStringStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
