import { describe, expect, it } from 'vitest';
import { routeTargetAddress } from './routeTarget';

describe('routeTargetAddress', () => {
  it.each([
    ['http', 8080, 'api/http8080'],
    ['tcp', 6379, 'api/tcp6379'],
  ])('formats the declared %s endpoint separately from the service', (protocol, port, expected) => {
    expect(
      routeTargetAddress({
        service_id: 'service-1',
        component_name: 'api',
        endpoint_protocol: String(protocol),
        endpoint_container_port: Number(port),
        target_url: 'saved-service/old-component/http80',
      })
    ).toBe(expected);
  });

  it('preserves a custom HTTP target URL', () => {
    expect(routeTargetAddress({ target_url: 'https://example.com:8443' })).toBe(
      'https://example.com:8443'
    );
  });
});
