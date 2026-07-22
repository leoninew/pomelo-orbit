package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func applicationCreateInput(projectID string, req *pomeloorbit.ApplicationCreateReq) cddto.ApplicationCreateInput {
	kind := ""
	if req.Kind != nil {
		kind = *req.Kind
	}
	return cddto.ApplicationCreateInput{
		ProjectId:       projectID,
		Name:            req.Name,
		Code:            req.Code,
		Kind:            kind,
		ImagePullPolicy: req.ImagePullPolicy,
	}
}

func applicationUpdateInput(req *pomeloorbit.ApplicationUpdateReq) cddto.ApplicationUpdateInput {
	return cddto.ApplicationUpdateInput{
		Name:            req.Name,
		Code:            req.Code,
		ImagePullPolicy: req.ImagePullPolicy,
	}
}

func applicationDeleteInput(removeDir bool) cddto.ApplicationDeleteInput {
	return cddto.ApplicationDeleteInput{RemoveDir: removeDir}
}

func applicationDeployInput(req *pomeloorbit.ApplicationDeployReq) cddto.ApplicationDeployInput {
	return cddto.ApplicationDeployInput{
		VersionId:     req.VersionId,
		EnvironmentId: req.EnvironmentId,
		InstanceKey:   req.InstanceKey,
		ForceRecreate: req.ForceRecreate,
	}
}

func applicationServiceTargetInput(environmentId string, instanceKey string, serviceId string, removeVolumes bool) cddto.ApplicationServiceTargetInput {
	return cddto.ApplicationServiceTargetInput{
		EnvironmentId: environmentId,
		InstanceKey:   instanceKey,
		ServiceId:     serviceId,
		RemoveVolumes: removeVolumes,
	}
}

func applicationResponses(items []model.Application, services map[string][]model.Service) []pomeloorbit.ApplicationResp {
	resp := make([]pomeloorbit.ApplicationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, applicationResponse(item, services[item.Id]))
	}
	return resp
}

func applicationResponse(item model.Application, services []model.Service) pomeloorbit.ApplicationResp {
	serviceStatus := ""
	var serviceId *string
	var versionId *string
	if len(services) > 0 {
		primary := services[0]
		serviceStatus = primary.Status
		serviceId = &primary.Id
		versionId = &primary.VersionId
	}
	return pomeloorbit.ApplicationResp{
		Id:              item.Id,
		ProjectId:       item.ProjectId,
		Name:            item.Name,
		Code:            item.Code,
		Kind:            item.Kind,
		ImagePullPolicy: item.ImagePullPolicy,
		ServiceStatus:   serviceStatus,
		CreatedAt:       transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:       transportresponse.FormatTime(item.UpdatedAt),
		ServiceId:       serviceId,
		VersionId:       versionId,
		ServiceCount:    int32(len(services)),
	}
}

func serviceResponse(item model.Service) pomeloorbit.ServiceResp {
	return pomeloorbit.ServiceResp{
		Id:                      item.Id,
		ApplicationId:           item.ApplicationId,
		EnvironmentId:           item.EnvironmentId,
		InstanceKey:             item.InstanceKey,
		VersionId:               item.VersionId,
		LastSuccessfulVersionId: item.LastSuccessfulVersionId,
		Status:                  item.Status,
		CreatedAt:               transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:               transportresponse.FormatTime(item.UpdatedAt),
	}
}

func serviceResponses(items []model.Service) []pomeloorbit.ServiceResp {
	resp := make([]pomeloorbit.ServiceResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, serviceResponse(item))
	}
	return resp
}
