import type { ServiceComponentDeclaredEndpoint } from '@/gen/proto/orbit/v1/service/service';

export function publishedServiceComponentEndpoints(
  endpoints: ServiceComponentDeclaredEndpoint[]
): ServiceComponentDeclaredEndpoint[] {
  return endpoints.filter((endpoint) => endpoint.mode !== 'internal');
}
