package applicationport

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// BuildVersionForkInput identifies the source Version and the build output applied to its target Component.
type BuildVersionForkInput struct {
	SourceVersionId string
	Label           string
	ComponentName   string
	Image           string
	ArtifactId      string
}

// BuildVersionForker creates an unpublished Version from a source Version for a pipeline build.
type BuildVersionForker interface {
	ForkVersionForBuild(ctx context.Context, input BuildVersionForkInput) (model.Version, error)
}
