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
  list(
    projectId: string,
    params?: {
      page?: number;
      per_page?: number;
      search?: string;
    }
  ): Promise<RoutePaginatedResp> {
    return request.get('/api/route', { params: { project_id: projectId, ...params } });
  },

  get(projectId: string, id: string): Promise<RouteResp> {
    return request.get(`/api/route/${id}`, { params: { project_id: projectId } });
  },

  create(projectId: string, data: RouteCreateReq): Promise<RouteResp> {
    return request.post('/api/route', data, { params: { project_id: projectId } });
  },

  update(projectId: string, id: string, data: RouteUpdateReq): Promise<RouteResp> {
    return request.put(`/api/route/${id}`, data, { params: { project_id: projectId } });
  },

  delete(projectId: string, id: string): Promise<void> {
    return request.delete(`/api/route/${id}`, { params: { project_id: projectId } });
  },

  enable(projectId: string, id: string, data: RouteEnableReq): Promise<RouteEnableResp> {
    return request.post(`/api/route/${id}/enable`, data, { params: { project_id: projectId } });
  },

  disable(projectId: string, id: string, data: RouteDisableReq): Promise<RouteDisableResp> {
    return request.post(`/api/route/${id}/disable`, data, { params: { project_id: projectId } });
  },

  previewSync(projectId: string, data: RouteSyncPreviewReq): Promise<RouteSyncPreviewResp> {
    return request.post(
      '/api/route/sync/preview',
      data,
      remoteRequestConfig({
        params: { project_id: projectId },
      })
    );
  },

  confirmSync(projectId: string, data: RouteSyncConfirmReq): Promise<RouteSyncConfirmResp> {
    return request.post(
      '/api/route/sync/confirm',
      data,
      remoteRequestConfig({
        params: { project_id: projectId },
      })
    );
  },

  uploadCert(projectId: string, id: string, certFile: File): Promise<RouteResp> {
    const formData = new FormData();
    formData.append('pem', certFile);
    return request.post(`/api/route/${id}/cert`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      params: { project_id: projectId },
    });
  },

  disableHttps(projectId: string, id: string): Promise<RouteResp> {
    return request.delete(`/api/route/${id}/https`, { params: { project_id: projectId } });
  },

  enableLetsencrypt(
    projectId: string,
    id: string,
    data: RouteLetsEncryptEnableReq
  ): Promise<RouteResp> {
    return request.post(`/api/route/${id}/letsencrypt`, data, {
      params: { project_id: projectId },
    });
  },

  enableMkcert(projectId: string, id: string, data: RouteMkcertEnableReq): Promise<RouteResp> {
    return request.post(`/api/route/${id}/mkcert`, data, { params: { project_id: projectId } });
  },
};
