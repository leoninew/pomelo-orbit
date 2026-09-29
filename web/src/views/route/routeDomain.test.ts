import { describe, expect, it } from 'vitest';
import { domainAfterCodeChange, isValidRouteCode } from './routeDomain';

describe('route domain from code', () => {
  it('fills and updates a generated domain while retaining a manual domain', () => {
    expect(domainAfterCodeChange('', 'api', '', 'example.com')).toBe('api.example.com');
    expect(domainAfterCodeChange('api', 'web', 'api.example.com', 'example.com')).toBe(
      'web.example.com'
    );
    expect(domainAfterCodeChange('api', 'web', 'custom.example.net', 'example.com')).toBe(
      'custom.example.net'
    );
    expect(domainAfterCodeChange('api', 'web', 'api.internal.test', '')).toBe('api.internal.test');
    expect(domainAfterCodeChange('', 'api', '', '')).toBe('');
  });

  it('accepts only letter-leading DNS labels of up to 32 characters', () => {
    expect(isValidRouteCode('a'.repeat(32))).toBe(true);
    expect(isValidRouteCode('api1')).toBe(true);
    expect(isValidRouteCode('1-api')).toBe(false);
    expect(isValidRouteCode('123')).toBe(false);
    expect(domainAfterCodeChange('', '123', '', 'example.com')).toBe('');
    expect(isValidRouteCode('a'.repeat(33))).toBe(false);
    expect(isValidRouteCode('bad_name')).toBe(false);
    expect(isValidRouteCode('bad.name')).toBe(false);
    expect(isValidRouteCode('bad-')).toBe(false);
  });
});
