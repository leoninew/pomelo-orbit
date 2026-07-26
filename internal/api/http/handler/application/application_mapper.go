package applicationhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func applicationCreateInput(projectID string, req *applicationv1.ApplicationCreateReq) applicationdto.ApplicationCreateInput {
	kind := ""
	if req.Kind != nil {
		kind = *req.Kind
	}
	return applicationdto.ApplicationCreateInput{
		ProjectId:       projectID,
		Name:            req.Name,
		Code:            req.Code,
		Kind:            kind,
		ImagePullPolicy: req.ImagePullPolicy,
	}
}

func applicationUpdateInput(req *applicationv1.ApplicationUpdateReq) applicationdto.ApplicationUpdateInput {
	return applicationdto.ApplicationUpdateInput{
		Name:            req.Name,
		Code:            req.Code,
		ImagePullPolicy: req.ImagePullPolicy,
	}
}

func applicationResponses(items []model.Application) []applicationv1.ApplicationResp {
	resp := make([]applicationv1.ApplicationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, applicationResponse(item))
	}
	return resp
}

func applicationResponse(item model.Application) applicationv1.ApplicationResp {
	return applicationv1.ApplicationResp{
		Id:              item.Id,
		ProjectId:       item.ProjectId,
		Name:            item.Name,
		Code:            item.Code,
		Kind:            item.Kind,
		ImagePullPolicy: item.ImagePullPolicy,
		CreatedAt:       transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:       transportresponse.FormatTime(item.UpdatedAt),
	}
}

func serviceResponse(item model.Service) servicev1.ServiceResp {
	return servicev1.ServiceResp{
		Id:                      item.Id,
		ApplicationId:           item.ApplicationId,
		InstanceKey:             item.InstanceKey,
		VersionId:               item.VersionId,
		LastSuccessfulVersionId: item.LastSuccessfulVersionId,
		Status:                  item.Status,
		CreatedAt:               transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:               transportresponse.FormatTime(item.UpdatedAt),
	}
}
