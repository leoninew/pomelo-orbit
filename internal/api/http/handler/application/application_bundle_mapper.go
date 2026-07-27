package applicationhandler

import (
	"encoding/json"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func applicationImportInput(projectID string, req *applicationv1.ApplicationImportReq) applicationdto.ApplicationImportInput {
	components := make([]applicationdto.VersionComponentInput, 0, len(req.Components))
	for _, item := range req.Components {
		components = append(components, versionComponentInput(item))
	}
	exposes := make([]applicationdto.VersionExposeInput, 0, len(req.Exposes))
	for _, item := range req.Exposes {
		exposes = append(exposes, versionExposeInput(item))
	}
	kind := ""
	if req.Kind != nil {
		kind = *req.Kind
	}
	return applicationdto.ApplicationImportInput{
		ProjectId:       projectID,
		Name:            req.Name,
		Code:            req.Code,
		Kind:            kind,
		ImagePullPolicy: req.ImagePullPolicy,
		VersionLabel:    req.VersionLabel,
		VersionEnvJSON:  req.VersionEnvJson,
		VersionNote:     req.VersionNote,
		Components:      components,
		Exposes:         exposes,
	}
}

func applicationExportResponse(exported applicationdto.ApplicationExport) *applicationv1.ApplicationExportResp {
	resp := &applicationv1.ApplicationExportResp{
		Id:              exported.Application.Id,
		ProjectId:       exported.Application.ProjectId,
		Name:            exported.Application.Name,
		Code:            exported.Application.Code,
		Kind:            exported.Application.Kind,
		ImagePullPolicy: exported.Application.ImagePullPolicy,
		Versions:        make([]*applicationv1.VersionResp, 0, len(exported.Versions)),
		Services:        make([]*servicev1.ServiceResp, 0, len(exported.Services)),
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

func versionComponentInput(req *applicationv1.VersionComponentReq) applicationdto.VersionComponentInput {
	return applicationdto.VersionComponentInput{
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
		RestartPolicy:   req.RestartPolicy,
		TmpfsJSON:       req.TmpfsJson,
		UlimitsJSON:     req.UlimitsJson,
	}
}

func versionExposeInput(req *applicationv1.VersionExposeReq) applicationdto.VersionExposeInput {
	var listenPort *int
	if req.ListenPort != nil {
		v := int(*req.ListenPort)
		listenPort = &v
	}
	return applicationdto.VersionExposeInput{
		ComponentName: req.ComponentName,
		Protocol:      req.Protocol,
		ContainerPort: int(req.ContainerPort),
		PathPrefix:    req.PathPrefix,
		Access:        req.Access,
		ListenPort:    listenPort,
	}
}

func versionCreateInput(req *applicationv1.VersionCreateReq) applicationdto.VersionCreateInput {
	components := make([]applicationdto.VersionComponentInput, 0, len(req.Components))
	for _, item := range req.Components {
		components = append(components, versionComponentInput(item))
	}
	exposes := make([]applicationdto.VersionExposeInput, 0, len(req.Exposes))
	for _, item := range req.Exposes {
		exposes = append(exposes, versionExposeInput(item))
	}
	return applicationdto.VersionCreateInput{
		ApplicationId: req.ApplicationId,
		Label:         req.Label,
		EnvJSON:       req.EnvJson,
		Note:          req.Note,
		Components:    components,
		Exposes:       exposes,
	}
}

// versionUpdateInputFromJSON maps VersionUpdateReq and preserves empty repeated fields.
// protojson unmarshals "exposes":[] / "components":[] as nil, which would otherwise mean
// "leave unchanged"; raw JSON key presence is required to clear collections.
func versionUpdateInputFromJSON(data []byte) (applicationdto.VersionUpdateInput, error) {
	var req applicationv1.VersionUpdateReq
	if err := codec.UnmarshalProtoJSON(data, &req); err != nil {
		return applicationdto.VersionUpdateInput{}, err
	}
	var present map[string]json.RawMessage
	if err := json.Unmarshal(data, &present); err != nil {
		return applicationdto.VersionUpdateInput{}, err
	}
	input := applicationdto.VersionUpdateInput{
		Label:   req.Label,
		EnvJSON: req.EnvJson,
		Note:    req.Note,
	}
	if _, ok := present["components"]; ok {
		components := make([]applicationdto.VersionComponentInput, 0, len(req.Components))
		for _, item := range req.Components {
			if item == nil {
				continue
			}
			components = append(components, versionComponentInput(item))
		}
		input.Components = &components
	}
	if _, ok := present["exposes"]; ok {
		exposes := make([]applicationdto.VersionExposeInput, 0, len(req.Exposes))
		for _, item := range req.Exposes {
			if item == nil {
				continue
			}
			exposes = append(exposes, versionExposeInput(item))
		}
		input.Exposes = &exposes
	}
	return input, nil
}

func versionResponses(views []applicationdto.VersionView) []applicationv1.VersionResp {
	resp := make([]applicationv1.VersionResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, versionResponse(view))
	}
	return resp
}

func versionResponse(view applicationdto.VersionView) applicationv1.VersionResp {
	components := make([]*applicationv1.VersionComponentResp, 0, len(view.Components))
	for _, component := range view.Components {
		item := versionComponentResponse(component)
		components = append(components, &item)
	}
	exposes := make([]*applicationv1.VersionExposeResp, 0, len(view.Exposes))
	for _, expose := range view.Exposes {
		item := versionExposeResponse(expose)
		exposes = append(exposes, &item)
	}
	return applicationv1.VersionResp{
		Id:                   view.Version.Id,
		ApplicationId:        view.Version.ApplicationId,
		Label:                view.Version.Label,
		Status:               view.Version.Status,
		EnvJson:              view.Version.EnvJSON,
		CreatedFromVersionId: view.Version.CreatedFromVersionId,
		Note:                 view.Version.Note,
		ComponentSummary:     view.Version.ComponentSummary,
		CreatedAt:            transportresponse.FormatTime(view.Version.CreatedAt),
		UpdatedAt:            transportresponse.FormatTime(view.Version.UpdatedAt),
		Components:           components,
		Exposes:              exposes,
	}
}

func versionComponentResponse(component model.VersionComponent) applicationv1.VersionComponentResp {
	return applicationv1.VersionComponentResp{
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
		RestartPolicy:   component.RestartPolicy,
		TmpfsJson:       component.TmpfsJSON,
		UlimitsJson:     component.UlimitsJSON,
		CreatedAt:       transportresponse.FormatTime(component.CreatedAt),
		UpdatedAt:       transportresponse.FormatTime(component.UpdatedAt),
	}
}

func versionExposeResponse(expose model.VersionExpose) applicationv1.VersionExposeResp {
	var listenPort *int32
	if expose.ListenPort != nil {
		v := int32(*expose.ListenPort)
		listenPort = &v
	}
	return applicationv1.VersionExposeResp{
		Id:            expose.Id,
		VersionId:     expose.VersionId,
		ComponentName: expose.ComponentName,
		Protocol:      expose.Protocol,
		ContainerPort: int32(expose.ContainerPort),
		PathPrefix:    expose.PathPrefix,
		CreatedAt:     transportresponse.FormatTime(expose.CreatedAt),
		UpdatedAt:     transportresponse.FormatTime(expose.UpdatedAt),
		Access:        expose.Access,
		ListenPort:    listenPort,
	}
}
