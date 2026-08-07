import type {
  RepositoryCreateReq,
  RepositoryPaginatedResp,
  RepositoryResp,
  RepositoryUpdateReq,
} from '@/gen/proto/orbit/v1/repository/repository';
import request from '@/utils/request';

export const repositoryApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id?: string;
  }): Promise<RepositoryPaginatedResp> {
    return request.get('/api/repository', { params });
  },

  get(id: string): Promise<RepositoryResp> {
    return request.get(`/api/repository/${id}`);
  },

  create(data: RepositoryCreateReq, params: { project_id: string }): Promise<RepositoryResp> {
    return request.post('/api/repository', data, { params });
  },

  update(id: string, data: RepositoryUpdateReq): Promise<RepositoryResp> {
    return request.put(`/api/repository/${id}`, data);
  },

  delete(id: string, params?: { delete_workspace?: boolean }): Promise<void> {
    return request.delete(`/api/repository/${id}`, { params });
  },
};
