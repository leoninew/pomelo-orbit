"""CI 模块 API 路由"""

import json
import logging
import math
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Header, Query, Request, status

from pomelo_orbit.application.ci.di import get_ci_webhook_service, get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.application.ci.webhook_service import CIWebhookService
from pomelo_orbit.domain.ci.entities import PipelineTemplate
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger, StageDefinition, VariableDeclaration
from pomelo_orbit.infrastructure.di import get_security_service
from pomelo_orbit.infrastructure.security import SecurityService
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.dto.ci import (
    ArtifactResp,
    CredentialCreateReq,
    CredentialResp,
    CredentialUpdateReq,
    JobLogResp,
    JobResp,
    PipelineRunResp,
    PipelineSnapshotListItemResp,
    PipelineSnapshotResp,
    PipelineTemplateCreateReq,
    PipelineTemplateResp,
    PipelineTemplateUpdateReq,
    ProjectCreateReq,
    ProjectResp,
    ProjectUpdateReq,
    StageDefinitionDto,
    TriggerPipelineReq,
    VariableDeclarationDto,
)
from pomelo_orbit.interfaces.api.dto.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/v1/ci", tags=["ci"])


def _stage_dto(stage: StageDefinition) -> StageDefinitionDto:
    return StageDefinitionDto(**stage.model_dump())


def _decl_dto(decl: VariableDeclaration) -> VariableDeclarationDto:
    return VariableDeclarationDto(**decl.model_dump())


