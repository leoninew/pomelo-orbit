package servicehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
)

func serviceViewResponse(item servicedto.ServiceView) servicev1.ServiceResp {
	return servicev1.ServiceResp{
		Id:                         item.Service.Id,
		ApplicationId:              item.Service.ApplicationId,
		InstanceKey:                item.Service.InstanceKey,
		VersionId:                  item.Service.VersionId,
		LastSuccessfulVersionId:    item.Service.LastSuccessfulVersionId,
		Status:                     item.Service.Status,
		CreatedAt:                  transportresponse.FormatTime(item.Service.CreatedAt),
		UpdatedAt:                  transportresponse.FormatTime(item.Service.UpdatedAt),
		ApplicationName:            item.ApplicationName,
		ApplicationCode:            item.ApplicationCode,
		ApplicationKind:            item.ApplicationKind,
		VersionLabel:               item.VersionLabel,
		LastSuccessfulVersionLabel: item.LastSuccessfulVersionLabel,
	}
}

func serviceViewResponses(items []servicedto.ServiceView) []servicev1.ServiceResp {
	resp := make([]servicev1.ServiceResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, serviceViewResponse(item))
	}
	return resp
}

func serviceRuntimeConfigResponse(item servicedto.RuntimeConfigView) servicev1.ServiceRuntimeConfigResp {
	return servicev1.ServiceRuntimeConfigResp{ServiceId: item.ServiceId, VersionId: item.VersionId, RuntimeConfig: item.RuntimeConfig}
}
