import type {
  ServicePaginatedResp,
  ServiceResp,
  ServiceCreateReq,
  ServiceRuntimeConfigResp,
} from '@/gen/proto/orbit/v1/service/service';
import request from '@/utils/request';

export const serviceApi = {
  list(params: {
    project_id: string;
    page?: number;
    per_page?: number;
    application_id?: string;
    status?: string;
    search?: string;
  }): Promise<ServicePaginatedResp> {
    return request.get('/api/service', { params });
  },

  get(id: string): Promise<ServiceResp> {
    return request.get(`/api/service/${id}`);
  },

  create(payload: ServiceCreateReq): Promise<ServiceResp> {
    return request.post('/api/service', payload);
  },

  remove(id: string): Promise<void> {
    return request.delete(`/api/service/${id}`);
  },

  getRuntimeConfig(id: string): Promise<ServiceRuntimeConfigResp> {
    return request.get(`/api/service/${id}/runtime-config`);
  },

  updateRuntimeConfig(
    id: string,
    runtime_config: Record<string, string>
  ): Promise<ServiceRuntimeConfigResp> {
    return request.put(`/api/service/${id}/runtime-config`, { runtime_config });
  },
};
