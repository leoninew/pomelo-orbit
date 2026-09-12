package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/pipelineworkspace"
)

// newPipelineWorkspace keeps CI under the same Environment root as CD. CI is
// executed by the control-plane Docker runner, so only local Environments can
// provide a usable bind source; SSH Environments remain deployment targets.
func newPipelineWorkspace(cfg config.Config, stores domainStores, resolver pipelineworkspace.PhysicalWorkspaceResolver) *pipelineworkspace.Workspace {
	return pipelineworkspace.NewWithProjectResolver(
		cfg.Workspace.Root,
		resolver,
		func(ctx context.Context, projectID string) (string, error) {
			environment, err := stores.environment.EnvironmentByProject(ctx, strings.TrimSpace(projectID))
			if err != nil {
				return "", err
			}
			if !environment.IsLocal() {
				return "", errors.New("pipeline workspace requires a local Environment")
			}
			if strings.TrimSpace(environment.WorkspaceRoot) == "" {
				return "", errors.New("environment workspace root must be configured before running a pipeline")
			}
			return environment.WorkspaceRoot, nil
		},
	)
}
