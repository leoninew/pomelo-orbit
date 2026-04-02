"""证书值对象和相关领域逻辑"""

from pomelo_orbit.domain.exceptions import BusinessError


class Certificate:
    """证书值对象"""

    @staticmethod
    def parse_pem(pem_content: bytes) -> tuple[str, str]:
        """解析合并的 PEM 文件，返回 (cert_pem, cert_key)"""
        text = pem_content.decode("utf-8")
        cert_blocks: list[str] = []
        key_blocks: list[str] = []
        current: list[str] = []
        in_block = False

        for line in text.splitlines(keepends=True):
            stripped = line.strip()
            if stripped.startswith("-----BEGIN "):
                in_block = True
                current = [line]
            elif stripped.startswith("-----END ") and in_block:
                current.append(line)
                block = "".join(current)
                if "PRIVATE KEY" in stripped:
                    key_blocks.append(block)
                else:
                    cert_blocks.append(block)
                in_block = False
                current = []
            elif in_block:
                current.append(line)

        if not cert_blocks or not key_blocks:
            raise BusinessError("PEM 文件必须同时包含证书和私钥块")

        return "".join(cert_blocks), "".join(key_blocks)
