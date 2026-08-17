import { afterEach, describe, expect, it, vi } from 'vitest';
import { buildApiUrl, getAppVersion, getPublicUrl } from './config';

describe('runtime config', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it('uses same-origin API paths by default', () => {
    expect(getPublicUrl()).toBe('');
    expect(buildApiUrl('/api/health')).toBe('/api/health');
  });

  it('uses runtime publicUrl before Vite env', () => {
    vi.stubEnv('VITE_PUBLIC_URL', 'https://vite-api.preflite.cn');
    vi.stubGlobal('window', { __CONFIG__: { publicUrl: 'https://runtime-api.preflite.cn/' } });

    expect(getPublicUrl()).toBe('https://runtime-api.preflite.cn');
    expect(buildApiUrl('/api/health')).toBe('https://runtime-api.preflite.cn/api/health');
  });

  it('uses Vite public URL when runtime config is absent', () => {
    vi.stubEnv('VITE_PUBLIC_URL', 'https://vite-api.preflite.cn/');

    expect(getPublicUrl()).toBe('https://vite-api.preflite.cn');
    expect(buildApiUrl('/api/health')).toBe('https://vite-api.preflite.cn/api/health');
  });

  it('uses the runtime app version when present', () => {
    vi.stubGlobal('window', { __CONFIG__: { appVersion: '0.251.0' } });

    expect(getAppVersion()).toBe('0.251.0');
  });

  it('rejects paths outside configured API namespaces', () => {
    expect(() => buildApiUrl('api/health')).toThrow('API path must start with one of');
    expect(() => buildApiUrl('/apix/health')).toThrow('API path must start with one of');
  });
});
