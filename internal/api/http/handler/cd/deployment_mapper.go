package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func deploymentListInput(projectID string, applicationID string, status string, search string, dateFrom string, dateTo string, page int, perPage int) cddto.DeploymentListInput {
	return cddto.DeploymentListInput{
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

func deploymentResponses(items []model.Deployment) []pomeloorbit.DeploymentResp {
	resp := make([]pomeloorbit.DeploymentResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, deploymentResponse(item))
	}
	return resp
}

func deploymentResponse(item model.Deployment) pomeloorbit.DeploymentResp {
	return pomeloorbit.DeploymentResp{
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
		EnvironmentId:            item.EnvironmentId,
	}
}

func deploymentLogsResponse(log cddto.DeploymentLog) pomeloorbit.DeploymentLogsResp {
	return pomeloorbit.DeploymentLogsResp{Logs: log.Logs, Offset: int32(log.Offset), IsComplete: log.IsComplete, Status: log.Status}
}

func deploymentContainerLogsResponse(log cddto.DeploymentContainerLog) pomeloorbit.DeploymentContainerLogsResp {
	return pomeloorbit.DeploymentContainerLogsResp{Logs: log.Logs, Source: log.Source, IsRealtimeSupported: log.IsRealtimeSupported}
}
