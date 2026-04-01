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


class TestGetWorkspacePath:
    """测试 get_workspace_path"""

    @patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root")
    def test_returns_correct_path(self, mock_root):
        """测试返回正确的 workspace 路径"""
        mock_root.return_value = Path("/project")

        result = get_workspace_path("run-123")

        assert result == Path("/project/data/ci/runs/run-123/workspace")

    @patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root")
    def test_different_run_ids(self, mock_root):
        """测试不同的 run_id 返回不同路径"""
        mock_root.return_value = Path("/project")

        path1 = get_workspace_path("run-1")
        path2 = get_workspace_path("run-2")

        assert path1 != path2
        assert "run-1" in str(path1)
        assert "run-2" in str(path2)


class TestGetArtifactsPath:
    """测试 get_artifacts_path"""

    @patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root")
    def test_returns_correct_path(self, mock_root):
        """测试返回正确的 artifacts 路径"""
        mock_root.return_value = Path("/project")

        result = get_artifacts_path("run-123")

        assert result == Path("/project/data/ci/runs/run-123/artifacts")

    @patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root")
    def test_different_run_ids(self, mock_root):
        """测试不同的 run_id 返回不同路径"""
        mock_root.return_value = Path("/project")

        path1 = get_artifacts_path("run-1")
        path2 = get_artifacts_path("run-2")

        assert path1 != path2
        assert "run-1" in str(path1)
        assert "run-2" in str(path2)


class TestCreateWorkspace:
    """测试 create_workspace"""

    def test_creates_directories(self):
        """测试创建目录"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                workspace_path, artifacts_path = create_workspace("test-run")

                assert workspace_path.exists()
                assert workspace_path.is_dir()
                assert artifacts_path.exists()
                assert artifacts_path.is_dir()

    def test_returns_correct_paths(self):
        """测试返回正确的路径"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                workspace_path, artifacts_path = create_workspace("test-run")

                assert "workspace" in str(workspace_path)
                assert "artifacts" in str(artifacts_path)
                assert "test-run" in str(workspace_path)
                assert "test-run" in str(artifacts_path)

    def test_creates_parent_directories(self):
        """测试创建父目录"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                workspace_path, artifacts_path = create_workspace("test-run")

                # 验证父目录也被创建
                assert workspace_path.parent.exists()
                assert artifacts_path.parent.exists()

    def test_idempotent_creation(self):
        """测试重复创建不会报错"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                # 第一次创建
                workspace_path1, artifacts_path1 = create_workspace("test-run")

                # 第二次创建（应该不报错）
                workspace_path2, artifacts_path2 = create_workspace("test-run")

                assert workspace_path1 == workspace_path2
                assert artifacts_path1 == artifacts_path2
                assert workspace_path1.exists()
                assert artifacts_path1.exists()


class TestCleanupWorkspace:
    """测试 cleanup_workspace"""

    def test_removes_workspace_directory(self):
        """测试删除 workspace 目录"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                # 创建工作目录
                workspace_path, _ = create_workspace("test-run")
                assert workspace_path.exists()

                # 清理
                cleanup_workspace("test-run")

                # 验证 workspace 被删除
                assert not workspace_path.exists()

    def test_keeps_artifacts_directory(self):
        """测试保留 artifacts 目录"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                # 创建工作目录
                _, artifacts_path = create_workspace("test-run")
                assert artifacts_path.exists()

                # 清理
                cleanup_workspace("test-run")

                # 验证 artifacts 仍然存在
                assert artifacts_path.exists()

    def test_cleanup_nonexistent_workspace(self):
        """测试清理不存在的 workspace 不报错"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                # 清理不存在的 workspace（应该不报错）
                cleanup_workspace("nonexistent-run")

    def test_cleanup_with_files(self):
        """测试清理包含文件的 workspace"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                # 创建工作目录并添加文件
                workspace_path, _ = create_workspace("test-run")
                test_file = workspace_path / "test.txt"
                test_file.write_text("test content")
                assert test_file.exists()

                # 清理
                cleanup_workspace("test-run")

                # 验证文件和目录都被删除
                assert not test_file.exists()
                assert not workspace_path.exists()

    def test_cleanup_with_subdirectories(self):
        """测试清理包含子目录的 workspace"""
        with tempfile.TemporaryDirectory() as tmpdir:
            with patch("pomelo_orbit.infrastructure.ci.workspace.get_project_root") as mock_root:
                mock_root.return_value = Path(tmpdir)

                # 创建工作目录并添加子目录
                workspace_path, _ = create_workspace("test-run")
                subdir = workspace_path / "subdir"
                subdir.mkdir()
                (subdir / "file.txt").write_text("content")

                # 清理
                cleanup_workspace("test-run")

                # 验证子目录和文件都被删除
                assert not subdir.exists()
                assert not workspace_path.exists()
