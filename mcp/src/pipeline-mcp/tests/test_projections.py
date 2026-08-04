from __future__ import annotations

from pomelo_pipeline_mcp.client import PipelineResponse
from pomelo_pipeline_mcp.projections import detail, page, run, snapshot, stage, stage_log, template


def test_template_projection_excludes_stage_scripts_and_variable_values() -> None:
    response = PipelineResponse(
        data={
            "id": "template-1",
            "name": "build",
            "stages": [
                {
                    "id": "stage-1",
                    "name": "test",
                    "script": "echo $TOKEN",
                    "image": "golang:1.25",
                    "artifacts": [{"type": "file", "name": "build", "path": "/workspace/dist/app"}],
                }
            ],
            "variable_declarations": [
                {"name": "TOKEN", "value": "secret", "default": "also-secret", "secret": True, "editable": True}
            ],
        },
        request_id="request-1",
    )

    result = detail(response, template)

    assert result["stages"] == [
        {"id": "stage-1", "name": "test", "image": "golang:1.25", "artifacts": [{"type": "file", "name": "build"}]}
    ]
    assert result["variable_declarations"] == [{"name": "TOKEN", "secret": True, "editable": True}]
    assert "script" not in result
    assert "value" not in str(result)
    assert "default" not in str(result)
    assert "/workspace/dist/app" not in str(result)


def test_run_projection_excludes_variables_and_error_messages_from_paginated_results() -> None:
    response = PipelineResponse(
        data={
            "items": [
                {
                    "id": "run-1",
                    "status": "faulted",
                    "error_message": "token=secret",
                    "variables_snapshot": [{"name": "TOKEN", "value": "secret", "secret": True}],
                    "pipeline_stage_runs": [
                        {"id": "stage-run-1", "status": "failed", "error_message": "password=secret"}
                    ],
                }
            ],
            "total": 1,
            "page": 1,
            "per_page": 20,
            "pages": 1,
        },
        request_id="request-2",
    )

    result = page(response, run)

    assert result["items"] == [
        {
            "id": "run-1",
            "status": "faulted",
            "variables_snapshot": [{"name": "TOKEN", "secret": True}],
            "pipeline_stage_runs": [{"id": "stage-run-1", "status": "failed"}],
        }
    ]
    assert "token=secret" not in str(result)
    assert "password=secret" not in str(result)


def test_stage_log_redacts_common_credentials() -> None:
    response = PipelineResponse(
        data={
            "logs": "PASSWORD=secret\nAuthorization: Bearer abcdef\nghp_abcdefghijklmnopqrstuvwxyz\n",
            "offset": 72,
            "is_complete": True,
        },
        request_id="request-3",
    )

    result = stage_log(response, offset=0, max_bytes=1024)

    assert "secret" not in result["logs"]
    assert "abcdef" not in result["logs"]
    assert "ghp_" not in result["logs"]
    assert result["offset"] == len(response.data["logs"].encode("utf-8"))
    assert result["is_complete"] is True
    assert result["truncated"] is False


def test_stage_log_uses_utf8_byte_offset_for_truncated_windows() -> None:
    response = PipelineResponse(
        data={"logs": "alpha-中文-omega", "offset": 18, "is_complete": True}, request_id="request-4"
    )

    result = stage_log(response, offset=10, max_bytes=8)

    assert result["logs"] == "alpha-"
    assert result["offset"] == 16
    assert result["server_offset"] == 18
    assert result["is_complete"] is False
    assert result["truncated"] is True


def test_stage_projection_does_not_expose_script() -> None:
    assert stage({"id": "stage-1", "script": "echo secret", "name": "test"}) == {"id": "stage-1", "name": "test"}


def test_snapshot_projection_does_not_expose_artifact_workspace_paths() -> None:
    response = PipelineResponse(
        data={
            "id": "snapshot-1",
            "stages_snapshot": [
                {
                    "id": "stage-1",
                    "name": "build",
                    "script": "echo $TOKEN",
                    "artifacts": [{"type": "file", "name": "build", "path": "/workspace/dist/app"}],
                }
            ],
        },
        request_id="request-5",
    )

    result = detail(response, snapshot)

    assert result["stages_snapshot"] == [
        {"id": "stage-1", "name": "build", "artifacts": [{"type": "file", "name": "build"}]}
    ]
    assert "/workspace/dist/app" not in str(result)
    assert "echo $TOKEN" not in str(result)
