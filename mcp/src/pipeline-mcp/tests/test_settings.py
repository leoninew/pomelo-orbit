from __future__ import annotations

import pytest

from pomelo_pipeline_mcp.settings import Settings, SettingsError


def test_load_requires_a_url_and_short_lived_jwt() -> None:
    with pytest.raises(SettingsError, match="POMELO_PIPELINE_URL"):
        Settings.load({"POMELO_PIPELINE_JWT": "token"})

    with pytest.raises(SettingsError, match="POMELO_PIPELINE_JWT"):
        Settings.load({"POMELO_PIPELINE_URL": "http://pipeline.test"})


def test_load_accepts_absolute_http_url() -> None:
    settings = Settings.load({"POMELO_PIPELINE_URL": "https://pipeline.test/", "POMELO_PIPELINE_JWT": "short-lived"})

    assert settings.pipeline_url == "https://pipeline.test"
    assert settings.jwt == "short-lived"


@pytest.mark.parametrize("url", ["pipeline.test", "file:///tmp/pipeline", "ftp://pipeline.test"])
def test_load_rejects_non_http_urls(url: str) -> None:
    with pytest.raises(SettingsError, match="absolute HTTP"):
        Settings.load({"POMELO_PIPELINE_URL": url, "POMELO_PIPELINE_JWT": "token"})
