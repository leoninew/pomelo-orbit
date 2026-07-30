package applicationhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/commandline"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func versionComponentInput(req *applicationv1.VersionComponentReq) applicationdto.VersionComponentInput {
	return applicationdto.VersionComponentInput{
		Name: req.Name, Image: req.Image,
		Command: req.Command,
		Env:     componentEnvInput(req.Env), Ports: componentPortInput(req.Ports), Mounts: componentMountInput(req.Mounts),
		Dependencies: componentDependencyInput(req.Dependencies),
		Healthcheck:  componentHealthcheckInput(req.Healthcheck), Resources: componentResourcesInput(req.Resources),
		PullPolicy: req.PullPolicy, RestartPolicy: req.RestartPolicy, Tmpfs: componentTmpfsInput(req.Tmpfs), Ulimits: componentUlimitInput(req.Ulimits),
	}
}

func versionComponentBasicUpdateInput(req *applicationv1.VersionComponentBasicUpdateReq) applicationdto.VersionComponentBasicUpdateInput {
	return applicationdto.VersionComponentBasicUpdateInput{
		Name: req.Name, Image: req.Image, Command: req.Command, PullPolicy: req.PullPolicy, RestartPolicy: req.RestartPolicy,
	}
}

func versionComponentCreateInput(req *applicationv1.VersionComponentCreateReq) applicationdto.VersionComponentInput {
	return applicationdto.VersionComponentInput{
		Name: req.Name, Image: req.Image, Command: req.Command, PullPolicy: req.PullPolicy, RestartPolicy: req.RestartPolicy,
	}
}

func versionComponentRuntimeUpdateInput(req *applicationv1.VersionComponentRuntimeUpdateReq) applicationdto.VersionComponentRuntimeUpdateInput {
	return applicationdto.VersionComponentRuntimeUpdateInput{
		Healthcheck: componentHealthcheckInput(req.Healthcheck),
	}
}

func versionComponentPortsUpdateInput(req *applicationv1.VersionComponentPortsUpdateReq) applicationdto.VersionComponentPortsUpdateInput {
	return applicationdto.VersionComponentPortsUpdateInput{Ports: componentPortInput(req.Ports)}
}

func versionComponentEnvUpdateInput(req *applicationv1.VersionComponentEnvUpdateReq) applicationdto.VersionComponentEnvUpdateInput {
	return applicationdto.VersionComponentEnvUpdateInput{Env: componentEnvInput(req.Env)}
}

func versionComponentMountsUpdateInput(req *applicationv1.VersionComponentMountsUpdateReq) applicationdto.VersionComponentMountsUpdateInput {
	return applicationdto.VersionComponentMountsUpdateInput{Mounts: componentMountInput(req.Mounts)}
}

func versionComponentDependenciesUpdateInput(req *applicationv1.VersionComponentDependenciesUpdateReq) applicationdto.VersionComponentDependenciesUpdateInput {
	return applicationdto.VersionComponentDependenciesUpdateInput{Dependencies: componentDependencyInput(req.Dependencies)}
}

func versionComponentAdvancedUpdateInput(req *applicationv1.VersionComponentAdvancedUpdateReq) applicationdto.VersionComponentAdvancedUpdateInput {
	return applicationdto.VersionComponentAdvancedUpdateInput{
		Resources: componentResourcesInput(req.Resources), Tmpfs: componentTmpfsInput(req.Tmpfs), Ulimits: componentUlimitInput(req.Ulimits),
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
				SourceIsHostPath: item.SourceIsHostPath, Content: stringValue(item.Content),
				Mode: item.Mode, IgnoreIfExists: item.IgnoreIfExists,
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

func componentHealthcheckInput(input *applicationv1.ComponentHealthcheck) *applicationdto.VersionComponentHealthcheckInput {
	if input == nil {
		return nil
	}
	return &applicationdto.VersionComponentHealthcheckInput{
		TestMode: input.TestMode, Test: input.Test, Interval: input.Interval, Timeout: input.Timeout,
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

func versionCreateInput(req *applicationv1.VersionCreateReq) applicationdto.VersionCreateInput {
	components := make([]applicationdto.VersionComponentInput, 0, len(req.Components))
	for _, item := range req.Components {
		components = append(components, versionComponentInput(item))
	}
	return applicationdto.VersionCreateInput{
		ApplicationId: req.ApplicationId,
		Label:         req.Label,
		Note:          req.Note,
		Components:    components,
	}
}

func versionUpdateInput(req *applicationv1.VersionUpdateReq) applicationdto.VersionUpdateInput {
	return applicationdto.VersionUpdateInput{Label: req.Label, Note: req.Note}
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
	return applicationv1.VersionResp{
		Id:                   view.Version.Id,
		ApplicationId:        view.Version.ApplicationId,
		Label:                view.Version.Label,
		Status:               view.Version.Status,
		CreatedFromVersionId: view.Version.CreatedFromVersionId,
		Note:                 view.Version.Note,
		ComponentSummary:     view.Version.ComponentSummary,
		CreatedAt:            transportresponse.FormatTime(view.Version.CreatedAt),
		UpdatedAt:            transportresponse.FormatTime(view.Version.UpdatedAt),
		Components:           components,
	}
}

func versionComponentResponse(component model.VersionComponent) applicationv1.VersionComponentResp {
	return applicationv1.VersionComponentResp{
		Id: component.Id, VersionId: component.VersionId, Name: component.Name, Image: component.Image,
		Command: commandline.Format(component.Command),
		Env:     componentEnvResponse(component.Env), Ports: componentPortResponse(component.Ports), Mounts: componentMountResponse(component.Mounts),
		Dependencies: componentDependencyResponse(component.Dependencies),
		Healthcheck:  componentHealthcheckResponse(component.Healthcheck), Resources: componentResourcesResponse(component.Resources),
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
			SourceIsHostPath: item.SourceIsHostPath, Content: optionalString(item.Content),
			Mode: item.Mode, IgnoreIfExists: item.IgnoreIfExists,
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
		TestMode: input.TestMode, Test: input.Test, Interval: input.Interval, Timeout: input.Timeout,
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
