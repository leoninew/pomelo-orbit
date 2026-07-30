import type {
  ServiceConfigReq,
  ServiceBasicUpdateReq,
  ServiceDeployReq,
  ServiceDeployResp,
  ServicePaginatedResp,
  ServicePreviewResp,
  ServiceResp,
  ServiceCreateReq,
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

  updateConfiguration(id: string, payload: ServiceConfigReq): Promise<ServiceResp> {
    return request.put(`/api/service/${id}/config`, payload);
  },

  updateBasic(id: string, payload: ServiceBasicUpdateReq): Promise<ServiceResp> {
    return request.put(`/api/service/${id}/basic`, payload);
  },

  preview(id: string): Promise<ServicePreviewResp> {
    return request.post(`/api/service/${id}/preview`, {});
  },

  deploy(id: string, payload: ServiceDeployReq): Promise<ServiceDeployResp> {
    return request.post(`/api/service/${id}/deploy`, payload);
  },
};
