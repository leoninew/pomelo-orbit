package delivery

import (
	"context"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerEnvironmentTools(server *mcp.Server) {
	addTool(server, "orbit_get_project_environment", "Read the unique local or SSH deployment Environment for one Project. SSH responses never include private keys or initialization credentials.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
	}) (map[string]any, error) {
		environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, input.ProjectId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"project_id": input.ProjectId, "environment": environmentOutput(environment)}, nil
	})

	addTool(server, "orbit_update_project_environment", "Update the explicit local or SSH target of a Project's unique deployment Environment. Local requires the nested local object; SSH requires the nested ssh object.", func(ctx context.Context, input struct {
		ProjectId  string  `json:"project_id" jsonschema:"required"`
		State      *string `json:"state,omitempty"`
		TargetType *string `json:"target_type,omitempty"`
		Local      *struct {
			WorkspaceRoot string `json:"workspace_root"`
		} `json:"local,omitempty"`
		SSH *struct {
			Platform      string `json:"platform"`
			Host          string `json:"host"`
			Port          int    `json:"port"`
			Username      string `json:"username"`
			WorkspaceRoot string `json:"workspace_root"`
		} `json:"ssh,omitempty"`
	}) (map[string]any, error) {
		if input.State == nil && input.TargetType == nil && input.Local == nil && input.SSH == nil {
			return nil, apperror.New(apperror.KindValidation, "at least one Environment field must be supplied")
		}
		var ssh *environmentdto.SSHTargetInput
		var local *environmentdto.LocalTargetInput
		if input.Local != nil {
			local = &environmentdto.LocalTargetInput{WorkspaceRoot: input.Local.WorkspaceRoot}
		}
		if input.SSH != nil {
			ssh = &environmentdto.SSHTargetInput{Platform: input.SSH.Platform, Host: input.SSH.Host, Port: input.SSH.Port, Username: input.SSH.Username, WorkspaceRoot: input.SSH.WorkspaceRoot}
		}
		environment, err := c.deps.Environment.UpdateForUser(ctx, c.deps.ActorUserId, input.ProjectId, environmentdto.UpdateInput{
			State: input.State, TargetType: input.TargetType, Local: local, SSH: ssh,
		})
		if err != nil {
			return nil, err
		}
		return writeResult("update_project_environment", map[string]string{"project_id": input.ProjectId, "environment_id": environment.Id}, "PUT", "/api/project/"+input.ProjectId+"/environment", map[string]any{"environment": environmentOutput(environment)}), nil
	})

	addTool(server, "orbit_probe_project_environment", "Probe Docker Compose prerequisites for a Project's active local or SSH Environment. SSH also verifies key authentication and host-key pinning.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
	}) (map[string]any, error) {
		environment, err := c.deps.Environment.ProbeForUser(ctx, c.deps.ActorUserId, input.ProjectId)
		if err != nil {
			return nil, err
		}
		return writeResult("probe_project_environment", map[string]string{"project_id": input.ProjectId, "environment_id": environment.Id}, "POST", "/api/project/"+input.ProjectId+"/environment/probe", map[string]any{"environment": environmentOutput(environment)}), nil
	})
}
