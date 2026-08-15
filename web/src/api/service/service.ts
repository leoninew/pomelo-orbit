import type {
  ServiceBasicUpdateReq,
  ServiceComponentDetailResp,
  ServiceComponentResp,
  ServiceComponentOverlayUpdateReq,
  ServiceDeployReq,
  ServiceDeployResp,
  ServiceEnvUpdateReq,
  ServicePaginatedResp,
  ServicePreviewReq,
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

  getComponent(serviceId: string, componentId: string): Promise<ServiceComponentDetailResp> {
    return request.get(`/api/service/${serviceId}/component/${componentId}`);
  },

  updateComponent(
    serviceId: string,
    componentId: string,
    payload: ServiceComponentOverlayUpdateReq
  ): Promise<ServiceComponentResp> {
    return request.put(`/api/service/${serviceId}/component/${componentId}`, payload);
  },

  updateBasic(id: string, payload: ServiceBasicUpdateReq): Promise<ServiceResp> {
    return request.put(`/api/service/${id}/basic`, payload);
  },

  updateEnv(id: string, payload: ServiceEnvUpdateReq): Promise<ServiceResp> {
    return request.put(`/api/service/${id}/env`, payload);
  },

  preview(id: string, data: ServicePreviewReq): Promise<ServicePreviewResp> {
    return request.post(`/api/service/${id}/preview`, data);
  },

  deploy(id: string, payload: ServiceDeployReq): Promise<ServiceDeployResp> {
    return request.post(`/api/service/${id}/deploy`, payload);
  },
};
