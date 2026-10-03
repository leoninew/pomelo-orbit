import type { InternalAxiosRequestConfig } from 'axios';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: null }),
}));

vi.mock('@/utils/handle-unauthorized', () => ({
  handleUnauthorized: vi.fn(),
}));

import { routeApi } from '@/api/route/route';
import request, { DEFAULT_REQUEST_TIMEOUT } from '@/utils/request';

const originalAdapter = request.defaults.adapter;

afterEach(() => {
  request.defaults.adapter = originalAdapter;
});

describe('Route synchronization request deadlines', () => {
  it('uses server phase deadlines for sync while preserving ordinary request limits', async () => {
    const requests: InternalAxiosRequestConfig[] = [];
    request.defaults.adapter = async (config) => {
      requests.push(config);
      return { data: {}, status: 200, statusText: 'OK', headers: {}, config };
    };

    await routeApi.previewSync('project-1', {
      scope: 'selected',
      route_ids: ['route-1'],
      changes: [],
    });
    await routeApi.confirmSync('project-1', {
      route_ids: ['route-1'],
      changes: [],
      business_hash: 'business',
      publication_hash: 'publication',
    });
    await routeApi.get('project-1', 'route-1');

    expect(requests.map(({ url, timeout }) => ({ url, timeout }))).toEqual([
      { url: '/api/route/sync/preview', timeout: 0 },
      { url: '/api/route/sync/confirm', timeout: 0 },
      { url: '/api/route/route-1', timeout: DEFAULT_REQUEST_TIMEOUT },
    ]);
  });
});
