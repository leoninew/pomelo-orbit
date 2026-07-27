from __future__ import annotations

from pomelo_orbit_mcp.tools.orbit import application_service_list_result


def test_application_service_list_exposes_only_lifecycle_summary() -> None:
    result = application_service_list_result(
        "application-1",
        [
            {
                "id": "service-1",
                "application_id": "application-1",
                "instance_key": "default",
                "status": "stopped",
                "version_id": "version-1",
                "last_successful_version_id": "version-0",
                "created_at": "2026-07-27T00:00:00Z",
                "updated_at": "2026-07-27T01:00:00Z",
                "runtime_config": {"PASSWORD": "not-for-output"},
                "future_server_field": "not-for-output",
            }
        ],
    )

    assert result == {
        "application_id": "application-1",
        "services": [
            {
                "id": "service-1",
                "application_id": "application-1",
                "instance_key": "default",
                "status": "stopped",
                "version_id": "version-1",
                "last_successful_version_id": "version-0",
                "created_at": "2026-07-27T00:00:00Z",
                "updated_at": "2026-07-27T01:00:00Z",
            }
        ],
    }


def test_application_service_list_omits_optional_missing_fields() -> None:
    result = application_service_list_result(
        "application-1",
        [{"id": "service-1", "application_id": "application-1", "instance_key": "default", "status": "stopped"}],
    )

    assert result["services"] == [
        {
            "id": "service-1",
            "application_id": "application-1",
            "instance_key": "default",
            "status": "stopped",
        }
    ]
