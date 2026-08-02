package gatewayhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	gatewaydto "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/dto"
	gatewayv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/gateway"
)

func gatewayCreateInput(req *gatewayv1.GatewayCreateReq) gatewaydto.GatewayCreateInput {
	return gatewaydto.GatewayCreateInput{
		ProjectId:                  req.ProjectId,
		Code:                       req.Code,
		Name:                       req.Name,
		RestApiUrl:                 req.RestApiUrl,
		BaseDomain:                 req.BaseDomain,
		InitialComponentImage:      req.InitialComponentImage,
		InitialComponentPullPolicy: req.InitialComponentPullPolicy,
		DefaultEntrypoint:          req.DefaultEntrypoint,
		TLSMode:                    req.TlsMode,
	}
}

func gatewayUpdateInput(req *gatewayv1.GatewayUpdateReq) gatewaydto.GatewayUpdateInput {
	return gatewaydto.GatewayUpdateInput{
		Name:              req.Name,
		RestApiUrl:        req.RestApiUrl,
		BaseDomain:        req.BaseDomain,
		DefaultEntrypoint: req.DefaultEntrypoint,
		TLSMode:           req.TlsMode,
	}
}

func gatewayResponses(items []gatewaydto.GatewayView) []gatewayv1.GatewayResp {
	resp := make([]gatewayv1.GatewayResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, gatewayResponse(item))
	}
	return resp
}

func gatewayResponse(view gatewaydto.GatewayView) gatewayv1.GatewayResp {
	app := view.Application
	cfg := view.Config
	projectId := ""
	if app.ProjectId != nil {
		projectId = *app.ProjectId
	}
	exposures := make([]*gatewayv1.GatewayExposureItem, 0, len(view.Exposures))
	for _, item := range view.Exposures {
		exposures = append(exposures, &gatewayv1.GatewayExposureItem{
			ApplicationId:   item.ApplicationId,
			ApplicationCode: item.ApplicationCode,
			ComponentName:   item.ComponentName,
			Protocol:        item.Protocol,
			Access:          item.Access,
			ContainerPort:   int32(item.ContainerPort),
			ListenPort:      int32(item.ListenPort),
			PublicHost:      item.PublicHost,
			InternalDns:     item.InternalDns,
			ClientHint:      item.ClientHint,
		})
	}
	return gatewayv1.GatewayResp{
		Id:                app.Id,
		ProjectId:         projectId,
		Code:              app.Code,
		Name:              app.Name,
		Kind:              app.Kind,
		RestApiUrl:        cfg.RestApiUrl,
		BaseDomain:        cfg.BaseDomain,
		CreatedAt:         transportresponse.FormatTime(app.CreatedAt),
		UpdatedAt:         transportresponse.FormatTime(app.UpdatedAt),
		ConfigUpdatedAt:   transportresponse.FormatTime(cfg.UpdatedAt),
		DefaultEntrypoint: cfg.DefaultEntrypoint,
		TlsMode:           cfg.TLSMode,
		Exposures:         exposures,
	}
}
