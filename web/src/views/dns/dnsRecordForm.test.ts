import { describe, expect, it } from 'vitest';
import { validateDNSCreateForm, validateDNSRecordForm } from './dnsRecordForm';

describe('DNS A record editing', () => {
  it('accepts IPv4 with automatic or numeric TTL', () => {
    expect(validateDNSRecordForm({ content: ' 192.0.2.1 ', ttl: '1', proxied: false })).toEqual({});
    expect(validateDNSRecordForm({ content: '192.0.2.1', ttl: '300', proxied: false })).toEqual({});
  });
  it.each(['2001:db8::1', '192.0.2.256', '192.000.2.1', '192.0.2'])(
    'rejects invalid A record address %s',
    (content) => {
      expect(validateDNSRecordForm({ content, ttl: '300', proxied: false }).content).toBe(
        'invalidIPv4'
      );
    }
  );
  it.each(['', '0', '29', '86401', '1.5', '3e2'])('rejects invalid TTL %s', (ttl) => {
    expect(validateDNSRecordForm({ content: '192.0.2.1', ttl, proxied: false }).ttl).toBe(
      'invalidTTL'
    );
  });
  it('uses automatic TTL while proxying', () => {
    expect(validateDNSRecordForm({ content: '192.0.2.1', ttl: '', proxied: true })).toEqual({});
  });
});

describe('DNS A record creation', () => {
  it('requires a name and validates record fields', () => {
    expect(validateDNSCreateForm({ name: ' ', content: 'bad', ttl: '1', proxied: false })).toEqual({
      name: 'invalidName',
      content: 'invalidIPv4',
    });
    expect(
      validateDNSCreateForm({ name: '@', content: '192.0.2.1', ttl: '1', proxied: false })
    ).toEqual({});
  });
});
