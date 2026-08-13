package applicationsvc

import (
	"context"
	"fmt"

	applicationport "github.com/leoninew/pomelo-orbit/internal/application/application/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type versionForkStore interface {
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	CreateVersionWithVersionComponents(ctx context.Context, version model.Version, components []model.VersionComponent) error
}

type buildVersionForkStore interface {
	versionForkStore
	Version(ctx context.Context, id string) (model.Version, error)
}

func forkVersion(ctx context.Context, store versionForkStore, source model.Version, label string, mutate func([]model.VersionComponent) error) (model.Version, error) {
	components, err := store.VersionComponentsByVersion(ctx, source.Id)
	if err != nil {
		return model.Version{}, fmt.Errorf("list source version components: %w", err)
	}
	fromId := source.Id
	version := model.Version{
		Id:                   idutil.NewId(),
		ApplicationId:        source.ApplicationId,
		Label:                label,
		Status:               status.VersionStatusUnpublished,
		CreatedFromVersionId: &fromId,
		Note:                 source.Note,
	}
	for index := range components {
		components[index].Id = idutil.NewId()
		components[index].VersionId = version.Id
	}
	if mutate != nil {
		if err := mutate(components); err != nil {
			return model.Version{}, err
		}
	}
	if err := store.CreateVersionWithVersionComponents(ctx, version, components); err != nil {
		return model.Version{}, fmt.Errorf("create forked version: %w", err)
	}
	return version, nil
}

// ForkVersionForBuild forks one Version and applies all component-bound image
// artifacts from the same PipelineRun atomically.
func (s Service) ForkVersionForBuild(ctx context.Context, input applicationport.BuildVersionForkInput) (model.Version, error) {
	return forkVersionForBuild(ctx, s.store, input)
}

func forkVersionForBuild(ctx context.Context, store buildVersionForkStore, input applicationport.BuildVersionForkInput) (model.Version, error) {
	source, err := store.Version(ctx, input.SourceVersionId)
	if err != nil {
		return model.Version{}, fmt.Errorf("load source version: %w", err)
	}
	return forkVersion(ctx, store, source, input.Label, func(components []model.VersionComponent) error {
		updates := make(map[string]applicationport.BuildVersionComponentUpdate, len(input.Components))
		for _, update := range input.Components {
			updates[update.ComponentName] = update
		}
		for index := range components {
			update, exists := updates[components[index].Name]
			if !exists {
				continue
			}
			components[index].Image = update.Image
			components[index].ArtifactId = &update.ArtifactId
			components[index].Artifact = &model.VersionComponentArtifact{
				ArtifactId: update.ArtifactId, ArtifactName: update.ArtifactName,
				ImageRef: update.Image, LocalImageSha256: update.LocalImageSha256,
				SourceCommitSha: update.SourceCommitSha,
			}
			delete(updates, components[index].Name)
		}
		if len(updates) == 0 {
			return nil
		}
		for name := range updates {
			return fmt.Errorf("source version does not contain component %s", name)
		}
		return nil
	})
}
