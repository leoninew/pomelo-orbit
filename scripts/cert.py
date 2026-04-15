#!/usr/bin/env python3
"""
cert.py - 证书工具

用法:
  uv run --project backend/ python scripts/cert.py new -n <domain>
  uv run --project backend/ python scripts/cert.py check -n <domain>
"""

import argparse
import datetime
import logging
import socket
import ssl
import subprocess
import sys
import tempfile
from pathlib import Path

logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger(__name__)

SCRIPT_DIR = Path(__file__).parent.resolve()
CERT_DIR = SCRIPT_DIR.parent / "backend/data/cd/traefik/data/certs"


def run(
    cmd: list[str],
    capture: bool = True,
    input: str | None = None,
    encoding: str = "utf-8",
) -> subprocess.CompletedProcess:
    return subprocess.run(
        cmd,
        capture_output=capture,
        text=True,
        input=input,
        encoding=encoding,
        errors="ignore",
    )


def _mkcert_ca_path() -> Path | None:
    r = run(["mkcert", "-CAROOT"])
    if r.returncode != 0:
        return None
    return Path(r.stdout.strip()) / "rootCA.pem"


def _load_cert_from_pem(pem_text: str):
    from cryptography import x509  # noqa: PLC0415
    from cryptography.hazmat.backends import default_backend  # noqa: PLC0415

    lines = pem_text.splitlines()
    block: list[str] = []
    in_block = False
    for line in lines:
        if line.strip() == "-----BEGIN CERTIFICATE-----":
            in_block = True
            block = [line]
        elif line.strip() == "-----END CERTIFICATE-----" and in_block:
            block.append(line)
            break
        elif in_block:
            block.append(line)
    pem_bytes = ("\n".join(block) + "\n").encode()
    return x509.load_pem_x509_certificate(pem_bytes, default_backend())


def _check_ca_in_windows_store(ca_cert) -> bool:
    r = run(["certutil", "-store", "Root"])
    if r.returncode != 0 or not r.stdout:
        return False
    return "mkcert" in r.stdout.lower()


def _verify_cert_chain(leaf_cert, ca_cert) -> bool:
    try:
        leaf_cert.verify_directly_issued_by(ca_cert)  # type: ignore[attr-defined]
        return True
    except Exception:
        pass
    try:
        from cryptography.hazmat.primitives.asymmetric import padding, ec

        pub = ca_cert.public_key()
        sig_algo = leaf_cert.signature_hash_algorithm
        pub.verify(
            leaf_cert.signature,
            leaf_cert.tbs_certificate_bytes,
            ec.ECDSA(sig_algo)
            if "ec" in type(pub).__name__.lower()
            else padding.PKCS1v15(),
            sig_algo,
        )
        return True
    except Exception:
        return False


def _print_cert_info(cert, domain: str) -> tuple[bool, bool]:
    from cryptography import x509 as x509_mod  # noqa: PLC0415

    subject = cert.subject.rfc4514_string()
    issuer = cert.issuer.rfc4514_string()
    expiry = (
        cert.not_valid_after_utc
        if hasattr(cert, "not_valid_after_utc")
        else cert.not_valid_after
    )  # type: ignore[attr-defined]
    now = datetime.datetime.now(tz=expiry.tzinfo)
    days = (expiry - now).days

    logger.info(f"Subject: {subject}")
    logger.info(f"Issuer: {issuer}")
    if days > 0:
        logger.info(f"有效期剩余: {days} 天（到期: {expiry.strftime('%Y-%m-%d')}）")
    else:
        logger.warning(f"有效期剩余: {days} 天（到期: {expiry.strftime('%Y-%m-%d')}）")

    try:
        san_ext = cert.extensions.get_extension_for_class(
            x509_mod.SubjectAlternativeName
        )
        sans = san_ext.value.get_values_for_type(x509_mod.DNSName)
    except Exception:
        sans = []
    logger.info(f"SAN: {', '.join(sans) or '(无)'}")

    san_ok = domain in sans
    is_mkcert = "mkcert" in issuer.lower()
    if san_ok:
        logger.info(f"SAN 包含 {domain}")
    else:
        logger.warning(f"SAN 不包含 {domain}")
    if is_mkcert:
        logger.info("由 mkcert 签发")
    else:
        logger.warning("非 mkcert 签发")
    return san_ok, is_mkcert


def cmd_new(domain: str) -> None:
    CERT_DIR.mkdir(parents=True, exist_ok=True)
    out_file = CERT_DIR / f"{domain}.pem"

    logger.info(f"生成证书: {domain}")
    logger.info(f"输出目录: {CERT_DIR}")

    with tempfile.TemporaryDirectory(dir=CERT_DIR) as tmp:
        tmp_path = Path(tmp)
        cert_file = tmp_path / "cert.pem"
        key_file = tmp_path / "key.pem"

        def to_native(p: Path) -> str:
            r = run(["cygpath", "-w", str(p)])
            return r.stdout.strip() if r.returncode == 0 else str(p)

        result = run(
            [
                "mkcert",
                "-cert-file",
                to_native(cert_file),
                "-key-file",
                to_native(key_file),
                domain,
            ]
        )
        if result.returncode != 0:
            logger.error(result.stderr)
            sys.exit(1)

        out_file.write_bytes(cert_file.read_bytes() + key_file.read_bytes())

    first_line = out_file.read_text().splitlines()[0]
    if first_line != "-----BEGIN CERTIFICATE-----":
        logger.error(f"合并结果首行异常: {first_line}")
        sys.exit(1)

    pem_text = out_file.read_text()
    logger.info(f"输出: {out_file}")
    logger.info(
        f"证书块: {pem_text.count('BEGIN CERTIFICATE')}  私钥块: {pem_text.count('BEGIN PRIVATE KEY')}"
    )


