import type { PipelineSnapshotResp } from '@/gen/proto/orbit/api/v1/snapshot';
import type {
  PipelineTemplateCreateReq,
  PipelineTemplatePaginatedResp,
  PipelineTemplateResp,
  PipelineTemplateUpdateReq,
  StageOrchestrationReq,
  TemplateVariableResolveResp,
} from '@/gen/proto/orbit/api/v1/template';
import type { VariableDeclarationReq } from '@/gen/proto/orbit/api/v1/common';
import request from '@/utils/request';

export const pipelineTemplateApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<PipelineTemplatePaginatedResp> {
    return request.get('/api/ci/template', { params });
  },

  get(id: string): Promise<PipelineTemplateResp> {
    return request.get(`/api/ci/template/${id}`);
  },

  create(
    data: PipelineTemplateCreateReq,
    params: { project_id: string }
  ): Promise<PipelineTemplateResp> {
    return request.post('/api/ci/template', data, { params });
  },

  update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplateResp> {
    return request.put(`/api/ci/template/${id}`, data);
  },

  resolveVariables(
    data: {
      orchestration: StageOrchestrationReq[];
      variable_declarations?: VariableDeclarationReq[];
    },
    params: { project_id: string }
  ): Promise<TemplateVariableResolveResp> {
    return request.post('/api/ci/template/resolve-variables', data, { params });
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/ci/template/${id}`);
  },

  duplicate(id: string): Promise<PipelineTemplateResp> {
    return request.post(`/api/ci/template/${id}/duplicate`, {});
  },

  getSnapshot(snapshotId: string): Promise<PipelineSnapshotResp> {
    return request.get(`/api/ci/snapshot/${snapshotId}`);
  },
};
