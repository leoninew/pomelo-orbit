package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func applicationCreateInput(projectID string, req *pomeloorbit.ApplicationCreateReq) cddto.ApplicationCreateInput {
	return cddto.ApplicationCreateInput{
		ProjectId:       projectID,
		Name:            req.Name,
		Code:            req.Code,
		ImagePullPolicy: req.ImagePullPolicy,
		RouteManaged:    req.RouteManaged,
	}
}

func applicationUpdateInput(req *pomeloorbit.ApplicationUpdateReq) cddto.ApplicationUpdateInput {
	return cddto.ApplicationUpdateInput{
		Name:            req.Name,
		Code:            req.Code,
		ImagePullPolicy: req.ImagePullPolicy,
		RouteManaged:    req.RouteManaged,
	}
}

func applicationDeleteInput(removeDir bool) cddto.ApplicationDeleteInput {
	return cddto.ApplicationDeleteInput{RemoveDir: removeDir}
}

func applicationDeployInput(req *pomeloorbit.ApplicationDeployReq) cddto.ApplicationDeployInput {
	return cddto.ApplicationDeployInput{ForceRecreate: req.ForceRecreate}
}

func applicationResponses(items []model.Application) []pomeloorbit.ApplicationResp {
	resp := make([]pomeloorbit.ApplicationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, applicationResponse(item))
	}
	return resp
}

func applicationResponse(item model.Application) pomeloorbit.ApplicationResp {
	return pomeloorbit.ApplicationResp{
		Id:              item.Id,
		ProjectId:       item.ProjectId,
		Name:            item.Name,
		Code:            item.Code,
		ImagePullPolicy: item.ImagePullPolicy,
		Status:          item.Status,
		RouteManaged:    item.RouteManaged,
		CreatedAt:       transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:       transportresponse.FormatTime(item.UpdatedAt),
	}
}
