package deploymenthandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	deploymentv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/deployment"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func deploymentListInput(projectID string, applicationID string, status string, search string, dateFrom string, dateTo string, page int, perPage int) deploymentdto.DeploymentListInput {
	return deploymentdto.DeploymentListInput{
		ProjectId:     projectID,
		ApplicationId: applicationID,
		Status:        status,
		Search:        search,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		Page:          page,
		PerPage:       perPage,
	}
}

func deploymentResponses(items []model.Deployment) []deploymentv1.DeploymentResp {
	resp := make([]deploymentv1.DeploymentResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, deploymentResponse(item))
	}
	return resp
}

func deploymentResponse(item model.Deployment) deploymentv1.DeploymentResp {
	return deploymentv1.DeploymentResp{
		Id:                       item.Id,
		ProjectId:                item.ProjectId,
		ApplicationId:            item.ApplicationId,
		ApplicationName:          item.ApplicationName,
		OperationType:            item.OperationType,
		TriggerType:              item.TriggerType,
		CommandText:              item.CommandText,
		Status:                   item.Status,
		StartedAt:                transportresponse.FormatTime(item.StartedAt),
		FinishedAt:               transportresponse.FormatOptionalTime(item.FinishedAt),
		DurationMs:               transportresponse.OptionalInt32(item.DurationMs),
		LogText:                  item.LogText,
		ErrorMessage:             item.ErrorMessage,
		IsRollback:               item.IsRollback,
		RollbackFromDeploymentId: item.RollbackFromDeploymentId,
		VersionId:                item.VersionId,
		ServiceId:                item.ServiceId,
		OptionsJson:              item.OptionsJSON,
	}
}

func deploymentLogsResponse(log deploymentdto.DeploymentLog) deploymentv1.DeploymentLogsResp {
	return deploymentv1.DeploymentLogsResp{Logs: log.Logs, Offset: int32(log.Offset), IsComplete: log.IsComplete, Status: log.Status}
}

func deploymentContainerLogsResponse(log deploymentdto.DeploymentContainerLog) deploymentv1.DeploymentContainerLogsResp {
	return deploymentv1.DeploymentContainerLogsResp{Logs: log.Logs, Source: log.Source, IsRealtimeSupported: log.IsRealtimeSupported}
}

func applicationStatusResponse(containers []deploymentdto.RuntimeContainer) *applicationv1.ApplicationStatusResp {
	response := make([]*applicationv1.ApplicationContainerStatusResp, 0, len(containers))
	for _, container := range containers {
		response = append(response, &applicationv1.ApplicationContainerStatusResp{
			Id:      container.ID,
			Name:    container.Name,
			Service: container.Service,
			State:   container.State,
			Status:  container.Status,
			Health:  container.Health,
			Image:   container.Image,
		})
	}
	return &applicationv1.ApplicationStatusResp{Containers: response}
}
