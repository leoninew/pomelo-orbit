import type {
  GatewayCreateReq,
  GatewayPaginatedResp,
  GatewayResp,
  GatewayUpdateReq,
} from '@/gen/proto/orbit/v1/gateway';
import request from '@/utils/request';

export const gatewayApi = {
  list(params: {
    project_id: string;
    page?: number;
    per_page?: number;
    search?: string;
  }): Promise<GatewayPaginatedResp> {
    return request.get('/api/cd/gateway', { params });
  },

  get(id: string): Promise<GatewayResp> {
    return request.get(`/api/cd/gateway/${id}`);
  },

  create(data: GatewayCreateReq, params?: { project_id?: string }): Promise<GatewayResp> {
    return request.post('/api/cd/gateway', data, { params });
  },

  update(id: string, data: GatewayUpdateReq): Promise<GatewayResp> {
    return request.put(`/api/cd/gateway/${id}`, data);
  },

  delete(id: string, removeDir: boolean = false): Promise<void> {
    return request.delete(`/api/cd/gateway/${id}`, {
      params: { remove_dir: removeDir },
    });
  },
};
