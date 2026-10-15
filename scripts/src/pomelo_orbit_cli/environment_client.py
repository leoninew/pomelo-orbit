"""PAT-only Orbit transport. Credentials and remote payloads are never logged."""

from __future__ import annotations

import base64
import binascii
import json
from dataclasses import dataclass
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

from pomelo_orbit_cli.settings import ClientSettings


@dataclass
class EnvironmentError(Exception):
    code: str
    message: str
    request_id: str = ""
    status: int | None = None

    def __str__(self) -> str:
        suffix = f" (requestId={self.request_id})" if self.request_id else ""
        return f"{self.code}: {self.message}{suffix}"


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(
        self, req: Any, fp: Any, code: int, msg: str, headers: Any, newurl: str
    ) -> None:
        return None


def api_error(status: int, data: bytes) -> EnvironmentError:
    try:
        value = json.loads(data)
        if not isinstance(value, dict) or not all(
            isinstance(value.get(k), str) and value[k]
            for k in ("code", "error", "requestId")
        ):
            raise ValueError
        return EnvironmentError(
            value["code"], value["error"], value["requestId"], status
        )
    except (ValueError, UnicodeError):
        return EnvironmentError(
            "contract_mismatch",
            "Orbit returned an invalid error response",
            status=status,
        )


class EnvironmentClient:
    def __init__(self, settings: ClientSettings):
        self.settings = settings.validated()
        self.opener = build_opener(ProxyHandler({}), NoRedirect())

    def url(self, path: str) -> str:
        assert self.settings.base_url is not None
        return self.settings.base_url.rstrip("/") + "/api/cli/" + path

    def terminal_url(self, project: str) -> str:
        url = self.url("environment/terminal")
        return (
            ("wss:" + url[6:] if url.startswith("https:") else "ws:" + url[5:])
            + "?"
            + urlencode({"project_code": project})
        )

    def request(
        self, path: str, payload: dict[str, Any] | None = None, timeout: int = 15
    ) -> dict[str, Any]:
        req = Request(
            self.url(path),
            data=json.dumps(payload).encode() if payload is not None else None,
            headers={
                "Authorization": f"Bearer {self.settings.access_token}",
                "Content-Type": "application/json",
                "Accept": "application/json",
            },
        )
        try:
            with self.opener.open(req, timeout=timeout) as response:
                data = response.read(4 * 1024 * 1024 + 1)
                request_id = response.headers.get("X-Request-Id", "")
                content_type = response.headers.get_content_type()
            if len(data) > 4 * 1024 * 1024 or content_type != "application/json":
                raise ValueError
            value = json.loads(data)
            if not isinstance(value, dict):
                raise ValueError
            value["requestId"] = request_id
            return value
        except HTTPError as error:
            try:
                raise api_error(error.code, error.read(64 * 1024)) from None
            finally:
                error.close()
        except (URLError, OSError, TimeoutError):
            message = "Orbit connection failed"
            if payload is not None:
                message += "; command result is unknown; do not retry automatically"
            raise EnvironmentError("transport_error", message) from None
        except (ValueError, UnicodeError):
            raise EnvironmentError(
                "contract_mismatch", "Orbit returned an invalid response"
            ) from None

    def projects(self) -> dict[str, Any]:
        value = self.request("project")
        projects = value.get("projects")
        if not isinstance(projects, list) or not all(
            isinstance(p, dict)
            and all(
                isinstance(p.get(k), str)
                for k in (
                    "id",
                    "name",
                    "code",
                    "target_type",
                    "environment_id",
                    "platform",
                    "host",
                    "username",
                )
            )
            and isinstance(p.get("is_active"), bool)
            and type(p.get("port")) is int
            for p in projects
        ):
            raise EnvironmentError("contract_mismatch", "Invalid project list")
        return value

    def execute(self, payload: dict[str, Any], timeout: int) -> dict[str, Any]:
        value = self.request("environment/exec", payload, timeout + 20)
        try:
            if value.get("state") not in {
                "exited",
                "timed_out",
                "output_limit",
                "access_or_target_changed",
                "transport_lost",
            }:
                raise ValueError
            for key in ("project_id", "project_code", "environment_id"):
                if not isinstance(value.get(key), str):
                    raise ValueError
            if type(value.get("output_truncated")) is not bool:
                raise ValueError
            for key in ("target_revision", "duration_ms"):
                value[key] = int(value[key])
            if value["state"] == "exited":
                if type(value.get("exit_code")) is not int:
                    raise ValueError
            elif "exit_code" in value:
                raise ValueError
            for key in ("stdout", "stderr"):
                value[key] = base64.b64decode(value[key], validate=True)
        except (ValueError, KeyError, TypeError, binascii.Error):
            raise EnvironmentError(
                "contract_mismatch",
                "Invalid command result; do not retry automatically",
            ) from None
        return value
