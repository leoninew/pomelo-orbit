"""CI Workspace 单元测试"""

import tempfile
from pathlib import Path
from unittest.mock import patch

from pomelo_orbit.infrastructure.ci.workspace import (
    cleanup_project,
    cleanup_run_secrets,
    create_workspace,
    get_artifacts_path,
    get_workspace_path,
)

PATCH_ROOT = "pomelo_orbit.infrastructure.ci.workspace.get_project_root"
PROJECT_CODE = "my-project"
RUN_ID = "run-123"


class TestGetWorkspacePath:
    @patch(PATCH_ROOT)
    def test_returns_correct_path(self, mock_root):
        mock_root.return_value = Path("/project")
        assert get_workspace_path(PROJECT_CODE) == Path(f"/project/data/ci/{PROJECT_CODE}/workspace")

    @patch(PATCH_ROOT)
    def test_same_project_same_path(self, mock_root):
        mock_root.return_value = Path("/project")
        assert get_workspace_path(PROJECT_CODE) == get_workspace_path(PROJECT_CODE)

    @patch(PATCH_ROOT)
    def test_different_projects_different_paths(self, mock_root):
        mock_root.return_value = Path("/project")
        assert get_workspace_path("proj-a") != get_workspace_path("proj-b")


class TestGetArtifactsPath:
    @patch(PATCH_ROOT)
    def test_returns_correct_path(self, mock_root):
        mock_root.return_value = Path("/project")
        assert get_artifacts_path(RUN_ID) == Path(f"/project/data/ci/runs/{RUN_ID}/artifacts")

    @patch(PATCH_ROOT)
    def test_different_runs_different_paths(self, mock_root):
        mock_root.return_value = Path("/project")
        assert get_artifacts_path("run-1") != get_artifacts_path("run-2")


class TestCreateWorkspace:
    def test_creates_directories(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, artifacts_path = create_workspace(PROJECT_CODE, RUN_ID)
            assert workspace_path.exists() and workspace_path.is_dir()
            assert artifacts_path.exists() and artifacts_path.is_dir()

    def test_returns_correct_paths(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, artifacts_path = create_workspace(PROJECT_CODE, RUN_ID)
            assert PROJECT_CODE in str(workspace_path) and "workspace" in str(workspace_path)
            assert RUN_ID in str(artifacts_path) and "artifacts" in str(artifacts_path)

    def test_creates_parent_directories(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, artifacts_path = create_workspace(PROJECT_CODE, RUN_ID)
            assert workspace_path.parent.exists()
            assert artifacts_path.parent.exists()

    def test_idempotent_creation(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            wp1, ap1 = create_workspace(PROJECT_CODE, RUN_ID)
            wp2, ap2 = create_workspace(PROJECT_CODE, RUN_ID)
            assert wp1 == wp2 and ap1 == ap2
            assert wp1.exists() and ap1.exists()

    def test_workspace_shared_across_runs(self):
        """同一项目不同 run 共享同一 workspace 目录"""
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            wp1, _ = create_workspace(PROJECT_CODE, "run-1")
            wp2, _ = create_workspace(PROJECT_CODE, "run-2")
            assert wp1 == wp2

    def test_artifacts_isolated_per_run(self):
        """不同 run 的 artifacts 目录相互隔离"""
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            _, ap1 = create_workspace(PROJECT_CODE, "run-1")
            _, ap2 = create_workspace(PROJECT_CODE, "run-2")
            assert ap1 != ap2


class TestCleanupRun:
    def test_removes_secrets_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            create_workspace(PROJECT_CODE, RUN_ID)
            secrets_path = Path(tmpdir) / "data" / "ci" / "runs" / RUN_ID / "secrets"
            secrets_path.mkdir(parents=True, exist_ok=True)
            (secrets_path / "id_rsa").write_text("private key")
            cleanup_run_secrets(RUN_ID)
            assert not secrets_path.exists()

    def test_keeps_artifacts(self):
        """cleanup_run_secrets 不应删除 artifacts"""
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            _, artifacts_path = create_workspace(PROJECT_CODE, RUN_ID)
            (artifacts_path / "output.tar.gz").write_text("artifact")
            cleanup_run_secrets(RUN_ID)
            assert artifacts_path.exists()

    def test_keeps_workspace(self):
        """cleanup_run_secrets 不应删除 workspace"""
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, _ = create_workspace(PROJECT_CODE, RUN_ID)
            cleanup_run_secrets(RUN_ID)
            assert workspace_path.exists()

    def test_cleanup_nonexistent_run(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            cleanup_run_secrets("nonexistent-run")  # should not raise


class TestCleanupProject:
    def test_removes_entire_project_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            workspace_path, _ = create_workspace(PROJECT_CODE, RUN_ID)
            cleanup_project(PROJECT_CODE)
            assert not workspace_path.exists()

    def test_cleanup_nonexistent_project(self):
        with tempfile.TemporaryDirectory() as tmpdir, patch(PATCH_ROOT) as mock_root:
            mock_root.return_value = Path(tmpdir)
            cleanup_project("nonexistent-project")  # should not raise
