package cihandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func credentialResponse(item model.Credential) pomeloorbit.CredentialResp {
	return pomeloorbit.CredentialResp{
		Id:        item.Id,
		Name:      item.Name,
		Type:      item.Type,
		CreatedAt: transportresponse.FormatTime(item.CreatedAt),
	}
}

func credentialDetailResponse(item cidto.CredentialDetail) pomeloorbit.CredentialDetailResp {
	credential := credentialResponse(item.Credential)
	return pomeloorbit.CredentialDetailResp{
		Id:        credential.Id,
		Name:      credential.Name,
		Type:      credential.Type,
		Data:      item.Data,
		CreatedAt: credential.CreatedAt,
	}
}