def _paginated_runs(runs: list, total: int, page: int, per_page: int) -> PaginatedResp[PipelineRunResp]:
    return PaginatedResp(
        items=[PipelineRunResp.model_validate(r.__dict__) for r in runs],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


# =============================================================================
# Credentials
# =============================================================================


@router.get("/credentials", response_model=PaginatedResp[CredentialResp])
def list_credentials(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[CredentialResp]:
    creds, total = pipeline_service.list_credentials(page=page, per_page=per_page)
    return PaginatedResp(
        items=[CredentialResp.model_validate(c.__dict__) for c in creds],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("/credentials", response_model=CredentialResp, status_code=status.HTTP_201_CREATED)
def create_credential(
    data: CredentialCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
    _current_user=Depends(get_current_user),
) -> CredentialResp:
    encrypted = security_service.encrypt_value(data.data)
    cred = pipeline_service.create_credential(name=data.name, credential_type=data.type, encrypted_data=encrypted)
    return CredentialResp.model_validate(cred.__dict__)


@router.put("/credentials/{credential_id}", response_model=CredentialResp)
def update_credential(
    credential_id: str,
    data: CredentialUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
    _current_user=Depends(get_current_user),
) -> CredentialResp:
    encrypted = security_service.encrypt_value(data.data) if data.data else None
    cred = pipeline_service.update_credential(credential_id, name=data.name, encrypted_data=encrypted)
    return CredentialResp.model_validate(cred.__dict__)


@router.delete("/credentials/{credential_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_credential(
    credential_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_credential(credential_id)


# =============================================================================
# Pipeline Templates
# =============================================================================


def _template_resp(tmpl: PipelineTemplate, latest_version: int | None) -> PipelineTemplateResp:
    data = tmpl.__dict__.copy()
    data["stages"] = [_stage_dto(s) for s in tmpl.stages]
    data["variable_declarations"] = [_decl_dto(d) for d in tmpl.variable_declarations]
    data["latest_snapshot_version"] = latest_version
    return PipelineTemplateResp.model_validate(data)


@router.get("/templates", response_model=PaginatedResp[PipelineTemplateResp])
def list_templates(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineTemplateResp]:
    items_with_version, total = pipeline_service.list_templates_with_latest_version(page=page, per_page=per_page)
    return PaginatedResp(
        items=[_template_resp(t, v) for t, v in items_with_version],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("/templates", response_model=PipelineTemplateResp, status_code=status.HTTP_201_CREATED)
def create_template(
    data: PipelineTemplateCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    stages = [StageDefinition(**s.model_dump()) for s in data.stages]
    decls = [VariableDeclaration(**d.model_dump()) for d in data.variable_declarations]
    tmpl = pipeline_service.create_template(
        name=data.name, stages=stages, description=data.description, variable_declarations=decls
    )
    return _template_resp(tmpl, latest_version=1)


@router.get("/templates/{template_id}", response_model=PipelineTemplateResp)
def get_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    tmpl = pipeline_service.get_template(template_id)
    latest = pipeline_service.get_template_latest_version(template_id)
    return _template_resp(tmpl, latest)


@router.put("/templates/{template_id}", response_model=PipelineTemplateResp)
def update_template(
    template_id: str,
    data: PipelineTemplateUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    stages = [StageDefinition(**s.model_dump()) for s in data.stages] if data.stages is not None else None
    decls = (
        [VariableDeclaration(**d.model_dump()) for d in data.variable_declarations]
        if data.variable_declarations is not None
        else None
    )
    tmpl = pipeline_service.update_template(
        template_id=template_id,
        name=data.name,
        description=data.description,
        stages=stages,
        variable_declarations=decls,
    )
    latest = pipeline_service.get_template_latest_version(template_id)
    return _template_resp(tmpl, latest)


@router.delete("/templates/{template_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_template(template_id)


@router.get("/templates/{template_id}/snapshots", response_model=list[PipelineSnapshotListItemResp])
def list_template_snapshots(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[PipelineSnapshotListItemResp]:
    snapshots = pipeline_service.list_template_snapshots(template_id)
    return [PipelineSnapshotListItemResp.model_validate(s.__dict__) for s in snapshots]


@router.get("/snapshots/{snapshot_id}", response_model=PipelineSnapshotResp)
def get_snapshot(
    snapshot_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineSnapshotResp:
    snapshot = pipeline_service.get_snapshot(snapshot_id)
    return PipelineSnapshotResp(
        id=snapshot.id,
        template_id=snapshot.template_id,
        version=snapshot.version,
        stages_snapshot=[_stage_dto(s) for s in snapshot.stages_snapshot],
        variable_declarations_snapshot=[_decl_dto(d) for d in snapshot.variable_declarations_snapshot],
        created_at=snapshot.created_at,
    )


# =============================================================================
# Projects
# =============================================================================


@router.get("/projects", response_model=PaginatedResp[ProjectResp])
def list_projects(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[ProjectResp]:
    projects, total = pipeline_service.list_projects(page=page, per_page=per_page)
    return PaginatedResp(
        items=[ProjectResp.model_validate(p.__dict__) for p in projects],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("/projects", response_model=ProjectResp, status_code=status.HTTP_201_CREATED)
def create_project(
    data: ProjectCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    project = pipeline_service.create_project(
        name=data.name,
        repository_url=data.repository_url,
        pipeline_snapshot_id=data.pipeline_snapshot_id,
        git_credential_id=data.git_credential_id,
        variable_overrides=data.variable_overrides,
        default_branch=data.default_branch,
    )
    return ProjectResp.model_validate(project.__dict__)


@router.get("/projects/{project_id}", response_model=ProjectResp)
def get_project(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    return ProjectResp.model_validate(pipeline_service.get_project(project_id).__dict__)


@router.put("/projects/{project_id}", response_model=ProjectResp)
def update_project(
    project_id: str,
    data: ProjectUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    project = pipeline_service.update_project(
        project_id=project_id,
        name=data.name,
        repository_url=data.repository_url,
        variable_overrides=data.variable_overrides,
        pipeline_snapshot_id=data.pipeline_snapshot_id,
        git_credential_id=data.git_credential_id,
        default_branch=data.default_branch,
    )
    return ProjectResp.model_validate(project.__dict__)


@router.delete("/projects/{project_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_project(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_project(project_id)


# =============================================================================
# Pipeline Runs
# =============================================================================


@router.get("/runs", response_model=PaginatedResp[PipelineRunResp])
def list_all_runs(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
    project_id: Annotated[str | None, Query()] = None,
) -> PaginatedResp[PipelineRunResp]:
    runs, total = pipeline_service.list_runs(project_id=project_id, page=page, per_page=per_page)
    return _paginated_runs(runs, total, page, per_page)


@router.get("/projects/{project_id}/runs", response_model=PaginatedResp[PipelineRunResp])
def list_runs(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineRunResp]:
    runs, total = pipeline_service.list_runs(project_id=project_id, page=page, per_page=per_page)
    return _paginated_runs(runs, total, page, per_page)


@router.post("/projects/{project_id}/runs", response_model=PipelineRunResp, status_code=status.HTTP_201_CREATED)
async def trigger_pipeline(
    project_id: str,
    data: TriggerPipelineReq,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    run, project, merged_vars = pipeline_service.create_run(
        project_id=project_id,
        trigger=PipelineRunTrigger.MANUAL,
        trigger_ref=data.trigger_ref,
        runtime_variables=data.variables,
    )
    background_tasks.add_task(pipeline_service.execute_run, run, project, merged_vars)
    return PipelineRunResp.model_validate(run.__dict__)


@router.post("/projects/{project_id}/trigger", response_model=PipelineRunResp, status_code=status.HTTP_201_CREATED)
async def trigger_pipeline_shortcut(
    project_id: str,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    data: TriggerPipelineReq | None = None,
) -> PipelineRunResp:
    req = data or TriggerPipelineReq()
    run, project, merged_vars = pipeline_service.create_run(
        project_id=project_id,
        trigger=PipelineRunTrigger.MANUAL,
        trigger_ref=req.trigger_ref,
        runtime_variables=req.variables,
    )
    background_tasks.add_task(pipeline_service.execute_run, run, project, merged_vars)
    return PipelineRunResp.model_validate(run.__dict__)


@router.get("/runs/{run_id}", response_model=PipelineRunResp)
def get_run(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    return PipelineRunResp.model_validate(pipeline_service.get_run(run_id).__dict__)


@router.get("/runs/{run_id}/artifacts", response_model=list[ArtifactResp])
def list_artifacts(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[ArtifactResp]:
    return [ArtifactResp.model_validate(a.__dict__) for a in pipeline_service.list_artifacts(run_id)]


@router.get("/runs/{run_id}/jobs", response_model=list[JobResp])
def list_jobs(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[JobResp]:
    return [JobResp.model_validate(j.__dict__) for j in pipeline_service.list_jobs(run_id)]


@router.get("/jobs/{job_id}/logs", response_model=JobLogResp | None)
def get_job_logs(
    job_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> JobLogResp | None:
    log = pipeline_service.get_job_log(job_id)
    return JobLogResp.model_validate(log.__dict__) if log else None


@router.post("/runs/{run_id}/cancel", response_model=PipelineRunResp)
def cancel_pipeline(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    return PipelineRunResp.model_validate(pipeline_service.cancel_run(run_id).__dict__)


@router.post("/runs/{run_id}/retry", response_model=PipelineRunResp, status_code=status.HTTP_201_CREATED)
async def retry_pipeline(
    run_id: str,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    new_run, project, variables = pipeline_service.create_retry_run(run_id)
    background_tasks.add_task(pipeline_service.execute_run, new_run, project, variables)
    return PipelineRunResp.model_validate(new_run.__dict__)


# =============================================================================
# Webhooks（无需认证）
# =============================================================================


@router.post("/webhooks/git")
async def receive_git_webhook(
    request: Request,
    background_tasks: BackgroundTasks,
    ci_webhook_service: Annotated[CIWebhookService, Depends(get_ci_webhook_service)],
    x_hub_signature_256: Annotated[str, Header(alias="X-Hub-Signature-256")] = "",
    x_gitlab_token: Annotated[str, Header(alias="X-Gitlab-Token")] = "",
) -> dict:
    payload_bytes = await request.body()
    payload = json.loads(payload_bytes)

    if x_hub_signature_256:
        result = ci_webhook_service.handle_github_webhook(
            payload_bytes=payload_bytes, payload=payload, signature=x_hub_signature_256
        )
    elif x_gitlab_token:
        result = ci_webhook_service.handle_gitlab_webhook(payload=payload, token=x_gitlab_token)
    else:
        result = ci_webhook_service.handle_github_webhook(payload_bytes=payload_bytes, payload=payload, signature="")

    pipeline_service = ci_webhook_service.pipeline_service
    for item in result.get("triggered", []):
        background_tasks.add_task(pipeline_service.execute_run, item["run"], item["project"], item["merged_vars"])

    return {
        "status": result["status"],
        "triggered": [{"project_id": t["project_id"], "run_id": t["run_id"]} for t in result.get("triggered", [])],
        "errors": result.get("errors", []),
        "reason": result.get("reason"),
    }
