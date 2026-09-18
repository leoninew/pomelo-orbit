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
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      application_id?: string;
      status?: string;
      search?: string;
    }
  ): Promise<ServicePaginatedResp> {
    return request.get('/api/service', { params: { project_id: projectId, ...params } });
  },

  get(projectId: string, id: string): Promise<ServiceResp> {
    return request.get(`/api/service/${id}`, { params: { project_id: projectId } });
  },

  create(projectId: string, payload: ServiceCreateReq): Promise<ServiceResp> {
    return request.post('/api/service', payload, { params: { project_id: projectId } });
  },

  remove(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/service/${id}`, { params: { project_id: projectId } });
  },

  getComponent(
    projectId: string,
    serviceId: string,
    componentId: string
  ): Promise<ServiceComponentDetailResp> {
    return request.get(`/api/service/${serviceId}/component/${componentId}`, {
      params: { project_id: projectId },
    });
  },

  updateComponent(
    projectId: string,
    serviceId: string,
    componentId: string,
    payload: ServiceComponentOverlayUpdateReq
  ): Promise<ServiceComponentResp> {
    return request.put(`/api/service/${serviceId}/component/${componentId}`, payload, {
      params: { project_id: projectId },
    });
  },

  updateBasic(projectId: string, id: string, payload: ServiceBasicUpdateReq): Promise<ServiceResp> {
    return request.put(`/api/service/${id}/basic`, payload, { params: { project_id: projectId } });
  },

  updateEnv(projectId: string, id: string, payload: ServiceEnvUpdateReq): Promise<ServiceResp> {
    return request.put(`/api/service/${id}/env`, payload, { params: { project_id: projectId } });
  },

  preview(projectId: string, id: string, data: ServicePreviewReq): Promise<ServicePreviewResp> {
    return request.post(`/api/service/${id}/preview`, data, { params: { project_id: projectId } });
  },

  deploy(projectId: string, id: string, payload: ServiceDeployReq): Promise<ServiceDeployResp> {
    return request.post(`/api/service/${id}/deploy`, payload, {
      params: { project_id: projectId },
    });
  },
};
