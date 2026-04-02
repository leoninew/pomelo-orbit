"""CI 模块 API 路由"""

import json
import logging
import math
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import APIRouter, BackgroundTasks, Depends, Header, Query, Request, status

from pomelo_orbit.application.ci.di import (
    get_ci_webhook_service,
    get_pipeline_service,
)
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.application.ci.webhook_service import CIWebhookService
from pomelo_orbit.domain.ci.entities import PipelineTemplate
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.di import get_security_service
from pomelo_orbit.infrastructure.security import SecurityService
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.dto.ci import (
    ArtifactResp,
    CredentialCreateReq,
    CredentialResp,
    JobLogResp,
    JobResp,
    PipelineRunResp,
    PipelineTemplateCreateReq,
    PipelineTemplateResp,
    PipelineTemplateUpdateReq,
    ProjectCreateReq,
    ProjectResp,
    ProjectUpdateReq,
    TriggerPipelineReq,
    VariableDeclarationResp,
    WebhookConfigResp,
)
from pomelo_orbit.interfaces.api.dto.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/v1/ci", tags=["ci"])


def _template_resp(tmpl: PipelineTemplate) -> PipelineTemplateResp:
    return PipelineTemplateResp(
        id=tmpl.id,
        name=tmpl.name,
        description=tmpl.description,
        content=tmpl.content,
        variable_declarations=[VariableDeclarationResp(**vd.model_dump()) for vd in (tmpl.variable_declarations or [])],
        is_builtin=tmpl.is_builtin,
        created_at=tmpl.created_at,
        updated_at=tmpl.updated_at,
    )


def _paginated_runs(runs: list, total: int, page: int, per_page: int) -> "PaginatedResp[PipelineRunResp]":
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
    """列出所有凭据（分页）"""
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
    """创建凭据（data 字段加密存储）"""
    encrypted = security_service.encrypt_value(data.data)
    cred = pipeline_service.create_credential(
        name=data.name,
        credential_type=data.type,
        encrypted_data=encrypted,
    )
    return CredentialResp.model_validate(cred.__dict__)


