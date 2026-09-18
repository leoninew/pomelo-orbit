package taskhandler

import (
	"encoding/json"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	taskv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/task"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
	"google.golang.org/protobuf/types/known/structpb"
)

func taskCreateInput(req *taskv1.CreateTaskReq) tasksvc.CreateInput {
	return tasksvc.CreateInput{
		Id:          req.Id,
		TaskType:    req.TaskType,
		Payload:     rawPayload(req.Payload),
		PayloadJSON: req.PayloadJson,
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
		LockedAt:     transport.FormatOptionalTime(item.LockedAt),
		StartedAt:    transport.FormatOptionalTime(item.StartedAt),
		FinishedAt:   transport.FormatOptionalTime(item.FinishedAt),
		ErrorMessage: item.ErrorMessage,
		CreatedAt:    transport.FormatTime(item.CreatedAt),
		UpdatedAt:    transport.FormatTime(item.UpdatedAt),
	}
}

func rawPayload(payload *structpb.Value) json.RawMessage {
	if payload == nil {
		return nil
	}
	data, err := transport.MarshalProtoJSON(payload)
	if err != nil {
		return nil
	}
	return data
}
