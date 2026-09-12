package gatewayhandler

import (
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	gatewayv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/gateway"
)

func gatewayUpdateInput(req *gatewayv1.GatewayUpdateReq) gatewaydto.GatewayUpdateInput {
	return gatewaydto.GatewayUpdateInput{
		Name:                    req.Name,
		RestApiUrl:              req.RestApiUrl,
		RestReadyTimeoutSeconds: intPointer(req.RestReadyTimeoutSeconds),
		BaseDomain:              req.BaseDomain,
		DefaultEntrypoint:       req.DefaultEntrypoint,
		TLSMode:                 req.TlsMode,
		AcmeProfile:             req.AcmeProfile,
		AcmeEmail:               req.AcmeEmail,
		DNSApiToken:             req.DnsApiToken,
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
	bindings := make([]*gatewayv1.GatewayVersionBinding, 0, len(cfg.VersionBindings))
	for _, binding := range cfg.VersionBindings {
		bindings = append(bindings, &gatewayv1.GatewayVersionBinding{Profile: binding.Profile, VersionId: binding.VersionId})
	}
	defaultServiceID, defaultServiceInstanceKey, defaultServiceCode, defaultServiceStatus := "", "", "", ""
	if view.DefaultService != nil {
		defaultServiceID = view.DefaultService.Id
		defaultServiceInstanceKey = view.DefaultService.InstanceKey
		defaultServiceCode = view.DefaultService.Code
		defaultServiceStatus = view.DefaultService.Status
	}
	return gatewayv1.GatewayResp{
		Id:                        app.Id,
		ProjectId:                 projectId,
		Code:                      app.Code,
		Name:                      app.Name,
		Kind:                      app.Kind,
		RestApiUrl:                cfg.RestApiUrl,
		BaseDomain:                cfg.BaseDomain,
		CreatedAt:                 transportresponse.FormatTime(app.CreatedAt),
		UpdatedAt:                 transportresponse.FormatTime(app.UpdatedAt),
		ConfigUpdatedAt:           transportresponse.FormatTime(cfg.UpdatedAt),
		DefaultEntrypoint:         cfg.DefaultEntrypoint,
		TlsMode:                   cfg.TLSMode,
		Exposures:                 exposures,
		DefaultServiceId:          defaultServiceID,
		DefaultServiceInstanceKey: defaultServiceInstanceKey,
		DefaultServiceCode:        defaultServiceCode,
		DefaultServiceStatus:      defaultServiceStatus,
		RestReadyTimeoutSeconds:   int32(cfg.RestReadyTimeoutSeconds),
		AcmeEmail:                 cfg.AcmeEmail,
		AcmeProfile:               cfg.AcmeProfile,
		DnsApiToken:               cfg.DNSApiToken,
		VersionBindings:           bindings,
	}
}

func intPointer(value *int32) *int {
	if value == nil {
		return nil
	}
	result := int(*value)
	return &result
}
