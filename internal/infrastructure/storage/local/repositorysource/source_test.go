package repositorysource

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceValidatesRepositoryAndResolvesRevision(t *testing.T) {
	repositoryPath := filepath.Join(t.TempDir(), "service")
	createGitRepository(t, repositoryPath)
	expectedHostPath := filepath.Join(t.TempDir(), "host-source")

	source := New(func(ctx context.Context, logicalPath string) (string, error) {
		if logicalPath != repositoryPath {
			t.Fatalf("unexpected source path for Docker mount: %q", logicalPath)
		}
		return expectedHostPath, nil
	})
	localPath, err := source.Validate(context.Background(), repositoryPath)
	if err != nil {
		t.Fatal(err)
	}
	if localPath != filepath.ToSlash(repositoryPath) {
		t.Fatalf("unexpected canonical local path: %q", localPath)
	}
	revision, err := source.ResolveRevision(context.Background(), localPath, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	wantRevision := gitOutputForTest(t, repositoryPath, "rev-parse", "HEAD")
	if revision != wantRevision {
		t.Fatalf("unexpected revision: got %q want %q", revision, wantRevision)
	}
	gotHostPath, err := source.DockerHostPath(context.Background(), localPath)
	if err != nil {
		t.Fatal(err)
	}
	if gotHostPath != expectedHostPath {
		t.Fatalf("unexpected Docker host path: %q", gotHostPath)
	}
}

func TestSourceAcceptsRepositoryOutsideDataRoot(t *testing.T) {
	dataRoot := t.TempDir()
	repositoryPath := filepath.Join(t.TempDir(), "source")
	createGitRepository(t, repositoryPath)

	source := New(nil)
	localPath, err := source.Validate(context.Background(), repositoryPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(filepath.FromSlash(localPath), dataRoot) {
		t.Fatalf("source path %q must not be constrained to data root %q", localPath, dataRoot)
	}
}

func TestSourceRejectsLinkedWorktree(t *testing.T) {
	root := t.TempDir()
	mainRepository := filepath.Join(root, "main")
	linkedWorktree := filepath.Join(root, "linked")
	createGitRepository(t, mainRepository)
	runGitForTest(t, mainRepository, "worktree", "add", linkedWorktree)

	source := New(nil)
	if _, err := source.Validate(context.Background(), linkedWorktree); err == nil || !strings.Contains(err.Error(), "linked Git worktrees") {
		t.Fatalf("expected linked worktree to be rejected, got %v", err)
	}
}

func createGitRepository(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, path, "init")
	runGitForTest(t, path, "config", "user.name", "Pomelo Orbit Test")
	runGitForTest(t, path, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, path, "add", "README.md")
	runGitForTest(t, path, "commit", "-m", "initial")
}

func runGitForTest(t *testing.T, repositoryPath string, args ...string) {
	t.Helper()
	if output, err := exec.Command("git", append([]string{"-C", repositoryPath}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func gitOutputForTest(t *testing.T, repositoryPath string, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", repositoryPath}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
