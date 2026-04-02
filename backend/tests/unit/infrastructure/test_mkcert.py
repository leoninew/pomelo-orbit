"""Tests for MkcertService."""

import subprocess
from unittest.mock import Mock, patch

import pytest

from pomelo_orbit.infrastructure.cd.cert.mkcert import MkcertService


@pytest.fixture
def mkcert_service():
    """Create MkcertService instance."""
    return MkcertService()


class TestIsMkcertAvailable:
    """Tests for is_mkcert_available method."""

    def test_returns_true_when_mkcert_available(self, mkcert_service):
        """Should return True when mkcert is available."""
        with patch("subprocess.run") as mock_run:
            mock_run.return_value = Mock(returncode=0)
            assert mkcert_service.is_mkcert_available() is True
            mock_run.assert_called_once_with(
                ["which", "mkcert"],
                capture_output=True,
                text=True,
                timeout=10,
            )

    def test_returns_false_when_mkcert_not_found(self, mkcert_service):
        """Should return False when mkcert not found."""
        with patch("subprocess.run", side_effect=FileNotFoundError):
            assert mkcert_service.is_mkcert_available() is False

    def test_returns_false_when_mkcert_fails(self, mkcert_service):
        """Should return False when mkcert command fails."""
        with patch("subprocess.run") as mock_run:
            mock_run.return_value = Mock(returncode=1)
            assert mkcert_service.is_mkcert_available() is False

    def test_returns_false_when_timeout(self, mkcert_service):
        """Should return False when mkcert times out."""
        with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("mkcert", 10)):
            assert mkcert_service.is_mkcert_available() is False


class TestIsCAInstalled:
    """Tests for is_ca_installed method."""

    def test_returns_true_when_ca_installed(self, mkcert_service, tmp_path):
        """Should return True when CA is installed."""
        ca_root = tmp_path / "ca"
        ca_root.mkdir()
        (ca_root / "rootCA.pem").write_text("test")

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = Mock(returncode=0, stdout=str(ca_root) + "\n")
            assert mkcert_service.is_ca_installed() is True

    def test_returns_false_when_ca_not_installed(self, mkcert_service, tmp_path):
        """Should return False when CA not installed."""
        ca_root = tmp_path / "ca"
        ca_root.mkdir()

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = Mock(returncode=0, stdout=str(ca_root) + "\n")
            assert mkcert_service.is_ca_installed() is False

    def test_returns_false_when_mkcert_not_found(self, mkcert_service):
        """Should return False when mkcert not found."""
        with patch("subprocess.run", side_effect=FileNotFoundError):
            assert mkcert_service.is_ca_installed() is False

    def test_returns_false_when_mkcert_fails(self, mkcert_service):
        """Should return False when mkcert command fails."""
        with patch("subprocess.run") as mock_run:
            mock_run.return_value = Mock(returncode=1)
            assert mkcert_service.is_ca_installed() is False

    def test_returns_false_when_timeout(self, mkcert_service):
        """Should return False when mkcert times out."""
        with patch("subprocess.run", side_effect=subprocess.TimeoutExpired("mkcert", 5)):
            assert mkcert_service.is_ca_installed() is False


class TestGenerateCert:
    """Tests for generate_cert method."""

    def test_raises_error_when_mkcert_not_available(self, mkcert_service):
        """Should raise ValueError when mkcert not available."""
        with (
            patch.object(mkcert_service, "is_mkcert_available", return_value=False),
            pytest.raises(ValueError, match="mkcert 未安装或不可用"),
        ):
            mkcert_service.generate_cert("example.com")

    def test_raises_error_when_ca_not_installed(self, mkcert_service):
        """Should raise ValueError when CA not installed."""
        with (
            patch.object(mkcert_service, "is_mkcert_available", return_value=True),
            patch.object(mkcert_service, "is_ca_installed", return_value=False),
            pytest.raises(ValueError, match="mkcert CA 未安装"),
        ):
            mkcert_service.generate_cert("example.com")

    def test_generates_cert_successfully(self, mkcert_service):
        """Should generate certificate successfully."""
        with (
            patch.object(mkcert_service, "is_mkcert_available", return_value=True),
            patch.object(mkcert_service, "is_ca_installed", return_value=True),
            patch("subprocess.run") as mock_run,
            patch("pathlib.Path.read_text") as mock_read,
        ):
            mock_run.return_value = Mock(returncode=0, stderr="")
            mock_read.side_effect = ["cert-content", "key-content"]

            cert_pem, key_pem = mkcert_service.generate_cert("example.com")

            assert cert_pem == "cert-content"
            assert key_pem == "key-content"
            assert mock_run.called

    def test_raises_error_when_generation_fails(self, mkcert_service):
        """Should raise ValueError when certificate generation fails."""
        with (
            patch.object(mkcert_service, "is_mkcert_available", return_value=True),
            patch.object(mkcert_service, "is_ca_installed", return_value=True),
            patch("subprocess.run") as mock_run,
            pytest.raises(ValueError, match="mkcert 生成证书失败"),
        ):
            mock_run.return_value = Mock(returncode=1, stderr="generation failed")
            mkcert_service.generate_cert("example.com")
