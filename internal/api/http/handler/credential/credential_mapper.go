package credentialhandler

import (
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	credentialv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/credential"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func credentialResponse(item model.Credential) credentialv1.CredentialResp {
	return credentialv1.CredentialResp{
		Id:        item.Id,
		Name:      item.Name,
		Type:      item.Type,
		CreatedAt: transportresponse.FormatTime(item.CreatedAt),
	}
}

func credentialDetailResponse(item credentialdto.CredentialDetail) credentialv1.CredentialDetailResp {
	credential := credentialResponse(item.Credential)
	return credentialv1.CredentialDetailResp{
		Id:        credential.Id,
		Name:      credential.Name,
		Type:      credential.Type,
		Data:      item.Data,
		CreatedAt: credential.CreatedAt,
	}
}
