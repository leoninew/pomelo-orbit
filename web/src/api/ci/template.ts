import type { ListResp, PaginatedResp } from '@/types/common';
import type { PipelineSnapshot } from '@/types/ci/snapshot';
import type {
  PipelineTemplate,
  PipelineTemplateCreateReq,
  StageOrchestrationReq,
  PipelineTemplateUpdateReq,
  VariableDeclaration,
  VariableDeclarationReq,
} from '@/types/ci/template';
import request from '@/utils/request';

export const pipelineTemplateApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<PaginatedResp<PipelineTemplate>> {
    return request.get('/api/ci/template', { params });
  },

  get(id: string): Promise<PipelineTemplate> {
    return request.get(`/api/ci/template/${id}`);
  },

  create(
    data: PipelineTemplateCreateReq,
    params: { project_id: string }
  ): Promise<PipelineTemplate> {
    return request.post('/api/ci/template', data, { params });
  },

  update(id: string, data: PipelineTemplateUpdateReq): Promise<PipelineTemplate> {
    return request.put(`/api/ci/template/${id}`, data);
  },

  resolveVariables(
    data: {
      orchestration: StageOrchestrationReq[];
      variable_declarations?: VariableDeclarationReq[];
    },
    params: { project_id: string }
  ): Promise<ListResp<VariableDeclaration>> {
    return request.post('/api/ci/template/resolve-variables', data, { params });
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/ci/template/${id}`);
  },

  duplicate(id: string): Promise<PipelineTemplate> {
    return request.post(`/api/ci/template/${id}/duplicate`, {});
  },

  getSnapshot(snapshotId: string): Promise<PipelineSnapshot> {
    return request.get(`/api/ci/snapshot/${snapshotId}`);
  },
};
