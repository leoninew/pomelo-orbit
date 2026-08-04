"""Validated configuration for the read-only pipeline MCP process."""

from __future__ import annotations

import os
from collections.abc import Mapping
from dataclasses import dataclass
from urllib.parse import urlparse


class SettingsError(ValueError):
    """Raised when the local MCP configuration is incomplete or unsafe."""


def _required(environ: Mapping[str, str], key: str) -> str:
    value = environ.get(key, "").strip()
    if not value:
        raise SettingsError(f"missing required setting: {key}")
    return value


@dataclass(frozen=True)
class Settings:
    """Settings that keep this MCP restricted to authenticated GET requests."""

    pipeline_url: str
    jwt: str

    @classmethod
    def load(cls, environ: Mapping[str, str] | None = None) -> Settings:
        values = os.environ if environ is None else environ
        pipeline_url = _required(values, "POMELO_PIPELINE_URL").rstrip("/")
        parsed = urlparse(pipeline_url)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc:
            raise SettingsError("POMELO_PIPELINE_URL must be an absolute HTTP(S) URL")
        return cls(pipeline_url=pipeline_url, jwt=_required(values, "POMELO_PIPELINE_JWT"))
