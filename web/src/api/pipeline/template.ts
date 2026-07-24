import type { PipelineSnapshotResp } from '@/gen/proto/orbit/v1/pipeline/snapshot';
import type {
  PipelineTemplateCreateReq,
  PipelineTemplateDuplicateReq,
  PipelineTemplatePaginatedResp,
  PipelineTemplateResp,
  PipelineTemplateUpdateReq,
  TemplateVariableResolveReq,
  TemplateVariableResolveResp,
} from '@/gen/proto/orbit/v1/pipeline/template';
import request from '@/utils/request';

export const pipelineTemplateApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<PipelineTemplatePaginatedResp> {
    return request.get('/api/pipeline/template', { params });
  },

  get(id: string): Promise<PipelineTemplateResp> {
    return request.get(`/api/pipeline/template/${id}`);
  },

  create(
    data: PipelineTemplateCreateReq,
    params: { project_id: string }
  ): Promise<PipelineTemplateResp> {
    return request.post('/api/pipeline/template', data, { params });
  },

  update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplateResp> {
    return request.put(`/api/pipeline/template/${id}`, data);
  },

  resolveVariables(
    data: TemplateVariableResolveReq,
    params: { project_id: string }
  ): Promise<TemplateVariableResolveResp> {
    return request.post('/api/pipeline/template/resolve-variables', data, { params });
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/pipeline/template/${id}`);
  },

  duplicate(id: string, data: PipelineTemplateDuplicateReq): Promise<PipelineTemplateResp> {
    return request.post(`/api/pipeline/template/${id}/duplicate`, data);
  },

  getSnapshot(snapshotId: string): Promise<PipelineSnapshotResp> {
    return request.get(`/api/pipeline/snapshot/${snapshotId}`);
  },
};
