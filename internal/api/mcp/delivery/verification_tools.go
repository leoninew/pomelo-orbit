package delivery

import (
	"context"
	"time"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerVerificationTools(server *mcp.Server) {
	addTool(server, "verify_deployment", "Verify deployment state with a concise result; set detail=true for raw diagnostic evidence.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		DeploymentId  string `json:"deployment_id" jsonschema:"required"`
		Detail        bool   `json:"detail,omitempty"`
	}) (map[string]any, error) {
		result, err := c.deps.Deployment.VerifyDeployment(ctx, c.deps.ActorUserId, input.ApplicationId, input.DeploymentId, deploymentdto.DeploymentVerificationInput{StabilityWindow: 60 * time.Second, StabilityPoll: 2 * time.Second})
		if err != nil {
			return nil, err
		}
		output := map[string]any{"application_id": input.ApplicationId, "deployment_id": input.DeploymentId, "conclusion": result.Conclusion, "differences": result.Differences}
		if input.Detail {
			output["evidence"] = result.Evidence
		} else {
			for key, value := range result.Summary {
				output[key] = value
			}
		}
		return output, nil
	})
}
