package routehandler

import (
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
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
		CreatedAt:             transportresponse.FormatTime(route.CreatedAt),
		UpdatedAt:             transportresponse.FormatTime(route.UpdatedAt),
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
	return routev1.TraefikRouteListResp{Items: transportresponse.Ptrs(respItems), Total: int32(len(items))}
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
