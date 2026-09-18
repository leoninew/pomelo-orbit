import type {
  GatewayPaginatedResp,
  GatewayResp,
  GatewayUpdateReq,
} from '@/gen/proto/orbit/v1/gateway/gateway';
import request from '@/utils/request';

export const gatewayApi = {
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      search?: string;
    }
  ): Promise<GatewayPaginatedResp> {
    return request.get('/api/gateway', { params: { project_id: projectId, ...params } });
  },

  get(projectId: string, id: string): Promise<GatewayResp> {
    return request.get(`/api/gateway/${id}`, { params: { project_id: projectId } });
  },

  update(projectId: string, id: string, data: GatewayUpdateReq): Promise<GatewayResp> {
    return request.put(`/api/gateway/${id}`, data, { params: { project_id: projectId } });
  },

  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/gateway/${id}`, { params: { project_id: projectId } });
  },
};
