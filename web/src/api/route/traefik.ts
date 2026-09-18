import type { TraefikConfigResp, TraefikRouteListResp } from '@/gen/proto/orbit/v1/route/traefik';
import request, { remoteRequestConfig } from '@/utils/request';

export const traefikRouteApi = {
  list(projectId: string): Promise<TraefikRouteListResp> {
    return request.get(
      '/api/route/traefik',
      remoteRequestConfig({ params: { project_id: projectId } })
    );
  },
  getConfig(projectId: string): Promise<TraefikConfigResp> {
    return request.get(
      '/api/route/traefik/config',
      remoteRequestConfig({ params: { project_id: projectId } })
    );
  },
};
