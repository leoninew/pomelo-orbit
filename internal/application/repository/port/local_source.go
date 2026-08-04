package port

import "context"

// LocalDirectorySource validates local Git repositories and resolves their immutable inputs.
type LocalDirectorySource interface {
	Validate(ctx context.Context, localPath string) (string, error)
	ResolveRevision(ctx context.Context, localPath string, ref string) (string, error)
	DockerHostPath(ctx context.Context, localPath string) (string, error)
}
