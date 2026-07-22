package cdhandler

import (
	"encoding/json"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func applicationImportInput(projectID string, req *pomeloorbit.ApplicationImportReq) cddto.ApplicationImportInput {
	components := make([]cddto.ComponentInput, 0, len(req.Components))
	for _, item := range req.Components {
		components = append(components, componentInput(item))
	}
	exposes := make([]cddto.ExposeInput, 0, len(req.Exposes))
	for _, item := range req.Exposes {
		exposes = append(exposes, exposeInput(item))
	}
	return cddto.ApplicationImportInput{
		ProjectId:       projectID,
		Name:            req.Name,
		Code:            req.Code,
		ImagePullPolicy: req.ImagePullPolicy,
		VersionLabel:    req.VersionLabel,
		VersionEnvJSON:  req.VersionEnvJson,
		VersionNote:     req.VersionNote,
		Components:      components,
		Exposes:         exposes,
	}
}

func applicationExportResponse(exported cddto.ApplicationExport) *pomeloorbit.ApplicationExportResp {
	resp := &pomeloorbit.ApplicationExportResp{
		Id:              exported.Application.Id,
		ProjectId:       exported.Application.ProjectId,
		Name:            exported.Application.Name,
		Code:            exported.Application.Code,
		ImagePullPolicy: exported.Application.ImagePullPolicy,
		Versions:        make([]*pomeloorbit.VersionResp, 0, len(exported.Versions)),
		Services:        make([]*pomeloorbit.ServiceResp, 0, len(exported.Services)),
	}
	for _, view := range exported.Versions {
		item := versionResponse(view)
		resp.Versions = append(resp.Versions, &item)
	}
	for _, svc := range exported.Services {
		item := serviceResponse(svc)
		resp.Services = append(resp.Services, &item)
	}
	return resp
}

func componentInput(req *pomeloorbit.ComponentReq) cddto.ComponentInput {
	return cddto.ComponentInput{
		Name:            req.Name,
		Image:           req.Image,
		CommandJSON:     req.CommandJson,
		ArgsJSON:        req.ArgsJson,
		EnvJSON:         req.EnvJson,
		PortsJSON:       req.PortsJson,
		MountsJSON:      req.MountsJson,
		NetworksJSON:    req.NetworksJson,
		DependsOnJSON:   req.DependsOnJson,
		HealthcheckJSON: req.HealthcheckJson,
		ResourcesJSON:   req.ResourcesJson,
		PullPolicy:      req.PullPolicy,
	}
}

func exposeInput(req *pomeloorbit.ExposeReq) cddto.ExposeInput {
	return cddto.ExposeInput{
		ComponentName: req.ComponentName,
		Protocol:      req.Protocol,
		ContainerPort: int(req.ContainerPort),
		PathPrefix:    req.PathPrefix,
	}
}

func versionCreateInput(req *pomeloorbit.VersionCreateReq) cddto.VersionCreateInput {
	components := make([]cddto.ComponentInput, 0, len(req.Components))
	for _, item := range req.Components {
		components = append(components, componentInput(item))
	}
	exposes := make([]cddto.ExposeInput, 0, len(req.Exposes))
	for _, item := range req.Exposes {
		exposes = append(exposes, exposeInput(item))
	}
	return cddto.VersionCreateInput{
		ApplicationId: req.ApplicationId,
		Label:         req.Label,
		EnvJSON:       req.EnvJson,
		Note:          req.Note,
		Components:    components,
		Exposes:       exposes,
	}
}

func versionUpdateInput(req *pomeloorbit.VersionUpdateReq) cddto.VersionUpdateInput {
	input := cddto.VersionUpdateInput{
		Label:   req.Label,
		EnvJSON: req.EnvJson,
		Note:    req.Note,
	}
	if req.Components != nil {
		components := make([]cddto.ComponentInput, 0, len(req.Components))
		for _, item := range req.Components {
			components = append(components, componentInput(item))
		}
		input.Components = &components
	}
	if req.Exposes != nil {
		exposes := make([]cddto.ExposeInput, 0, len(req.Exposes))
		for _, item := range req.Exposes {
			exposes = append(exposes, exposeInput(item))
		}
		input.Exposes = &exposes
	}
	return input
}

func versionResponses(views []cddto.VersionView) []pomeloorbit.VersionResp {
	resp := make([]pomeloorbit.VersionResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, versionResponse(view))
	}
	return resp
}

func versionResponse(view cddto.VersionView) pomeloorbit.VersionResp {
	components := make([]*pomeloorbit.ComponentResp, 0, len(view.Components))
	for _, component := range view.Components {
		item := componentResponse(component)
		components = append(components, &item)
	}
	exposes := make([]*pomeloorbit.ExposeResp, 0, len(view.Exposes))
	for _, expose := range view.Exposes {
		item := exposeResponse(expose)
		exposes = append(exposes, &item)
	}
	return pomeloorbit.VersionResp{
		Id:                   view.Version.Id,
		ApplicationId:        view.Version.ApplicationId,
		Label:                view.Version.Label,
		Status:               view.Version.Status,
		EnvJson:              view.Version.EnvJSON,
		CreatedFromVersionId: view.Version.CreatedFromVersionId,
		Note:                 view.Version.Note,
		CreatedAt:            transportresponse.FormatTime(view.Version.CreatedAt),
		UpdatedAt:            transportresponse.FormatTime(view.Version.UpdatedAt),
		Components:           components,
		Exposes:              exposes,
	}
}

func componentResponse(component model.Component) pomeloorbit.ComponentResp {
	return pomeloorbit.ComponentResp{
		Id:              component.Id,
		VersionId:       component.VersionId,
		Name:            component.Name,
		Image:           component.Image,
		CommandJson:     component.CommandJSON,
		ArgsJson:        component.ArgsJSON,
		EnvJson:         component.EnvJSON,
		PortsJson:       component.PortsJSON,
		MountsJson:      component.MountsJSON,
		NetworksJson:    component.NetworksJSON,
		DependsOnJson:   component.DependsOnJSON,
		HealthcheckJson: component.HealthcheckJSON,
		ResourcesJson:   component.ResourcesJSON,
		PullPolicy:      component.PullPolicy,
		CreatedAt:       transportresponse.FormatTime(component.CreatedAt),
		UpdatedAt:       transportresponse.FormatTime(component.UpdatedAt),
	}
}

func exposeResponse(expose model.Expose) pomeloorbit.ExposeResp {
	return pomeloorbit.ExposeResp{
		Id:            expose.Id,
		VersionId:     expose.VersionId,
		ComponentName: expose.ComponentName,
		Protocol:      expose.Protocol,
		ContainerPort: int32(expose.ContainerPort),
		PathPrefix:    expose.PathPrefix,
		CreatedAt:     transportresponse.FormatTime(expose.CreatedAt),
		UpdatedAt:     transportresponse.FormatTime(expose.UpdatedAt),
	}
}

func decodeDomainsJSON(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var domains []string
	if err := json.Unmarshal([]byte(raw), &domains); err != nil {
		return []string{}
	}
	return domains
}
