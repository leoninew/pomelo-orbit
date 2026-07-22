package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func environmentCreateInput(req *pomeloorbit.EnvironmentCreateReq) cddto.EnvironmentCreateInput {
	return cddto.EnvironmentCreateInput{
		ProjectId:   req.ProjectId,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}
}

func environmentUpdateInput(req *pomeloorbit.EnvironmentUpdateReq) cddto.EnvironmentUpdateInput {
	return cddto.EnvironmentUpdateInput{
		Name:        req.Name,
		Description: req.Description,
	}
}

func environmentBindingInputs(items []*pomeloorbit.EnvironmentBindingReq) []cddto.EnvironmentBindingInput {
	out := make([]cddto.EnvironmentBindingInput, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, cddto.EnvironmentBindingInput{
			ComponentName: item.ComponentName,
			Protocol:      item.Protocol,
			ContainerPort: int(item.ContainerPort),
			Domains:       item.Domains,
			Entrypoint:    item.Entrypoint,
			TLSMode:       item.TlsMode,
			SNIHost:       item.SniHost,
			Note:          item.Note,
		})
	}
	return out
}

func environmentListResponses(items []model.Environment) []pomeloorbit.EnvironmentResp {
	resp := make([]pomeloorbit.EnvironmentResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, environmentMetaResponse(item, nil))
	}
	return resp
}

func environmentResponse(view cddto.EnvironmentView) pomeloorbit.EnvironmentResp {
	return environmentMetaResponse(view.Environment, view.Bindings)
}

func environmentMetaResponse(env model.Environment, bindings []model.EnvironmentBinding) pomeloorbit.EnvironmentResp {
	items := make([]*pomeloorbit.EnvironmentBindingResp, 0, len(bindings))
	for _, binding := range bindings {
		item := environmentBindingResponse(binding)
		items = append(items, &item)
	}
	return pomeloorbit.EnvironmentResp{
		Id:          env.Id,
		ProjectId:   env.ProjectId,
		Code:        env.Code,
		Name:        env.Name,
		Description: env.Description,
		CreatedAt:   transportresponse.FormatTime(env.CreatedAt),
		UpdatedAt:   transportresponse.FormatTime(env.UpdatedAt),
		Bindings:    items,
	}
}

func environmentBindingResponse(binding model.EnvironmentBinding) pomeloorbit.EnvironmentBindingResp {
	return pomeloorbit.EnvironmentBindingResp{
		Id:            binding.Id,
		EnvironmentId: binding.EnvironmentId,
		ComponentName: binding.ComponentName,
		Protocol:      binding.Protocol,
		ContainerPort: int32(binding.ContainerPort),
		Domains:       decodeDomainsJSON(binding.DomainsJSON),
		Entrypoint:    binding.Entrypoint,
		TlsMode:       binding.TLSMode,
		SniHost:       binding.SNIHost,
		Note:          binding.Note,
		CreatedAt:     transportresponse.FormatTime(binding.CreatedAt),
		UpdatedAt:     transportresponse.FormatTime(binding.UpdatedAt),
	}
}
