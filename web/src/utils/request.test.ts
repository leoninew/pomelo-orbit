import { AxiosError, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { authStore, handleUnauthorized } = vi.hoisted(() => ({
  authStore: { token: null as string | null },
  handleUnauthorized: vi.fn(),
}));

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}));

vi.mock('@/utils/handle-unauthorized', () => ({
  handleUnauthorized,
}));

import request, { isHttpErrorResponse, toApiError } from '@/utils/request';

describe('HTTP error contract', () => {
  beforeEach(() => {
    authStore.token = null;
    handleUnauthorized.mockReset();
  });

  it('retains status, code, and request ID from a valid response', () => {
    const error = toApiError(409, {
      code: 'version_already_published',
      error: 'Version is already published.',
      requestId: 'request-2',
    });

    expect(error).toMatchObject({
      message: 'Version is already published.',
      status: 409,
      code: 'version_already_published',
      requestId: 'request-2',
      kind: 'api',
    });
  });

  it('rejects the old detail response as a contract mismatch', () => {
    expect(isHttpErrorResponse({ detail: 'database password=secret' })).toBe(false);
    expect(toApiError(500, { detail: 'database password=secret' })).toMatchObject({
      message: '服务响应格式异常',
      status: 500,
      code: 'contract_mismatch',
      kind: 'contract_mismatch',
    });
  });

  it('preserves the 401 boundary while retaining structured metadata', async () => {
    const config = { headers: {} } as InternalAxiosRequestConfig;
    const response: AxiosResponse = {
      data: { code: 'unauthorized', error: 'Expired session.', requestId: 'request-3' },
      status: 401,
      statusText: 'Unauthorized',
      headers: {},
      config,
    };
    request.defaults.adapter = async () => {
      throw new AxiosError('request failed', 'ERR_BAD_RESPONSE', config, {}, response);
    };

    await expect(request.get('/api/projects')).rejects.toMatchObject({
      message: '登录已过期，请重新登录',
      status: 401,
      code: 'unauthorized',
      requestId: 'request-3',
    });
    expect(handleUnauthorized).toHaveBeenCalledTimes(1);
  });

  it('normalizes an Axios network failure without exposing its transport message', async () => {
    const config = { headers: {} } as InternalAxiosRequestConfig;
    request.defaults.adapter = async () => {
      throw new AxiosError('socket reset', 'ERR_NETWORK', config, {});
    };

    await expect(request.get('/api/projects')).rejects.toMatchObject({
      message: '网络连接失败，请检查网络设置',
      code: 'network_error',
      kind: 'network',
    });
  });
});
