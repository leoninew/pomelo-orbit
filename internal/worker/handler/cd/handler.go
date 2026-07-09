package cd

import (
	"context"
	"encoding/json"
	"fmt"

	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

type Payload struct {
	ApplicationId string `json:"application_id"`
	DeploymentId  string `json:"deployment_id"`
	RemoveVolumes bool   `json:"remove_volumes"`
	ForceRecreate bool   `json:"force_recreate"`
}

type ApplicationDeployer interface {
	ExecuteApplicationDeploy(ctx context.Context, applicationId string, deploymentId string, forceRecreate bool) error
	ExecuteApplicationRestart(ctx context.Context, applicationId string, deploymentId string) error
	ExecuteApplicationStop(ctx context.Context, applicationId string, deploymentId string, removeVolumes bool) error
}

type Handler struct {
	deployer  ApplicationDeployer
	operation string
}

func NewDeployHandler(deployer ApplicationDeployer) Handler {
	return Handler{deployer: deployer, operation: "deploy"}
}

func NewRestartHandler(deployer ApplicationDeployer) Handler {
	return Handler{deployer: deployer, operation: "restart"}
}

func NewStopHandler(deployer ApplicationDeployer) Handler {
	return Handler{deployer: deployer, operation: "stop"}
}

func (h Handler) Handle(ctx context.Context, item tasksvc.Task) error {
	var payload Payload
	if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("parse cd task payload: %w", err)
	}
	if payload.ApplicationId == "" || payload.DeploymentId == "" {
		return fmt.Errorf("application_id and deployment_id are required")
	}
	switch h.operation {
	case "deploy":
		return h.deployer.ExecuteApplicationDeploy(ctx, payload.ApplicationId, payload.DeploymentId, payload.ForceRecreate)
	case "restart":
		return h.deployer.ExecuteApplicationRestart(ctx, payload.ApplicationId, payload.DeploymentId)
	case "stop":
		return h.deployer.ExecuteApplicationStop(ctx, payload.ApplicationId, payload.DeploymentId, payload.RemoveVolumes)
	default:
		return fmt.Errorf("unsupported cd task operation: %s", h.operation)
	}
}
