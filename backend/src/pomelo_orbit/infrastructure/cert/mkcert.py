"""mkcert 证书生成服务"""

import logging
import subprocess
import tempfile
from pathlib import Path

logger = logging.getLogger(__name__)


class MkcertService:
    """mkcert 证书生成服务（无状态）"""

    def is_mkcert_available(self) -> bool:
        """检查 mkcert 是否可用"""
        try:
            # mkcert -version 有网络请求，可能很慢
            result = subprocess.run(
                ["which", "mkcert"],
                capture_output=True,
                text=True,
                timeout=10,
            )
            return result.returncode == 0
        except (FileNotFoundError, subprocess.TimeoutExpired):
            return False

    def is_ca_installed(self) -> bool:
        """检查 CA 是否已安装到系统信任库"""
        try:
            result = subprocess.run(
                ["mkcert", "-CAROOT"],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode == 0:
                ca_root = Path(result.stdout.strip())
                return (ca_root / "rootCA.pem").exists()
            return False
        except (FileNotFoundError, subprocess.TimeoutExpired):
            return False

    def generate_cert(self, domain: str) -> tuple[str, str]:
        """生成 mkcert 证书

        Args:
            domain: 域名

        Returns:
            (cert_pem, key_pem) 元组

        Raises:
            ValueError: 证书生成失败
        """
        # 检查 mkcert 是否可用
        if not self.is_mkcert_available():
            raise ValueError("mkcert 未安装或不可用,请参考 https://github.com/FiloSottile/mkcert#installation")

        # 检查 CA 是否已安装
        if not self.is_ca_installed():
            raise ValueError("mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cert_file = tmp_path / "cert.pem"
            key_file = tmp_path / "key.pem"

            # 调用 mkcert 生成证书
            result = subprocess.run(
                ["mkcert", "-cert-file", str(cert_file), "-key-file", str(key_file), domain],
                capture_output=True,
                text=True,
            )

            if result.returncode != 0:
                raise ValueError(f"mkcert 生成证书失败: {result.stderr}")

            # 读取证书和私钥
            cert_pem = cert_file.read_text()
            key_pem = key_file.read_text()

            return cert_pem, key_pem
