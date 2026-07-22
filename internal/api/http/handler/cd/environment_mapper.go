package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func environmentCreateInput(req *pomeloorbit.EnvironmentCreateReq) cddto.EnvironmentCreateInput {
	return cddto.EnvironmentCreateInput{
		ProjectId:         req.ProjectId,
		Code:              req.Code,
		Name:              req.Name,
		Description:       req.Description,
		DefaultEntrypoint: req.DefaultEntrypoint,
		TCPEntrypoint:     req.TcpEntrypoint,
		TLSMode:           req.TlsMode,
	}
}

func environmentUpdateInput(req *pomeloorbit.EnvironmentUpdateReq) cddto.EnvironmentUpdateInput {
	return cddto.EnvironmentUpdateInput{
		Name:              req.Name,
		Description:       req.Description,
		DefaultEntrypoint: req.DefaultEntrypoint,
		TCPEntrypoint:     req.TcpEntrypoint,
		TLSMode:           req.TlsMode,
	}
}

func environmentListResponses(items []model.Environment) []pomeloorbit.EnvironmentResp {
	resp := make([]pomeloorbit.EnvironmentResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, environmentMetaResponse(item))
	}
	return resp
}

func environmentResponse(view cddto.EnvironmentView) pomeloorbit.EnvironmentResp {
	return environmentMetaResponse(view.Environment)
}

func environmentMetaResponse(env model.Environment) pomeloorbit.EnvironmentResp {
	return pomeloorbit.EnvironmentResp{
		Id:                env.Id,
		ProjectId:         env.ProjectId,
		Code:              env.Code,
		Name:              env.Name,
		Description:       env.Description,
		CreatedAt:         transportresponse.FormatTime(env.CreatedAt),
		UpdatedAt:         transportresponse.FormatTime(env.UpdatedAt),
		DefaultEntrypoint: env.DefaultEntrypoint,
		TcpEntrypoint:     env.TCPEntrypoint,
		TlsMode:           env.TLSMode,
	}
}
