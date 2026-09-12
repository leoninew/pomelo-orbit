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
  RouteSyncConfirmReq,
  RouteSyncConfirmResp,
  RouteSyncPreviewReq,
  RouteSyncPreviewResp,
  RouteUpdateReq,
} from '@/gen/proto/orbit/v1/route/route';
import request, { remoteRequestConfig } from '@/utils/request';

export const routeApi = {
  list(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    project_id: string;
  }): Promise<RoutePaginatedResp> {
    return request.get('/api/route', { params });
  },

  get(id: string): Promise<RouteResp> {
    return request.get(`/api/route/${id}`);
  },

  create(data: RouteCreateReq, params: { project_id: string }): Promise<RouteResp> {
    return request.post('/api/route', data, { params });
  },

  update(id: string, data: RouteUpdateReq): Promise<RouteResp> {
    return request.put(`/api/route/${id}`, data);
  },

  delete(id: string): Promise<void> {
    return request.delete(`/api/route/${id}`);
  },

  enable(id: string, data: RouteEnableReq): Promise<RouteEnableResp> {
    return request.post(`/api/route/${id}/enable`, data);
  },

  disable(id: string, data: RouteDisableReq): Promise<RouteDisableResp> {
    return request.post(`/api/route/${id}/disable`, data);
  },

  previewSync(
    data: RouteSyncPreviewReq,
    params: { project_id: string }
  ): Promise<RouteSyncPreviewResp> {
    return request.post(
      '/api/route/sync/preview',
      data,
      remoteRequestConfig({
        params,
      })
    );
  },

  confirmSync(
    data: RouteSyncConfirmReq,
    params: { project_id: string }
  ): Promise<RouteSyncConfirmResp> {
    return request.post(
      '/api/route/sync/confirm',
      data,
      remoteRequestConfig({
        params,
      })
    );
  },

  uploadCert(id: string, certFile: File): Promise<RouteResp> {
    const formData = new FormData();
    formData.append('pem', certFile);
    return request.post(`/api/route/${id}/cert`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },

  disableHttps(id: string): Promise<RouteResp> {
    return request.delete(`/api/route/${id}/https`);
  },

  enableLetsencrypt(id: string, data: RouteLetsEncryptEnableReq): Promise<RouteResp> {
    return request.post(`/api/route/${id}/letsencrypt`, data);
  },

  enableMkcert(id: string, data: RouteMkcertEnableReq): Promise<RouteResp> {
    return request.post(`/api/route/${id}/mkcert`, data);
  },
};
