import type { TraefikConfigResp, TraefikRouteListResp } from '@/gen/proto/orbit/v1/route/traefik';
import request from '@/utils/request';

export const traefikRouteApi = {
  list(params: { project_id: string }): Promise<TraefikRouteListResp> {
    return request.get('/api/route/traefik', { params });
  },
  getConfig(params: { project_id: string }): Promise<TraefikConfigResp> {
    return request.get('/api/route/traefik/config', { params });
  },
};
