from __future__ import annotations

from pomelo_pipeline_mcp.settings import Settings


def make_settings() -> Settings:
    return Settings(pipeline_url="http://pipeline.test", jwt="not-for-output")
