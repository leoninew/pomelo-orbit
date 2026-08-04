package repositorysource

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	repositoryport "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/port"
)

var _ repositoryport.LocalDirectorySource = Source{}

type PhysicalPathResolver func(ctx context.Context, logicalPath string) (string, error)

type Source struct {
	resolver PhysicalPathResolver
}

func New(resolver PhysicalPathResolver) Source {
	return Source{resolver: resolver}
}

func (s Source) Validate(ctx context.Context, localPath string) (string, error) {
	path, err := s.resolvePath(localPath)
	if err != nil {
		return "", err
	}
	if err := inspectGitRepository(ctx, path); err != nil {
		return "", err
	}
	return filepath.ToSlash(path), nil
}

func (s Source) ResolveRevision(ctx context.Context, localPath string, ref string) (string, error) {
	path, err := s.resolvePath(localPath)
	if err != nil {
		return "", err
	}
	if err := inspectGitRepository(ctx, path); err != nil {
		return "", err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("source revision is required")
	}
	output, err := gitOutput(ctx, path, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve source revision %q: %w", ref, err)
	}
	return strings.TrimSpace(output), nil
}

func (s Source) DockerHostPath(ctx context.Context, localPath string) (string, error) {
	path, err := s.resolvePath(localPath)
	if err != nil {
		return "", err
	}
	if err := inspectGitRepository(ctx, path); err != nil {
		return "", err
	}
	if s.resolver == nil {
		return path, nil
	}
	hostPath, err := s.resolver(ctx, path)
	if err != nil {
		return "", fmt.Errorf("resolve local source host path: %w", err)
	}
	return hostPath, nil
}

func (s Source) resolvePath(localPath string) (string, error) {
	localPath = filepath.FromSlash(strings.TrimSpace(localPath))
	if localPath == "" {
		return "", fmt.Errorf("local source path is required")
	}
	candidate, err := filepath.Abs(localPath)
	if err != nil {
		return "", fmt.Errorf("resolve local source path: %w", err)
	}
	path, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve local source path: %w", err)
	}
	return path, nil
}

func inspectGitRepository(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat local source: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("local source path must be a directory")
	}
	isBare, err := gitOutput(ctx, path, "rev-parse", "--is-bare-repository")
	if err != nil {
		return fmt.Errorf("local source is not a Git repository: %w", err)
	}
	if strings.TrimSpace(isBare) == "true" {
		return nil
	}
	isWorkTree, err := gitOutput(ctx, path, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(isWorkTree) != "true" {
		return fmt.Errorf("local source must be a Git worktree or bare repository")
	}
	gitDir, err := gitPath(ctx, path, "--git-dir")
	if err != nil {
		return err
	}
	commonDir, err := gitPath(ctx, path, "--git-common-dir")
	if err != nil {
		return err
	}
	if gitDir != commonDir {
		return fmt.Errorf("linked Git worktrees are not supported as local sources")
	}
	return nil
}

func gitPath(ctx context.Context, repositoryPath string, arg string) (string, error) {
	output, err := gitOutput(ctx, repositoryPath, "rev-parse", arg)
	if err != nil {
		return "", fmt.Errorf("inspect Git metadata: %w", err)
	}
	value := strings.TrimSpace(output)
	if !filepath.IsAbs(value) {
		value = filepath.Join(repositoryPath, value)
	}
	resolved, err := filepath.EvalSymlinks(value)
	if err != nil {
		return "", fmt.Errorf("resolve Git metadata: %w", err)
	}
	return filepath.Clean(resolved), nil
}

func gitOutput(ctx context.Context, repositoryPath string, args ...string) (string, error) {
	commandArgs := append([]string{"-C", repositoryPath}, args...)
	output, err := exec.CommandContext(ctx, "git", commandArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}
