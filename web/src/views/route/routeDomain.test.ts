import { describe, expect, it } from 'vitest';
import { domainAfterCodeChange, isValidRouteCode, routeDomainSuffix } from './routeDomain';

describe('route domain from code', () => {
  it.each([
    {
      scenario: 'prefers the external domain when both domains are configured',
      externalDomain: 'example.com',
      internalDomain: 'internal.test',
      createdDomain: 'api.example.com',
      updatedDomain: 'web.example.com',
    },
    {
      scenario: 'uses the internal domain when the external domain is empty',
      externalDomain: '',
      internalDomain: 'internal.test',
      createdDomain: 'api.internal.test',
      updatedDomain: 'web.internal.test',
    },
  ])('$scenario', ({ externalDomain, internalDomain, createdDomain, updatedDomain }) => {
    const suffix = routeDomainSuffix(externalDomain, internalDomain);
    expect(domainAfterCodeChange('', 'api', '', suffix)).toBe(createdDomain);
    expect(domainAfterCodeChange('api', 'web', createdDomain, suffix)).toBe(updatedDomain);
    expect(domainAfterCodeChange('api', 'web', 'custom.example.net', suffix)).toBe(
      'custom.example.net'
    );
    expect(domainAfterCodeChange('api', '', createdDomain, suffix)).toBe('');
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
