package delivery

import (
	"context"
	"time"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerDeploymentTools(server *mcp.Server) {
	addTool(server, "orbit_deploy", "Create an Orbit deployment and immediately return its persisted command summary.", func(ctx context.Context, input struct {
		ServiceId     string `json:"service_id" jsonschema:"required"`
		ForceRecreate bool   `json:"force_recreate,omitempty"`
	}) (map[string]any, error) {
		result, err := c.deps.Deployment.DeployService(ctx, c.deps.ActorUserId, input.ServiceId, deploymentdto.DeployServiceInput{ForceRecreate: input.ForceRecreate})
		if err != nil {
			return nil, err
		}
		deployment, err := c.deps.Deployment.DeploymentForUser(ctx, c.deps.ActorUserId, result.DeploymentId)
		if err != nil {
			return nil, err
		}
		return writeResult("deploy_service", map[string]string{"service_id": input.ServiceId, "deployment_id": result.DeploymentId}, "POST", "/api/service/"+input.ServiceId+"/deploy", map[string]any{"deployment_id": result.DeploymentId, "warnings": result.Warnings, "command_text": deployment.CommandText, "deployment": deploymentOutput(deployment)}), nil
	})

	addTool(server, "orbit_stop", "Create an Orbit stop deployment, optionally requesting managed volume removal.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		ServiceId     string `json:"service_id" jsonschema:"required"`
		RemoveVolumes bool   `json:"remove_volumes,omitempty"`
	}) (map[string]any, error) {
		deploymentId, err := c.deps.Deployment.StopApplication(ctx, c.deps.ActorUserId, input.ApplicationId, deploymentdto.ServiceTargetInput{ServiceId: input.ServiceId, RemoveVolumes: input.RemoveVolumes})
		if err != nil {
			return nil, err
		}
		deployment, err := c.deps.Deployment.DeploymentForUser(ctx, c.deps.ActorUserId, deploymentId)
		if err != nil {
			return nil, err
		}
		return writeResult("stop_application", map[string]string{"application_id": input.ApplicationId, "service_id": input.ServiceId, "deployment_id": deploymentId}, "POST", "/api/application/"+input.ApplicationId+"/stop", map[string]any{"deployment_id": deploymentId, "command_text": deployment.CommandText, "deployment": deploymentOutput(deployment)}), nil
	})

	addTool(server, "orbit_restart", "Create an Orbit restart deployment and immediately return its command summary.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		ServiceId     string `json:"service_id" jsonschema:"required"`
	}) (map[string]any, error) {
		deploymentId, err := c.deps.Deployment.RestartApplication(ctx, c.deps.ActorUserId, input.ApplicationId, deploymentdto.ServiceTargetInput{ServiceId: input.ServiceId})
		if err != nil {
			return nil, err
		}
		deployment, err := c.deps.Deployment.DeploymentForUser(ctx, c.deps.ActorUserId, deploymentId)
		if err != nil {
			return nil, err
		}
		return writeResult("restart_application", map[string]string{"application_id": input.ApplicationId, "service_id": input.ServiceId, "deployment_id": deploymentId}, "POST", "/api/application/"+input.ApplicationId+"/restart", map[string]any{"deployment_id": deploymentId, "command_text": deployment.CommandText, "deployment": deploymentOutput(deployment)}), nil
	})

	addTool(server, "orbit_deployment_status", "Read the current Orbit Deployment state and command text.", func(ctx context.Context, input struct {
		DeploymentId string `json:"deployment_id" jsonschema:"required"`
	}) (map[string]any, error) {
		deployment, err := c.deps.Deployment.DeploymentForUser(ctx, c.deps.ActorUserId, input.DeploymentId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"deployment": deploymentOutput(deployment)}, nil
	})

	addTool(server, "orbit_deployment_logs", "Read Orbit worker logs for a Deployment from an incremental offset.", func(ctx context.Context, input struct {
		DeploymentId string `json:"deployment_id" jsonschema:"required"`
		Offset       int    `json:"offset,omitempty"`
	}) (map[string]any, error) {
		logs, err := c.deps.Deployment.DeploymentLog(ctx, c.deps.ActorUserId, input.DeploymentId, input.Offset)
		if err != nil {
			return nil, err
		}
		return map[string]any{"deployment_id": input.DeploymentId, "logs": deploymentLogsOutput(logs)}, nil
	})

	addTool(server, "orbit_wait_deployment", "Wait only until an Orbit Deployment reaches a terminal state or the configured timeout.", func(ctx context.Context, input struct {
		DeploymentId   string `json:"deployment_id" jsonschema:"required"`
		TimeoutSeconds *int   `json:"timeout_seconds,omitempty"`
	}) (map[string]any, error) {
		var timeout *time.Duration
		if input.TimeoutSeconds != nil {
			value := time.Duration(*input.TimeoutSeconds) * time.Second
			timeout = &value
		}
		result, err := c.deps.Deployment.WaitDeployment(ctx, c.deps.ActorUserId, input.DeploymentId, timeout)
		if err != nil {
			return nil, err
		}
		return map[string]any{"operation": "wait_deployment", "deployment_id": input.DeploymentId, "timed_out": result.TimedOut, "deployment": deploymentOutput(result.Deployment), "status": result.Deployment.Status}, nil
	})
}
