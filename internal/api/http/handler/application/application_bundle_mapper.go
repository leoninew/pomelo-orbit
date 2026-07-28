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
		Name: req.Name, Image: req.Image,
		Command: append([]string(nil), req.Command...), Args: append([]string(nil), req.Args...),
		Env: componentEnvInput(req.Env), Ports: componentPortInput(req.Ports), Mounts: componentMountInput(req.Mounts),
		Networks: append([]string(nil), req.Networks...), Dependencies: componentDependencyInput(req.Dependencies),
		Healthcheck: componentHealthcheckInput(req.Healthcheck), Resources: componentResourcesInput(req.Resources),
		PullPolicy: req.PullPolicy, RestartPolicy: req.RestartPolicy, Tmpfs: componentTmpfsInput(req.Tmpfs), Ulimits: componentUlimitInput(req.Ulimits),
	}
}

func componentEnvInput(items []*applicationv1.ComponentEnv) []model.VersionComponentEnv {
	result := make([]model.VersionComponentEnv, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, model.VersionComponentEnv{Key: item.Key, Value: item.Value})
		}
	}
	return result
}

func componentPortInput(items []*applicationv1.ComponentPort) []model.VersionComponentPort {
	result := make([]model.VersionComponentPort, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, model.VersionComponentPort{HostPort: int(item.HostPort), ContainerPort: int(item.ContainerPort)})
		}
	}
	return result
}

func componentMountInput(items []*applicationv1.ComponentMount) []model.VersionComponentMount {
	result := make([]model.VersionComponentMount, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, model.VersionComponentMount{
				SourceType: item.SourceType, Source: item.Source, Target: item.Target, ReadOnly: item.ReadOnly,
				Content: stringValue(item.Content), ContentMode: stringValue(item.ContentMode),
			})
		}
	}
	return result
}

func componentDependencyInput(items []*applicationv1.ComponentDependency) []model.VersionComponentDependency {
	result := make([]model.VersionComponentDependency, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, model.VersionComponentDependency{Name: item.Name, Condition: item.Condition})
		}
	}
	return result
}

func componentHealthcheckInput(input *applicationv1.ComponentHealthcheck) *model.VersionComponentHealthcheck {
	if input == nil {
		return nil
	}
	return &model.VersionComponentHealthcheck{
		TestMode: input.TestMode, Test: append([]string(nil), input.Test...), Interval: input.Interval, Timeout: input.Timeout,
		Retries: intValue(input.Retries), StartPeriod: input.StartPeriod, StartInterval: input.StartInterval, Disabled: input.Disabled,
	}
}

func componentResourcesInput(input *applicationv1.ComponentResources) *model.VersionComponentResources {
	if input == nil {
		return nil
	}
	return &model.VersionComponentResources{
		LimitCPUs: input.LimitCpus, LimitMemory: input.LimitMemory,
		ReservationCPUs: input.ReservationCpus, ReservationMemory: input.ReservationMemory,
	}
}

func componentTmpfsInput(items []*applicationv1.ComponentTmpfs) []model.VersionComponentTmpfs {
	result := make([]model.VersionComponentTmpfs, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, model.VersionComponentTmpfs{Target: item.Target, SizeBytes: item.SizeBytes, Mode: item.Mode})
		}
	}
	return result
}

func componentUlimitInput(items []*applicationv1.ComponentUlimit) []model.VersionComponentUlimit {
	result := make([]model.VersionComponentUlimit, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, model.VersionComponentUlimit{Name: item.Name, Soft: item.Soft, Hard: item.Hard})
		}
	}
	return result
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

