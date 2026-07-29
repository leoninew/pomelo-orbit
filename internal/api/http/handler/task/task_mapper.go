package taskhandler

import (
	"encoding/json"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	taskv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/task"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	"google.golang.org/protobuf/types/known/structpb"
)

func taskCreateInput(req *taskv1.CreateTaskReq) tasksvc.CreateInput {
	return tasksvc.CreateInput{
		Id:          req.Id,
		TaskType:    req.TaskType,
		Payload:     rawPayload(req.Payload),
		PayloadJSON: req.PayloadJson,
		MaxAttempts: int(req.MaxAttempts),
	}
}

func pipelineRunExecutePayload(runId string) map[string]string {
	return map[string]string{"pipeline_run_id": runId}
}

func applicationDeploymentPayload(applicationId string, deploymentId string) map[string]string {
	return map[string]string{"application_id": applicationId, "deployment_id": deploymentId}
}

func taskResponse(item *tasksvc.Task) taskv1.TaskResp {
	return taskv1.TaskResp{
		Id:           item.Id,
		TaskType:     item.TaskType,
		PayloadJson:  item.PayloadJSON,
		Status:       item.Status,
		Attempts:     int32(item.Attempts),
		MaxAttempts:  int32(item.MaxAttempts),
		LockedBy:     item.LockedBy,
		LockedAt:     transportresponse.FormatOptionalTime(item.LockedAt),
		StartedAt:    transportresponse.FormatOptionalTime(item.StartedAt),
		FinishedAt:   transportresponse.FormatOptionalTime(item.FinishedAt),
		ErrorMessage: item.ErrorMessage,
		CreatedAt:    transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:    transportresponse.FormatTime(item.UpdatedAt),
	}
}

func rawPayload(payload *structpb.Value) json.RawMessage {
	if payload == nil {
		return nil
	}
	data, err := transportcodec.MarshalProtoJSON(payload)
	if err != nil {
		return nil
	}
	return data
}