def cmd_check(domain: str) -> None:
    from cryptography import x509  # noqa: PLC0415
    from cryptography.hazmat.backends import default_backend  # noqa: PLC0415

    logger.info(f"检查域名: {domain}")

    logger.info("\n1. mkcert 根 CA")
    ca_path = _mkcert_ca_path()
    ca_cert = None
    if not ca_path or not ca_path.exists():
        logger.warning("找不到 mkcert 根 CA（运行 mkcert -install）")
    else:
        logger.info(f"CA 文件: {ca_path}")
        ca_cert = x509.load_pem_x509_certificate(
            ca_path.read_bytes(), default_backend()
        )
        logger.info(f"CA Subject: {ca_cert.subject.rfc4514_string()}")
        if _check_ca_in_windows_store(ca_cert):
            logger.info("已导入 Windows 系统信任库（Root store）")
        else:
            logger.warning("未在 Windows 系统信任库中找到 mkcert CA")
            logger.info(f'修复: certutil -addstore "Root" "{ca_path}"')

    logger.info("\n2. 本地证书文件")
    pem_file = CERT_DIR / f"{domain}.pem"
    leaf_cert = None
    if not pem_file.exists():
        logger.warning(f"文件不存在: {pem_file}")
        logger.info(
            f"修复: uv run --project backend/ python scripts/cert.py new -n {domain}"
        )
    else:
        logger.info(f"文件: {pem_file}")
        pem_text = pem_file.read_text(encoding="utf-8")
        cert_count = pem_text.count("BEGIN CERTIFICATE")
        key_count = pem_text.count("BEGIN PRIVATE KEY") + pem_text.count(
            "BEGIN EC PRIVATE KEY"
        )
        logger.info(f"证书块: {cert_count}  私钥块: {key_count}")
        if cert_count >= 1:
            logger.info("包含证书")
        else:
            logger.warning("不包含证书")
        if key_count >= 1:
            logger.info("包含私钥")
        else:
            logger.warning("不包含私钥")

        logger.info("\n3. 证书内容")
        try:
            leaf_cert = _load_cert_from_pem(pem_text)
            _print_cert_info(leaf_cert, domain)
        except Exception as e:
            logger.error(f"解析证书失败: {e}")

    logger.info("\n4. 信任链验证（本地 CA → 叶证书）")
    if leaf_cert and ca_cert:
        if _verify_cert_chain(leaf_cert, ca_cert):
            logger.info("叶证书由 mkcert CA 签名验证通过")
        else:
            logger.warning("签名验证失败，证书不是由当前 mkcert CA 签发")
    else:
        logger.info("跳过（CA 或证书文件缺失）")

    logger.info("\n5. TLS 握手（系统信任库验证，模拟浏览器）")
    try:
        ctx = ssl.create_default_context()
        ctx.check_hostname = True
        ctx.verify_mode = ssl.CERT_REQUIRED
        with socket.create_connection((domain, 443), timeout=5) as sock:
            with ctx.wrap_socket(sock, server_hostname=domain) as ssock:
                der = ssock.getpeercert(binary_form=True)
                proto = ssock.version()

        logger.info(f"TLS 握手成功（{proto}）— 系统信任库验证通过，浏览器不会报红")

        assert der is not None, "未获取到服务端证书"
        server_cert = x509.load_der_x509_certificate(der, default_backend())
        _, is_mkcert = _print_cert_info(server_cert, domain)
        if not is_mkcert:
            logger.warning("服务端仍在使用 fallback 自签名证书，证书未下发生效")

    except ssl.SSLCertVerificationError as e:
        logger.error(f"证书验证失败（浏览器会报红）: {e.reason}")
        logger.info("可能原因: mkcert CA 未导入系统信任库，或服务端证书不匹配")
    except ConnectionRefusedError:
        logger.error("连接被拒绝（443 端口未监听）")
    except socket.timeout:
        logger.error(f"连接超时: {domain}:443")
    except Exception as e:
        logger.error(f"TLS 握手失败: {e}")


def main() -> None:
    parser = argparse.ArgumentParser(
        prog="cert.py",
        description="mkcert 证书工具",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=(
            "示例:\n"
            "  uv run --project backend/ python scripts/cert.py new -n pomelo-orbit.localhost\n"
            "  uv run --project backend/ python scripts/cert.py check -n pomelo-orbit.localhost"
        ),
    )
    sub = parser.add_subparsers(dest="cmd")

    p_new = sub.add_parser("new", help="生成 mkcert 证书并输出合并 PEM")
    p_new.add_argument("-n", dest="domain", required=True, metavar="domain")

    p_check = sub.add_parser("check", help="检查证书信任链（CA → 叶证书 → TLS 握手）")
    p_check.add_argument("-n", dest="domain", required=True, metavar="domain")

    args = parser.parse_args()

    if args.cmd == "new":
        cmd_new(args.domain)
    elif args.cmd == "check":
        cmd_check(args.domain)
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