// versionUpdateInputFromJSON preserves an explicit empty expose collection.
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
		Id: component.Id, VersionId: component.VersionId, Name: component.Name, Image: component.Image,
		Command: append([]string(nil), component.Command...), Args: append([]string(nil), component.Args...),
		Env: componentEnvResponse(component.Env), Ports: componentPortResponse(component.Ports), Mounts: componentMountResponse(component.Mounts),
		Networks: append([]string(nil), component.Networks...), Dependencies: componentDependencyResponse(component.Dependencies),
		Healthcheck: componentHealthcheckResponse(component.Healthcheck), Resources: componentResourcesResponse(component.Resources),
		PullPolicy: component.PullPolicy, RestartPolicy: component.RestartPolicy, Tmpfs: componentTmpfsResponse(component.Tmpfs), Ulimits: componentUlimitResponse(component.Ulimits),
		CreatedAt: transportresponse.FormatTime(component.CreatedAt), UpdatedAt: transportresponse.FormatTime(component.UpdatedAt),
	}
}

func componentEnvResponse(items []model.VersionComponentEnv) []*applicationv1.ComponentEnv {
	result := make([]*applicationv1.ComponentEnv, 0, len(items))
	for _, item := range items {
		result = append(result, &applicationv1.ComponentEnv{Key: item.Key, Value: item.Value})
	}
	return result
}

func componentPortResponse(items []model.VersionComponentPort) []*applicationv1.ComponentPort {
	result := make([]*applicationv1.ComponentPort, 0, len(items))
	for _, item := range items {
		result = append(result, &applicationv1.ComponentPort{HostPort: int32(item.HostPort), ContainerPort: int32(item.ContainerPort)})
	}
	return result
}

func componentMountResponse(items []model.VersionComponentMount) []*applicationv1.ComponentMount {
	result := make([]*applicationv1.ComponentMount, 0, len(items))
	for _, item := range items {
		result = append(result, &applicationv1.ComponentMount{
			SourceType: item.SourceType, Source: item.Source, Target: item.Target, ReadOnly: item.ReadOnly,
			Content: optionalString(item.Content), ContentMode: optionalString(item.ContentMode),
		})
	}
	return result
}

func componentDependencyResponse(items []model.VersionComponentDependency) []*applicationv1.ComponentDependency {
	result := make([]*applicationv1.ComponentDependency, 0, len(items))
	for _, item := range items {
		result = append(result, &applicationv1.ComponentDependency{Name: item.Name, Condition: item.Condition})
	}
	return result
}

func componentHealthcheckResponse(input *model.VersionComponentHealthcheck) *applicationv1.ComponentHealthcheck {
	if input == nil {
		return nil
	}
	return &applicationv1.ComponentHealthcheck{
		TestMode: input.TestMode, Test: append([]string(nil), input.Test...), Interval: input.Interval, Timeout: input.Timeout,
		Retries: int32Value(input.Retries), StartPeriod: input.StartPeriod, StartInterval: input.StartInterval, Disabled: input.Disabled,
	}
}

func componentResourcesResponse(input *model.VersionComponentResources) *applicationv1.ComponentResources {
	if input == nil {
		return nil
	}
	return &applicationv1.ComponentResources{
		LimitCpus: input.LimitCPUs, LimitMemory: input.LimitMemory,
		ReservationCpus: input.ReservationCPUs, ReservationMemory: input.ReservationMemory,
	}
}

func componentTmpfsResponse(items []model.VersionComponentTmpfs) []*applicationv1.ComponentTmpfs {
	result := make([]*applicationv1.ComponentTmpfs, 0, len(items))
	for _, item := range items {
		result = append(result, &applicationv1.ComponentTmpfs{Target: item.Target, SizeBytes: item.SizeBytes, Mode: item.Mode})
	}
	return result
}

func componentUlimitResponse(items []model.VersionComponentUlimit) []*applicationv1.ComponentUlimit {
	result := make([]*applicationv1.ComponentUlimit, 0, len(items))
	for _, item := range items {
		result = append(result, &applicationv1.ComponentUlimit{Name: item.Name, Soft: item.Soft, Hard: item.Hard})
	}
	return result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func intValue(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func int32Value(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
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
