package applicationport

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// BuildVersionComponentUpdate is one image artifact applied while a build forks
// a single Version.
type BuildVersionComponentUpdate struct {
	ComponentName    string
	Image            string
	ArtifactId       string
	ArtifactName     string
	LocalImageSha256 string
	SourceCommitSha  string
}

// BuildVersionForkInput identifies the source Version and all build outputs
// applied atomically to its target Components.
type BuildVersionForkInput struct {
	SourceVersionId string
	Label           string
	Components      []BuildVersionComponentUpdate
}

// BuildVersionForker creates an unpublished Version from a source Version for a pipeline build.
type BuildVersionForker interface {
	ForkVersionForBuild(ctx context.Context, input BuildVersionForkInput) (model.Version, error)
}
