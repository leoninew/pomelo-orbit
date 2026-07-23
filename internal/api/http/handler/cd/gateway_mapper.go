package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
)

func gatewayCreateInput(req *pomeloorbit.GatewayCreateReq) cddto.GatewayCreateInput {
	return cddto.GatewayCreateInput{
		ProjectId:         req.ProjectId,
		Code:              req.Code,
		Name:              req.Name,
		RestApiUrl:        req.RestApiUrl,
		BaseDomain:        req.BaseDomain,
		Image:             req.Image,
		ImagePullPolicy:   req.ImagePullPolicy,
		DefaultEntrypoint: req.DefaultEntrypoint,
		TLSMode:           req.TlsMode,
	}
}

func gatewayUpdateInput(req *pomeloorbit.GatewayUpdateReq) cddto.GatewayUpdateInput {
	return cddto.GatewayUpdateInput{
		Name:              req.Name,
		RestApiUrl:        req.RestApiUrl,
		BaseDomain:        req.BaseDomain,
		Image:             req.Image,
		ImagePullPolicy:   req.ImagePullPolicy,
		DefaultEntrypoint: req.DefaultEntrypoint,
		TLSMode:           req.TlsMode,
	}
}

func gatewayResponses(items []cddto.GatewayView) []pomeloorbit.GatewayResp {
	resp := make([]pomeloorbit.GatewayResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, gatewayResponse(item))
	}
	return resp
}

func gatewayResponse(view cddto.GatewayView) pomeloorbit.GatewayResp {
	app := view.Application
	cfg := view.Config
	projectId := ""
	if app.ProjectId != nil {
		projectId = *app.ProjectId
	}
	exposures := make([]*pomeloorbit.GatewayExposureItem, 0, len(view.Exposures))
	for _, item := range view.Exposures {
		exposures = append(exposures, &pomeloorbit.GatewayExposureItem{
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
	return pomeloorbit.GatewayResp{
		Id:                app.Id,
		ProjectId:         projectId,
		Code:              app.Code,
		Name:              app.Name,
		Kind:              app.Kind,
		RestApiUrl:        cfg.RestApiUrl,
		BaseDomain:        cfg.BaseDomain,
		Image:             cfg.Image,
		CreatedAt:         transportresponse.FormatTime(app.CreatedAt),
		UpdatedAt:         transportresponse.FormatTime(app.UpdatedAt),
		ConfigUpdatedAt:   transportresponse.FormatTime(cfg.UpdatedAt),
		ImagePullPolicy:   app.ImagePullPolicy,
		DefaultEntrypoint: cfg.DefaultEntrypoint,
		TlsMode:           cfg.TLSMode,
		Exposures:         exposures,
	}
}
