import { beforeEach, describe, expect, it, vi } from 'vitest';
import request from '@/utils/request';
import { dnsApi } from './dns';

vi.mock('@/utils/request', () => ({
  default: { get: vi.fn(), put: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

describe('DNS API project scope', () => {
  beforeEach(() => vi.clearAllMocks());
  it('sends the current project, Zone and pagination when listing A records', async () => {
    await dnsApi.records('project-1', 'zone-1', 2, 20, 'api');
    expect(request.get).toHaveBeenCalledWith('/api/dns/zone/zone-1/record', {
      params: { project_id: 'project-1', page: 2, per_page: 20, search: 'api' },
    });
  });
  it('creates an A record for the selected project and Zone', async () => {
    const input = { name: '@', content: '192.0.2.1', ttl: 1, proxied: false };
    await dnsApi.create('project-1', 'zone-1', input);
    expect(request.post).toHaveBeenCalledWith('/api/dns/zone/zone-1/record', input, {
      params: { project_id: 'project-1' },
    });
  });
  it('deletes the selected record in the current project and Zone', async () => {
    await dnsApi.delete('project-1', 'zone-1', 'record-1');
    expect(request.delete).toHaveBeenCalledWith('/api/dns/zone/zone-1/record/record-1', {
      params: { project_id: 'project-1' },
    });
  });
  it('preserves explicit false when turning proxying off', async () => {
    const input = { content: '192.0.2.2', ttl: 300, proxied: false };
    await dnsApi.update('project-1', 'zone-1', 'record-1', input);
    expect(request.put).toHaveBeenCalledWith('/api/dns/zone/zone-1/record/record-1', input, {
      params: { project_id: 'project-1' },
    });
  });
});
