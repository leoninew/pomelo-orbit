package routehandler

import (
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	routev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/route"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func routeCreateInput(req *routev1.RouteCreateReq) routedto.RouteCreateInput {
	return routedto.RouteCreateInput{
		Name:                  req.Name,
		Protocol:              req.Protocol,
		Domain:                req.Domain,
		PathPrefix:            req.PathPrefix,
		TargetUrl:             req.TargetUrl,
		ListenPort:            optionalInt(req.ListenPort),
		ServiceId:             req.GetServiceId(),
		ComponentName:         req.GetComponentName(),
		EndpointProtocol:      req.GetEndpointProtocol(),
		EndpointContainerPort: optionalInt(req.EndpointContainerPort),
		Enabled:               req.Enabled,
	}
}

func routeUpdateInput(req *routev1.RouteUpdateReq) routedto.RouteUpdateInput {
	return routedto.RouteUpdateInput{
		Name:                  req.Name,
		Protocol:              req.Protocol,
		Domain:                req.Domain,
		PathPrefix:            req.PathPrefix,
		TargetUrl:             req.TargetUrl,
		ListenPort:            optionalInt(req.ListenPort),
		ServiceId:             req.ServiceId,
		ComponentName:         req.ComponentName,
		EndpointProtocol:      req.EndpointProtocol,
		EndpointContainerPort: optionalInt(req.EndpointContainerPort),
		Enabled:               req.Enabled,
	}
}

func routeResponses(items []model.Route) []routev1.RouteResp {
	resp := make([]routev1.RouteResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, routeResponse(item))
	}
	return resp
}

func routeResponse(route model.Route) routev1.RouteResp {
	return routev1.RouteResp{
		Id:                    route.Id,
		Name:                  route.Name,
		Protocol:              route.Protocol,
		Domain:                route.Domain,
		PathPrefix:            route.PathPrefix,
		TargetUrl:             route.TargetUrl,
		ListenPort:            optionalInt32(route.ListenPort),
		ServiceId:             route.ServiceId,
		ComponentName:         route.ComponentName,
		EndpointProtocol:      route.EndpointProtocol,
		EndpointContainerPort: optionalInt32(route.EndpointContainerPort),
		Enabled:               route.Enabled,
		HttpsEnabled:          route.HTTPSEnabled,
		CertType:              route.CertType,
		AcmeChallenge:         route.AcmeChallenge,
		GatewayApplicationId:  route.GatewayApplicationId,
		Http01Available:       route.HTTP01Available,
		Dns01Available:        route.DNS01Available,
		AcmeChallengeHint:     route.ACMEChallengeHint,
		CreatedAt:             transport.FormatTime(route.CreatedAt),
		UpdatedAt:             transport.FormatTime(route.UpdatedAt),
	}
}

func optionalInt(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func optionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func traefikConfigResponse(config routedto.TraefikConfigView) routev1.TraefikConfigResp {
	return routev1.TraefikConfigResp{DashboardDomain: config.DashboardDomain, HttpsEnabled: config.HTTPSEnabled, BaseDomain: config.BaseDomain}
}

func traefikRouteListResponse(items []routeport.TraefikRouter) routev1.TraefikRouteListResp {
	respItems := make([]routev1.TraefikRouterResp, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, traefikRouterResponse(item))
	}
	return routev1.TraefikRouteListResp{Items: transport.Ptrs(respItems), Total: int32(len(items))}
}

func traefikRouterResponse(router routeport.TraefikRouter) routev1.TraefikRouterResp {
	return routev1.TraefikRouterResp{
		Name:        router.Name,
		Provider:    router.Provider,
		Status:      router.Status,
		Rule:        router.Rule,
		Service:     router.Service,
		Entrypoints: append([]string(nil), router.Entrypoints...),
		Tls:         router.TLS,
	}
}

func routeSyncChanges(items []*routev1.RouteSyncChange) ([]routedto.RouteSyncChange, error) {
	changes := make([]routedto.RouteSyncChange, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		change := routedto.RouteSyncChange{RouteId: item.RouteId, Enabled: item.Enabled}
		if item.Certificate != nil {
			certificate := &routedto.RouteSyncCertificateChange{
				Mode: item.Certificate.Mode, Challenge: item.Certificate.Challenge,
			}
			if item.Certificate.Pem != "" {
				certPEM, certKey, ok := splitPEM([]byte(item.Certificate.Pem))
				if !ok {
					return nil, apperror.New(apperror.KindValidation, "Invalid PEM certificate")
				}
				certificate.CertPEM, certificate.CertKey = certPEM, certKey
			}
			change.Certificate = certificate
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func routeSyncPreviewResponse(preview routedto.RouteSyncPreview) routev1.RouteSyncPreviewResp {
	differences := make([]*routev1.RouteSyncDiffResp, 0, len(preview.Differences))
	for _, difference := range preview.Differences {
		differences = append(differences, &routev1.RouteSyncDiffResp{
			Action:        difference.Action,
			RouteName:     difference.RouteName,
			Field:         difference.Field,
			BusinessValue: difference.BusinessValue,
			TraefikValue:  difference.TraefikValue,
		})
	}
	pending := make([]*routev1.RouteSyncPendingResp, 0, len(preview.Pending))
	for _, item := range preview.Pending {
		pending = append(pending, &routev1.RouteSyncPendingResp{
			RouteName: item.RouteName, Mode: item.Mode, Challenge: item.Challenge, PublishesNow: item.PublishesNow,
		})
	}
	return routev1.RouteSyncPreviewResp{
		BusinessHash: preview.BusinessHash,
		TraefikHash:  preview.TraefikHash,
		Matched:      preview.Matched,
		Differences:  differences,
		Pending:      pending,
	}
}
