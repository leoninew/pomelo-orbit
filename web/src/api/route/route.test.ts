import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/utils/request', () => ({
  default: { get: vi.fn().mockResolvedValue({}), post: vi.fn().mockResolvedValue({}) },
}));

import { routeApi } from '@/api/route/route';
import request from '@/utils/request';

afterEach(() => {
  vi.clearAllMocks();
});

describe('Route synchronization request deadlines', () => {
  it('uses server phase deadlines for sync while preserving ordinary request limits', async () => {
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

    expect(request.post).toHaveBeenNthCalledWith(
      1,
      '/api/route/sync/preview',
      { scope: 'selected', route_ids: ['route-1'], changes: [] },
      { params: { project_id: 'project-1' }, timeout: 0 }
    );
    expect(request.post).toHaveBeenNthCalledWith(
      2,
      '/api/route/sync/confirm',
      {
        route_ids: ['route-1'],
        changes: [],
        business_hash: 'business',
        publication_hash: 'publication',
      },
      { params: { project_id: 'project-1' }, timeout: 0 }
    );
    expect(request.get).toHaveBeenCalledWith('/api/route/route-1', {
      params: { project_id: 'project-1' },
    });
  });
});