@router.delete("/credentials/{credential_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_credential(
    credential_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    """删除凭据"""
    pipeline_service.delete_credential(credential_id)


# =============================================================================
# Pipeline Templates
# =============================================================================


@router.get("/templates", response_model=PaginatedResp[PipelineTemplateResp])
def list_templates(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineTemplateResp]:
    """列出所有模板（分页）"""
    templates, total = pipeline_service.list_templates(page=page, per_page=per_page)
    return PaginatedResp(
        items=[_template_resp(t) for t in templates],
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
    """创建 流水线模板"""
    tmpl = pipeline_service.create_template(
        name=data.name,
        content=data.content,
        description=data.description,
        variable_declarations=data.variable_declarations,
    )
    return _template_resp(tmpl)


@router.get("/templates/{template_id}", response_model=PipelineTemplateResp)
def get_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    """获取模板详情"""
    tmpl = pipeline_service.get_template(template_id)
    return _template_resp(tmpl)


@router.put("/templates/{template_id}", response_model=PipelineTemplateResp)
def update_template(
    template_id: str,
    data: PipelineTemplateUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    """更新模板"""
    tmpl = pipeline_service.update_template(
        template_id=template_id,
        name=data.name,
        description=data.description,
        content=data.content,
        variable_declarations=data.variable_declarations,
    )
    return _template_resp(tmpl)


@router.delete("/templates/{template_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    """删除模板"""
    pipeline_service.delete_template(template_id)


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
    """列出项目（分页）"""
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
    """创建项目"""
    project = pipeline_service.create_project(
        name=data.name,
        repository_url=data.repository_url,
        pipeline_template_id=data.pipeline_template_id,
        git_credential_id=data.git_credential_id,
        variable_overrides=data.variable_overrides,
        branch_filter=data.branch_filter,
        default_branch=data.default_branch,
        enable_webhook=data.enable_webhook,
    )
    return ProjectResp.model_validate(project.__dict__)


@router.get("/projects/{project_id}", response_model=ProjectResp)
def get_project(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    """获取项目详情"""
    project = pipeline_service.get_project(project_id)
    return ProjectResp.model_validate(project.__dict__)


@router.put("/projects/{project_id}", response_model=ProjectResp)
def update_project(
    project_id: str,
    data: ProjectUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    """更新项目"""
    project = pipeline_service.update_project(
        project_id=project_id,
        name=data.name,
        repository_url=data.repository_url,
        variable_overrides=data.variable_overrides,
        pipeline_template_id=data.pipeline_template_id,
        git_credential_id=data.git_credential_id,
        branch_filter=data.branch_filter,
        default_branch=data.default_branch,
    )
    return ProjectResp.model_validate(project.__dict__)


@router.delete("/projects/{project_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_project(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    """删除项目"""
    pipeline_service.delete_project(project_id)


@router.get("/projects/{project_id}/webhook", response_model=WebhookConfigResp)
def get_webhook_config(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
    _current_user=Depends(get_current_user),
) -> WebhookConfigResp:
    """获取项目 webhook 配置"""
    api_base_url: str = settings.get("app.base_url", "")
    config = pipeline_service.get_webhook_config(project_id, api_base_url)
    return WebhookConfigResp(**config)


@router.post("/projects/{project_id}/webhook/regenerate", response_model=WebhookConfigResp)
def regenerate_webhook_secret(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
    _current_user=Depends(get_current_user),
) -> WebhookConfigResp:
    """重新生成 webhook secret"""
    pipeline_service.regenerate_webhook_secret(project_id)
    api_base_url: str = settings.get("app.base_url", "")
    config = pipeline_service.get_webhook_config(project_id, api_base_url)
    return WebhookConfigResp(**config)


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
    """列出所有 pipeline runs（可按项目过滤）"""
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
    """列出项目的 pipeline runs"""
    runs, total = pipeline_service.list_runs(project_id=project_id, page=page, per_page=per_page)
    return _paginated_runs(runs, total, page, per_page)


@router.post(
    "/projects/{project_id}/runs",
    response_model=PipelineRunResp,
    status_code=status.HTTP_201_CREATED,
)
async def trigger_pipeline(
    project_id: str,
    data: TriggerPipelineReq,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    """手动触发 pipeline"""
    run, project, merged_vars = pipeline_service.create_run(
        project_id=project_id,
        trigger=PipelineRunTrigger.MANUAL,
        trigger_ref=data.trigger_ref,
        runtime_variables=data.variables,
    )
    background_tasks.add_task(pipeline_service.execute_run, run, project, merged_vars)
    return PipelineRunResp.model_validate(run.__dict__)


@router.post(
    "/projects/{project_id}/trigger",
    response_model=PipelineRunResp,
    status_code=status.HTTP_201_CREATED,
)
async def trigger_pipeline_shortcut(
    project_id: str,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    data: TriggerPipelineReq | None = None,
) -> PipelineRunResp:
    """手动触发 pipeline（快捷路由，body 可选）"""
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
    """获取 pipeline run 详情"""
    run = pipeline_service.get_run(run_id)
    return PipelineRunResp.model_validate(run.__dict__)


@router.get("/runs/{run_id}/artifacts", response_model=list[ArtifactResp])
def list_artifacts(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[ArtifactResp]:
    """列出 pipeline run 的所有制品"""
    artifacts = pipeline_service.list_artifacts(run_id)
    return [ArtifactResp.model_validate(a.__dict__) for a in artifacts]


@router.get("/runs/{run_id}/jobs", response_model=list[JobResp])
def list_jobs(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[JobResp]:
    """列出 pipeline run 的所有 jobs"""
    jobs = pipeline_service.list_jobs(run_id)
    return [JobResp.model_validate(j.__dict__) for j in jobs]


@router.get("/jobs/{job_id}/logs", response_model=JobLogResp | None)
def get_job_logs(
    job_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> JobLogResp | None:
    """获取 job 日志"""
    job_log = pipeline_service.get_job_log(job_id)
    if not job_log:
        return None
    return JobLogResp.model_validate(job_log.__dict__)


@router.post("/runs/{run_id}/cancel", response_model=PipelineRunResp)
def cancel_pipeline(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    """取消 pipeline run（waiting 或 running 状态）"""
    run = pipeline_service.cancel_run(run_id)
    return PipelineRunResp.model_validate(run.__dict__)


@router.post("/runs/{run_id}/retry", response_model=PipelineRunResp, status_code=status.HTTP_201_CREATED)
async def retry_pipeline(
    run_id: str,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    """重试失败的 pipeline run"""
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
    """
    接收 Git webhook（GitHub / GitLab）

    - GitHub: 通过 X-Hub-Signature-256 验证签名
    - GitLab: 通过 X-Gitlab-Token 验证
    """
    payload_bytes = await request.body()
    payload = json.loads(payload_bytes)

    if x_hub_signature_256:
        result = ci_webhook_service.handle_github_webhook(
            payload_bytes=payload_bytes,
            payload=payload,
            signature=x_hub_signature_256,
        )
    elif x_gitlab_token:
        result = ci_webhook_service.handle_gitlab_webhook(
            payload=payload,
            token=x_gitlab_token,
        )
    else:
        result = ci_webhook_service.handle_github_webhook(
            payload_bytes=payload_bytes,
            payload=payload,
            signature="",
        )

    # 调度后台执行
    pipeline_service = ci_webhook_service.pipeline_service
    for item in result.get("triggered", []):
        background_tasks.add_task(pipeline_service.execute_run, item["run"], item["project"], item["merged_vars"])

    # 返回时只保留可序列化的字段
    return {
        "status": result["status"],
        "triggered": [{"project_id": t["project_id"], "run_id": t["run_id"]} for t in result.get("triggered", [])],
        "errors": result.get("errors", []),
        "reason": result.get("reason"),
    }
