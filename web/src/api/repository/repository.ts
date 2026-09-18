import type {
  RepositoryCreateReq,
  RepositoryPaginatedResp,
  RepositoryResp,
  RepositoryUpdateReq,
} from '@/gen/proto/orbit/v1/repository/repository';
import request from '@/utils/request';

export const repositoryApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      search?: string;
    }
  ): Promise<RepositoryPaginatedResp> {
    return request.get('/api/repository', { params: { project_id: projectId, ...params } });
  },

  get(projectId: string, id: string): Promise<RepositoryResp> {
    return request.get(`/api/repository/${id}`, { params: { project_id: projectId } });
  },

  create(projectId: string, data: RepositoryCreateReq): Promise<RepositoryResp> {
    return request.post('/api/repository', data, { params: { project_id: projectId } });
  },

  update(projectId: string, id: string, data: RepositoryUpdateReq): Promise<RepositoryResp> {
    return request.put(`/api/repository/${id}`, data, { params: { project_id: projectId } });
  },

  delete(projectId: string, id: string, params?: { delete_workspace?: boolean }): Promise<void> {
    return request.delete(`/api/repository/${id}`, {
      params: { project_id: projectId, ...params },
    });
  },
};
