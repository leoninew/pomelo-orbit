import type {
  RouteCreateReq,
  RoutePaginatedResp,
  RouteResp,
  RouteSyncConfirmReq,
  RouteSyncConfirmResp,
  RouteSyncPreviewReq,
  RouteSyncPreviewResp,
  RouteUpdateReq,
} from '@/gen/proto/orbit/v1/route/route';
import request from '@/utils/request';

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

  uploadCert(projectId: string, id: string, file: File): Promise<RouteResp> {
    const data = new FormData();
    data.append('pem', file);
    return request.post(`/api/route/${id}/cert`, data, { params: { project_id: projectId } });
  },

  disableHTTPS(projectId: string, id: string): Promise<RouteResp> {
    return request.delete(`/api/route/${id}/https`, { params: { project_id: projectId } });
  },

  enableLetsEncrypt(projectId: string, id: string, challenge: 'http' | 'dns'): Promise<RouteResp> {
    return request.post(
      `/api/route/${id}/letsencrypt`,
      { challenge },
      { params: { project_id: projectId } }
    );
  },

  enableMkcert(projectId: string, id: string): Promise<RouteResp> {
    return request.post(`/api/route/${id}/mkcert`, {}, { params: { project_id: projectId } });
  },

  previewSync(projectId: string, data: RouteSyncPreviewReq): Promise<RouteSyncPreviewResp> {
    return request.post('/api/route/sync/preview', data, {
      params: { project_id: projectId },
      timeout: 0,
    });
  },

  confirmSync(projectId: string, data: RouteSyncConfirmReq): Promise<RouteSyncConfirmResp> {
    return request.post('/api/route/sync/confirm', data, {
      params: { project_id: projectId },
      timeout: 0,
    });
  },
};
