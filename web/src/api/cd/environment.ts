import type {
  EnvironmentCreateReq,
  EnvironmentPaginatedResp,
  EnvironmentResp,
  EnvironmentUpdateReq,
} from '@/gen/proto/orbit/v1/environment';
import request from '@/utils/request';

export const environmentApi = {
  list(params: {
    project_id: string;
    page?: number;
    per_page?: number;
    search?: string;
  }): Promise<EnvironmentPaginatedResp> {
    return request.get('/api/cd/environment', { params });
  },

  get(id: string): Promise<EnvironmentResp> {
    return request.get(`/api/cd/environment/${id}`);
  },

  create(data: EnvironmentCreateReq, params?: { project_id?: string }): Promise<EnvironmentResp> {
    return request.post('/api/cd/environment', data, { params });
  },

  update(id: string, data: EnvironmentUpdateReq): Promise<EnvironmentResp> {
    return request.put(`/api/cd/environment/${id}`, data);
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/cd/environment/${id}`);
  },
};
