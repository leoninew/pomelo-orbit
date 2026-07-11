package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func routeCreateInput(req *pomeloorbit.RouteCreateReq) cddto.RouteCreateInput {
	return cddto.RouteCreateInput{
		Name:       req.Name,
		Domain:     req.Domain,
		PathPrefix: req.PathPrefix,
		TargetURL:  req.TargetUrl,
		Enabled:    req.Enabled,
	}
}

func routeUpdateInput(req *pomeloorbit.RouteUpdateReq) cddto.RouteUpdateInput {
	return cddto.RouteUpdateInput{
		Name:       req.Name,
		Domain:     req.Domain,
		PathPrefix: req.PathPrefix,
		TargetURL:  req.TargetUrl,
		Enabled:    req.Enabled,
	}
}

func routeResponses(items []model.Route) []pomeloorbit.RouteResp {
	resp := make([]pomeloorbit.RouteResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, routeResponse(item))
	}
	return resp
}

func routeResponse(route model.Route) pomeloorbit.RouteResp {
	return pomeloorbit.RouteResp{
		Id:           route.Id,
		Name:         route.Name,
		Domain:       route.Domain,
		PathPrefix:   route.PathPrefix,
		TargetUrl:    route.TargetURL,
		Enabled:      route.Enabled,
		HttpsEnabled: route.HTTPSEnabled,
		CertType:     route.CertType,
		CreatedAt:    transportresponse.FormatTime(route.CreatedAt),
		UpdatedAt:    transportresponse.FormatTime(route.UpdatedAt),
	}
}

func traefikConfigResponse(config cddto.TraefikConfigResp) pomeloorbit.TraefikConfigResp {
	return pomeloorbit.TraefikConfigResp{DashboardDomain: config.DashboardDomain, HttpsEnabled: config.HTTPSEnabled}
}

func traefikRouteListResponse(resp cddto.TraefikRouteListResp) pomeloorbit.TraefikRouteListResp {
	items := make([]pomeloorbit.TraefikRouterResp, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, traefikRouterResponse(item))
	}
	return pomeloorbit.TraefikRouteListResp{Items: transportresponse.Ptrs(items), Total: int32(resp.Total)}
}

func traefikRouterResponse(router cddto.TraefikRouterResp) pomeloorbit.TraefikRouterResp {
	return pomeloorbit.TraefikRouterResp{
		Name:        router.Name,
		Provider:    router.Provider,
		Status:      router.Status,
		Rule:        router.Rule,
		Service:     router.Service,
		Entrypoints: append([]string(nil), router.Entrypoints...),
		Tls:         router.TLS,
	}
}
