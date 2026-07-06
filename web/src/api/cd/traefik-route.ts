import type {
  TraefikConfigResp,
  TraefikRouteListResp,
} from '@/gen/orbit/api/v1/traefik';
import request from '@/utils/request';

export const traefikRouteApi = {
  list(params: { project_id: string }): Promise<TraefikRouteListResp> {
    return request.get('/api/cd/traefik-route', { params });
  },
  getConfig(params: { project_id: string }): Promise<TraefikConfigResp> {
    return request.get('/api/cd/traefik-route/config', { params });
  },
};
