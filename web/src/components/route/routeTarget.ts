import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';

type RouteTarget = Pick<
  RouteResp,
  'service_id' | 'component_name' | 'endpoint_protocol' | 'endpoint_container_port' | 'target_url'
>;

export function routeTargetAddress(route: RouteTarget): string {
  if (!route.service_id) {
    return route.target_url;
  }
  return `${route.component_name}/${route.endpoint_protocol}${route.endpoint_container_port}`;
}
