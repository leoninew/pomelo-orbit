import { describe, expect, it } from 'vitest';
import type { ServiceComponentDeclaredEndpoint } from '@/gen/proto/orbit/v1/service/service';
import { publishedServiceComponentEndpoints } from './serviceComponentPorts';

function endpoint(
  values: Partial<ServiceComponentDeclaredEndpoint>
): ServiceComponentDeclaredEndpoint {
  return {
    protocol: 'tcp',
    container_port: 8080,
    mode: 'internal',
    ...values,
  };
}

describe('service component ports', () => {
  it('keeps published endpoint protocol and container port while omitting internal-only endpoints', () => {
    const gateway = endpoint({ protocol: 'http', container_port: 80, mode: 'gateway' });
    const published = publishedServiceComponentEndpoints([endpoint({ mode: 'internal' }), gateway]);
    expect(published).toEqual([gateway]);
    expect(published[0]).toMatchObject({ protocol: 'http', container_port: 80 });
  });
});
