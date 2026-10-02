package routehandler

import (
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
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
	return routev1.TraefikConfigResp{DashboardDomain: config.DashboardDomain, HttpsEnabled: config.HTTPSEnabled, InternalDomain: config.InternalDomain, ExternalDomain: config.ExternalDomain}
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
		changes = append(changes, routedto.RouteSyncChange{RouteId: item.RouteId, Enabled: item.Enabled})
	}
	return changes, nil
}

func routeSyncPreviewResponse(preview routedto.RouteSyncPreview) routev1.RouteSyncPreviewResp {
	items := make([]*routev1.RouteSyncPlanItemResp, 0, len(preview.Items))
	for _, item := range preview.Items {
		items = append(items, &routev1.RouteSyncPlanItemResp{RouteId: item.RouteId, RouteName: item.RouteName, Action: item.Action, Rule: routeSyncRuleResponse(item.Rule), CertType: item.CertType, AcmeChallenge: item.AcmeChallenge, BusinessHash: item.BusinessHash, PublicationHash: item.PublicationHash})
	}
	return routev1.RouteSyncPreviewResp{
		RouteIds:        preview.RouteIds,
		PublicationHash: preview.PublicationHash,
		BusinessHash:    preview.BusinessHash,
		Items:           items,
	}
}

func routeSyncConfirmResponse(result routedto.RouteSyncConfirmResult) routev1.RouteSyncConfirmResp {
	items := make([]*routev1.RouteSyncResultResp, 0, len(result.Results))
	for _, item := range result.Results {
		items = append(items, &routev1.RouteSyncResultResp{RouteId: item.RouteId, RouteName: item.RouteName, OperationId: item.OperationId, Code: item.Code, Error: item.Error, BusinessSave: item.BusinessSave, FileCommit: item.FileCommit, ConfigurationMatch: item.ConfigurationMatch, CertificateVerification: item.CertificateVerification, Recovery: item.Recovery, Cleanup: item.Cleanup})
	}
	return routev1.RouteSyncConfirmResp{Code: result.Code, Results: items}
}

func routeSyncRuleResponse(rule *routedto.RouteSyncRule) *routev1.RouteSyncRuleResp {
	if rule == nil {
		return nil
	}
	return &routev1.RouteSyncRuleResp{Protocol: rule.Protocol, Match: rule.Match, Target: rule.Target}
}
