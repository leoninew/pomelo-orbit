package servicehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/commandline"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func serviceViewResponse(item servicedto.ServiceView) servicev1.ServiceResp {
	componentImages := make(map[string]string, len(item.ComponentDefinitions))
	for _, definition := range item.ComponentDefinitions {
		componentImages[definition.Id] = definition.Image
	}
	effectiveEndpoints := make(map[string][]model.VersionComponentEndpoint, len(item.EffectiveComponents))
	for _, component := range item.EffectiveComponents {
		effectiveEndpoints[component.ServiceComponentId] = component.Endpoints
	}
	components := make([]*servicev1.ServiceComponentResp, 0, len(item.Components))
	for _, component := range item.Components {
		response := serviceComponentResponse(component)
		response.Image = componentImages[component.SourceVersionComponentId]
		response.ContainerName = model.RuntimeContainerName(item.ApplicationCode, component.ComponentName)
		response.EffectiveEndpoints = serviceComponentDeclaredEndpointsResponse(effectiveEndpoints[component.Id])
		components = append(components, &response)
	}
	env := make([]*servicev1.ServiceEnv, 0, len(item.Env))
	for _, value := range item.Env {
		env = append(env, &servicev1.ServiceEnv{Key: value.Key, Value: value.Value})
	}
	return servicev1.ServiceResp{Id: item.Service.Id, ApplicationId: item.Service.ApplicationId, InstanceKey: item.Service.InstanceKey, VersionId: item.Service.VersionId, Status: item.Service.Status, CreatedAt: transportresponse.FormatTime(item.Service.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.Service.UpdatedAt), ApplicationName: item.ApplicationName, ApplicationCode: item.ApplicationCode, ApplicationKind: item.ApplicationKind, VersionLabel: item.VersionLabel, Components: components, PendingDeploy: item.PendingDeploy, EffectivePlanHash: item.EffectivePlanHash, Env: env, EffectiveError: item.EffectiveError, ActiveDeployment: item.ActiveDeployment, Code: item.Service.Code}
}

func serviceViewResponses(items []servicedto.ServiceView) []servicev1.ServiceResp {
	result := make([]servicev1.ServiceResp, 0, len(items))
	for _, item := range items {
		result = append(result, serviceViewResponse(item))
	}
	return result
}

func serviceComponentResponse(item model.ServiceComponent) servicev1.ServiceComponentResp {
	env := make([]*servicev1.ServiceComponentEnvOverlay, 0, len(item.Env))
	for _, value := range item.Env {
		env = append(env, &servicev1.ServiceComponentEnvOverlay{Key: value.Key, Value: value.Value, State: string(value.State)})
	}
	mounts := make([]*servicev1.ServiceComponentMountOverlay, 0, len(item.Mounts))
	for _, value := range item.Mounts {
		mounts = append(mounts, &servicev1.ServiceComponentMountOverlay{Target: value.Target, Source: value.Source, SourceIsHostPath: value.SourceIsHostPath, State: string(value.State)})
	}
	endpoints := make([]*servicev1.ServiceComponentEndpointOverlay, 0, len(item.Endpoints))
	for _, value := range item.Endpoints {
		endpoints = append(endpoints, &servicev1.ServiceComponentEndpointOverlay{Name: value.Name, Mode: value.Mode, BindAddress: value.BindAddress, ListenPort: int32Ptr(value.ListenPort), Entrypoint: value.Entrypoint, PathPrefix: value.PathPrefix, State: string(value.State)})
	}
	return servicev1.ServiceComponentResp{Id: item.Id, ServiceId: item.ServiceId, SourceVersionComponentId: item.SourceVersionComponentId, ComponentName: item.ComponentName, Status: item.Status, Env: env, Mounts: mounts, Resources: serviceComponentResourcesResponse(item.Resources), Endpoints: endpoints, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt), Entrypoint: optionalCommandText(item.Entrypoint), Command: optionalCommandText(item.Command), PullPolicy: item.PullPolicy, RestartPolicy: item.RestartPolicy}
}

func serviceComponentDetailResponse(item servicedto.ServiceComponentDetail) servicev1.ServiceComponentDetailResp {
	serviceComponent := serviceComponentResponse(item.ServiceComponent)
	versionComponent := componentDeclarationResponse(item.VersionComponent)
	return servicev1.ServiceComponentDetailResp{ServiceComponent: &serviceComponent, VersionComponent: &versionComponent}
}

