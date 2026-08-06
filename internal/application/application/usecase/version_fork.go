package applicationsvc

import (
	"context"
	"fmt"

	applicationport "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
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

// ForkVersionForBuild forks a Version and applies the image artifact produced by a pipeline stage.
func (s Service) ForkVersionForBuild(ctx context.Context, input applicationport.BuildVersionForkInput) (model.Version, error) {
	return forkVersionForBuild(ctx, s.store, input)
}

func forkVersionForBuild(ctx context.Context, store buildVersionForkStore, input applicationport.BuildVersionForkInput) (model.Version, error) {
	source, err := store.Version(ctx, input.SourceVersionId)
	if err != nil {
		return model.Version{}, fmt.Errorf("load source version: %w", err)
	}
	return forkVersion(ctx, store, source, input.Label, func(components []model.VersionComponent) error {
		for index := range components {
			if components[index].Name != input.ComponentName {
				continue
			}
			components[index].Image = input.Image
			components[index].ArtifactId = &input.ArtifactId
			return nil
		}
		return fmt.Errorf("source version does not contain component %s", input.ComponentName)
	})
}
