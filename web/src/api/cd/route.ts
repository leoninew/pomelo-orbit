import type {
  RouteCreateReq,
  RouteDisableReq,
  RouteDisableResp,
  RouteEnableReq,
  RouteEnableResp,
  RouteLetsEncryptEnableReq,
  RouteMkcertEnableReq,
  RoutePaginatedResp,
  RouteResp,
  RouteSyncReq,
  RouteSyncResp,
  RouteUpdateReq,
} from '@/gen/proto/orbit/route';
import request from '@/utils/request';

export const routeApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id: string;
  }): Promise<RoutePaginatedResp> {
    return request.get('/api/cd/route', { params });
  },

  get(id: string): Promise<RouteResp> {
    return request.get(`/api/cd/route/${id}`);
  },

  create(data: RouteCreateReq, params: { project_id: string }): Promise<RouteResp> {
    return request.post('/api/cd/route', data, { params });
  },

  update(id: string, data: RouteUpdateReq): Promise<RouteResp> {
    return request.put(`/api/cd/route/${id}`, data);
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/cd/route/${id}`);
  },

  enable(id: string, data: RouteEnableReq): Promise<RouteEnableResp> {
    return request.post(`/api/cd/route/${id}/enable`, data);
  },

  disable(id: string, data: RouteDisableReq): Promise<RouteDisableResp> {
    return request.post(`/api/cd/route/${id}/disable`, data);
  },

  sync(data: RouteSyncReq, params: { project_id: string }): Promise<RouteSyncResp> {
    return request.post('/api/cd/route/sync', data, { params });
  },

  uploadCert(id: string, certFile: File): Promise<RouteResp> {
    const formData = new FormData();
    formData.append('pem', certFile);
    return request.post(`/api/cd/route/${id}/cert`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },

  disableHttps(id: string): Promise<RouteResp> {
    return request.delete(`/api/cd/route/${id}/https`);
  },

  enableLetsencrypt(id: string, data: RouteLetsEncryptEnableReq): Promise<RouteResp> {
    return request.post(`/api/cd/route/${id}/letsencrypt`, data);
  },

  enableMkcert(id: string, data: RouteMkcertEnableReq): Promise<RouteResp> {
    return request.post(`/api/cd/route/${id}/mkcert`, data);
  },
};
