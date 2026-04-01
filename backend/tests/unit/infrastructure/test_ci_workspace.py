"""CI Workspace 单元测试"""

import tempfile
from pathlib import Path
from unittest.mock import patch

from pomelo_orbit.infrastructure.ci.workspace import (
    cleanup_workspace,
    create_workspace,
    get_artifacts_path,
    get_workspace_path,
)

PATCH_ROOT = "pomelo_orbit.infrastructure.ci.workspace.get_project_root"


class TestGetWorkspacePath:

    @patch(PATCH_ROOT)
    def test_returns_correct_path(self, mock_root):
        mock_root.return_value = Path("/project")
        assert get_workspace_path("run-123") == Path("/project/data/ci/runs/run-123/workspace")

    @patch(PATCH_ROOT)
    def test_different_run_ids(self, mock_root):
        mock_root.return_value = Path("/project")
        path1 = get_workspace_path("run-1")
        path2 = get_workspace_path("run-2")
        assert path1 != path2
        assert "run-1" in str(path1)
        assert "run-2" in str(path2)


class TestGetArtifactsPath:

    @patch(PATCH_ROOT)
    def test_returns_correct_path(self, mock_root):
        mock_root.return_value = Path("/project")
        assert get_artifacts_path("run-123") == Path("/project/data/ci/runs/run-123/artifacts")

    @patch(PATCH_ROOT)
    def test_different_run_ids(self, mock_root):
        mock_root.return_value = Path("/project")
        path1 = get_artifacts_path("run-1")
        path2 = get_artifacts_path("run-2")
        assert path1 != path2


class TestCreateWorkspace:

    def test_creates_directories(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, artifacts_path = create_workspace("test-run")
            assert workspace_path.exists() and workspace_path.is_dir()
            assert artifacts_path.exists() and artifacts_path.is_dir()

    def test_returns_correct_paths(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, artifacts_path = create_workspace("test-run")
            assert "workspace" in str(workspace_path) and "test-run" in str(workspace_path)
            assert "artifacts" in str(artifacts_path) and "test-run" in str(artifacts_path)

    def test_creates_parent_directories(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, artifacts_path = create_workspace("test-run")
            assert workspace_path.parent.exists()
            assert artifacts_path.parent.exists()

    def test_idempotent_creation(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            wp1, ap1 = create_workspace("test-run")
            wp2, ap2 = create_workspace("test-run")
            assert wp1 == wp2 and ap1 == ap2
            assert wp1.exists() and ap1.exists()


class TestCleanupWorkspace:

    def test_removes_workspace_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, _ = create_workspace("test-run")
            assert workspace_path.exists()
            cleanup_workspace("test-run")
            assert not workspace_path.exists()

    def test_keeps_artifacts_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            _, artifacts_path = create_workspace("test-run")
            cleanup_workspace("test-run")
            assert not artifacts_path.exists()

    def test_cleanup_nonexistent_workspace(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            cleanup_workspace("nonexistent-run")  # should not raise

    def test_cleanup_with_files(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, _ = create_workspace("test-run")
            test_file = workspace_path / "test.txt"
            test_file.write_text("test content")
            cleanup_workspace("test-run")
            assert not test_file.exists()
            assert not workspace_path.exists()

    def test_cleanup_with_subdirectories(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, _ = create_workspace("test-run")
            subdir = workspace_path / "subdir"
            subdir.mkdir()
            (subdir / "file.txt").write_text("content")
            cleanup_workspace("test-run")
            assert not subdir.exists()
            assert not workspace_path.exists()
