import type { ServicePaginatedResp, ServiceResp } from '@/gen/proto/orbit/v1/service/service';
import request from '@/utils/request';

export const serviceApi = {
  list(params: {
    project_id: string;
    page?: number;
    per_page?: number;
    application_id?: string;
    environment_id?: string;
    status?: string;
    search?: string;
  }): Promise<ServicePaginatedResp> {
    return request.get('/api/service', { params });
  },

  get(id: string): Promise<ServiceResp> {
    return request.get(`/api/service/${id}`);
  },
};
