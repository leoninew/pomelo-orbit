package servicehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func serviceViewResponse(item servicedto.ServiceView) servicev1.ServiceResp {
	exposes := make([]*servicev1.ServiceExposeResp, 0, len(item.Exposes))
	for _, expose := range item.Exposes {
		response := serviceExposeResponse(expose)
		exposes = append(exposes, &response)
	}
	return servicev1.ServiceResp{
		Id:              item.Service.Id,
		ApplicationId:   item.Service.ApplicationId,
		InstanceKey:     item.Service.InstanceKey,
		VersionId:       item.Service.VersionId,
		Status:          item.Service.Status,
		CreatedAt:       transportresponse.FormatTime(item.Service.CreatedAt),
		UpdatedAt:       transportresponse.FormatTime(item.Service.UpdatedAt),
		ApplicationName: item.ApplicationName,
		ApplicationCode: item.ApplicationCode,
		ApplicationKind: item.ApplicationKind,
		VersionLabel:    item.VersionLabel,
		Exposes:         exposes,
		RuntimeConfig:   item.Service.RuntimeConfig,
	}
}

func serviceViewResponses(items []servicedto.ServiceView) []servicev1.ServiceResp {
	resp := make([]servicev1.ServiceResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, serviceViewResponse(item))
	}
	return resp
}

func serviceExposeInput(item *servicev1.ServiceExposeReq) servicedto.ServiceExposeInput {
	if item == nil {
		return servicedto.ServiceExposeInput{}
	}
	var listenPort *int
	if item.ListenPort != nil {
		value := int(*item.ListenPort)
		listenPort = &value
	}
	return servicedto.ServiceExposeInput{
		ComponentName: item.ComponentName, Protocol: item.Protocol, ContainerPort: int(item.ContainerPort),
		PathPrefix: item.PathPrefix, Access: item.Access, ListenPort: listenPort,
	}
}

func serviceExposeInputs(items []*servicev1.ServiceExposeReq) []servicedto.ServiceExposeInput {
	result := make([]servicedto.ServiceExposeInput, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, serviceExposeInput(item))
		}
	}
	return result
}

func serviceExposeResponse(item model.ServiceExpose) servicev1.ServiceExposeResp {
	var listenPort *int32
	if item.ListenPort != nil {
		value := int32(*item.ListenPort)
		listenPort = &value
	}
	return servicev1.ServiceExposeResp{
		Id: item.Id, ServiceId: item.ServiceId, ComponentName: item.ComponentName, Protocol: item.Protocol,
		ContainerPort: int32(item.ContainerPort), PathPrefix: item.PathPrefix, Access: item.Access, ListenPort: listenPort,
		CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt),
	}
}
