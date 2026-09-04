package delivery

import (
	"context"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerEnvironmentTools(server *mcp.Server) {
	addTool(server, "orbit_get_project_environment", "Read the unique SSH deployment Environment for one Project. The response never includes the deployment SSH private key.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
	}) (map[string]any, error) {
		environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, input.ProjectId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"project_id": input.ProjectId, "environment": environmentOutput(environment)}, nil
	})

	addTool(server, "orbit_update_project_environment", "Update configured fields of a Project's unique SSH deployment Environment. Use only project_id; the server keeps the Environment scoped to that Project. deployment_ssh_private_key and deployment_ssh_key_passphrase are write-only and never returned.", func(ctx context.Context, input struct {
		ProjectId                  string  `json:"project_id" jsonschema:"required"`
		State                      *string `json:"state,omitempty"`
		Platform                   *string `json:"platform,omitempty"`
		Host                       *string `json:"host,omitempty"`
		Port                       *int    `json:"port,omitempty"`
		Username                   *string `json:"username,omitempty"`
		WorkspaceRoot              *string `json:"workspace_root,omitempty"`
		DeploymentSSHPrivateKey    *string `json:"deployment_ssh_private_key,omitempty"`
		DeploymentSSHKeyPassphrase *string `json:"deployment_ssh_key_passphrase,omitempty"`
		HostKeyFingerprint         *string `json:"host_key_fingerprint,omitempty"`
	}) (map[string]any, error) {
		if input.State == nil && input.Platform == nil && input.Host == nil && input.Port == nil && input.Username == nil && input.WorkspaceRoot == nil && input.DeploymentSSHPrivateKey == nil && input.DeploymentSSHKeyPassphrase == nil && input.HostKeyFingerprint == nil {
			return nil, apperror.New(apperror.KindValidation, "at least one Environment field must be supplied")
		}
		environment, err := c.deps.Environment.UpdateForUser(ctx, c.deps.ActorUserId, input.ProjectId, environmentdto.UpdateInput{
			State:                      input.State,
			Platform:                   input.Platform,
			Host:                       input.Host,
			Port:                       input.Port,
			Username:                   input.Username,
			WorkspaceRoot:              input.WorkspaceRoot,
			DeploymentSSHPrivateKey:    input.DeploymentSSHPrivateKey,
			DeploymentSSHKeyPassphrase: input.DeploymentSSHKeyPassphrase,
			HostKeyFingerprint:         input.HostKeyFingerprint,
		})
		if err != nil {
			return nil, err
		}
		return writeResult("update_project_environment", map[string]string{"project_id": input.ProjectId, "environment_id": environment.Id}, "PUT", "/api/project/"+input.ProjectId+"/environment", map[string]any{"environment": environmentOutput(environment)}), nil
	})

	addTool(server, "orbit_probe_project_environment", "Probe SSH, host-key authentication, Docker Compose, and platform prerequisites for a Project's active Environment.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
	}) (map[string]any, error) {
		environment, err := c.deps.Environment.ProbeForUser(ctx, c.deps.ActorUserId, input.ProjectId)
		if err != nil {
			return nil, err
		}
		return writeResult("probe_project_environment", map[string]string{"project_id": input.ProjectId, "environment_id": environment.Id}, "POST", "/api/project/"+input.ProjectId+"/environment/probe", map[string]any{"environment": environmentOutput(environment)}), nil
	})
}