func componentDeclarationResponse(component model.VersionComponent) servicev1.ServiceComponentDefinitionResp {
	env := make([]*servicev1.ServiceComponentDeclaredEnv, 0, len(component.Env))
	for _, item := range component.Env {
		env = append(env, &servicev1.ServiceComponentDeclaredEnv{Key: item.Key, Value: item.Value})
	}
	mounts := make([]*servicev1.ServiceComponentDeclaredMount, 0, len(component.Mounts))
	for _, item := range component.Mounts {
		mounts = append(mounts, &servicev1.ServiceComponentDeclaredMount{SourceType: item.SourceType, Source: item.Source, Target: item.Target, ReadOnly: item.ReadOnly, SourceIsHostPath: item.SourceIsHostPath})
	}
	return servicev1.ServiceComponentDefinitionResp{Id: component.Id, Name: component.Name, Image: component.Image, Entrypoint: commandline.Format(component.Entrypoint), Command: commandline.Format(component.Command), PullPolicy: component.PullPolicy, RestartPolicy: component.RestartPolicy, Env: env, Endpoints: serviceComponentDeclaredEndpointsResponse(component.Endpoints), Mounts: mounts, Resources: componentResourcesResponse(component.Resources)}
}

func serviceComponentDeclaredEndpointsResponse(items []model.VersionComponentEndpoint) []*servicev1.ServiceComponentDeclaredEndpoint {
	endpoints := make([]*servicev1.ServiceComponentDeclaredEndpoint, 0, len(items))
	for _, item := range items {
		endpoints = append(endpoints, &servicev1.ServiceComponentDeclaredEndpoint{Name: item.Name, Protocol: item.Protocol, ContainerPort: int32(item.ContainerPort), Mode: item.Mode, BindAddress: item.BindAddress, ListenPort: int32Ptr(item.ListenPort), Entrypoint: item.Entrypoint, PathPrefix: item.PathPrefix})
	}
	return endpoints
}

func componentResourcesResponse(value *model.VersionComponentResources) *servicev1.ServiceComponentDeclaredResources {
	if value == nil {
		return nil
	}
	return &servicev1.ServiceComponentDeclaredResources{LimitCpus: value.LimitCPUs, LimitMemory: value.LimitMemory, ReservationCpus: value.ReservationCPUs, ReservationMemory: value.ReservationMemory}
}

func serviceComponentOverlayInput(req *servicev1.ServiceComponentOverlayUpdateReq) servicedto.ServiceComponentOverlayInput {
	if req == nil {
		return servicedto.ServiceComponentOverlayInput{}
	}
	result := servicedto.ServiceComponentOverlayInput{Entrypoint: req.Entrypoint, Command: req.Command, PullPolicy: req.PullPolicy, RestartPolicy: req.RestartPolicy, Resources: serviceComponentResourcesInput(req.Resources)}
	for _, item := range req.Env {
		if item != nil {
			result.Env = append(result.Env, model.ServiceComponentEnv{Key: item.Key, Value: item.Value, State: model.ServiceComponentOverlayState(item.State)})
		}
	}
	for _, item := range req.Mounts {
		if item != nil {
			result.Mounts = append(result.Mounts, model.ServiceComponentMount{Target: item.Target, Source: item.Source, SourceIsHostPath: item.SourceIsHostPath, State: model.ServiceComponentOverlayState(item.State)})
		}
	}
	for _, item := range req.Endpoints {
		if item != nil {
			result.Endpoints = append(result.Endpoints, model.ServiceComponentEndpoint{Name: item.Name, Mode: item.Mode, BindAddress: item.BindAddress, ListenPort: intPtr(item.ListenPort), Entrypoint: item.Entrypoint, PathPrefix: item.PathPrefix, State: model.ServiceComponentOverlayState(item.State)})
		}
	}
	return result
}

func optionalCommandText(value []string) *string {
	if value == nil {
		return nil
	}
	text := commandline.Format(value)
	return &text
}

func serviceEnvUpdateInput(req *servicev1.ServiceEnvUpdateReq) servicedto.ServiceEnvUpdateInput {
	if req == nil {
		return servicedto.ServiceEnvUpdateInput{}
	}
	result := servicedto.ServiceEnvUpdateInput{Env: make([]model.ServiceEnv, 0, len(req.Env))}
	for _, item := range req.Env {
		if item != nil {
			result.Env = append(result.Env, model.ServiceEnv{Key: item.Key, Value: item.Value})
		}
	}
	return result
}

func serviceComponentResourcesResponse(value *model.ServiceComponentResources) *servicev1.ServiceComponentResourceOverlay {
	if value == nil {
		return nil
	}
	return &servicev1.ServiceComponentResourceOverlay{LimitCpus: value.LimitCPUs, LimitMemory: value.LimitMemory, ReservationCpus: value.ReservationCPUs, ReservationMemory: value.ReservationMemory, State: string(value.State)}
}
func serviceComponentResourcesInput(value *servicev1.ServiceComponentResourceOverlay) *model.ServiceComponentResources {
	if value == nil {
		return nil
	}
	return &model.ServiceComponentResources{LimitCPUs: value.LimitCpus, LimitMemory: value.LimitMemory, ReservationCPUs: value.ReservationCpus, ReservationMemory: value.ReservationMemory, State: model.ServiceComponentOverlayState(value.State)}
}
func intPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	result := int(*value)
	return &result
}
func int32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	result := int32(*value)
	return &result
}
