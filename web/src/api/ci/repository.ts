import type { PipelineRunResp, PipelineRunTriggerReq } from '@/gen/orbit/api/v1/pipeline_run';
import type {
  RepositoryCreateReq,
  RepositoryPaginatedResp,
  RepositoryResp,
  RepositoryUpdateReq,
} from '@/gen/orbit/api/v1/repository';
import request from '@/utils/request';

export const repositoryApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<RepositoryPaginatedResp> {
    return request.get('/api/ci/repository', { params });
  },

  get(id: string): Promise<RepositoryResp> {
    return request.get(`/api/ci/repository/${id}`);
  },

  create(data: RepositoryCreateReq, params: { project_id: string }): Promise<RepositoryResp> {
    return request.post('/api/ci/repository', data, { params });
  },

  update(id: string, data: RepositoryUpdateReq): Promise<RepositoryResp> {
    return request.put(`/api/ci/repository/${id}`, data);
  },

  delete(id: string, params?: { delete_workspace?: boolean }): Promise<void> {
    return request.delete(`/api/ci/repository/${id}`, { params });
  },

  trigger(id: string, data: PipelineRunTriggerReq): Promise<PipelineRunResp> {
    return request.post(`/api/ci/repository/${id}/trigger`, data);
  },
};
