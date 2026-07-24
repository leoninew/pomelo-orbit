package environmenthandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	environmentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/environment/dto"
	environmentv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/environment"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func environmentCreateInput(req *environmentv1.EnvironmentCreateReq) environmentdto.EnvironmentCreateInput {
	return environmentdto.EnvironmentCreateInput{
		ProjectId:   req.ProjectId,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}
}

func environmentUpdateInput(req *environmentv1.EnvironmentUpdateReq) environmentdto.EnvironmentUpdateInput {
	return environmentdto.EnvironmentUpdateInput{
		Name:        req.Name,
		Description: req.Description,
	}
}

func environmentListResponses(items []model.Environment) []environmentv1.EnvironmentResp {
	resp := make([]environmentv1.EnvironmentResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, environmentMetaResponse(item))
	}
	return resp
}

func environmentResponse(view environmentdto.EnvironmentView) environmentv1.EnvironmentResp {
	return environmentMetaResponse(view.Environment)
}

func environmentMetaResponse(env model.Environment) environmentv1.EnvironmentResp {
	return environmentv1.EnvironmentResp{
		Id:          env.Id,
		ProjectId:   env.ProjectId,
		Code:        env.Code,
		Name:        env.Name,
		Description: env.Description,
		CreatedAt:   transportresponse.FormatTime(env.CreatedAt),
		UpdatedAt:   transportresponse.FormatTime(env.UpdatedAt),
	}
}
