import { beforeEach, describe, expect, it, vi } from 'vitest';

const { authStore, router } = vi.hoisted(() => ({
  authStore: { clearToken: vi.fn() },
  router: {
    currentRoute: { value: {} },
    replace: vi.fn(() => Promise.resolve()),
    resolve: vi.fn((path: string) => {
      const target = new URL(path, 'http://pomelo-orbit.local');
      if (target.pathname === '/login') {
        return {
          fullPath: `${target.pathname}${target.search}${target.hash}`,
          matched: [{ meta: { public: true } }],
          name: 'Login',
          query: { redirect: target.searchParams.get('redirect') ?? undefined },
        };
      }
      if (target.pathname === '/gateways') {
        return {
          fullPath: `${target.pathname}${target.search}${target.hash}`,
          matched: [{ meta: {} }],
          name: 'Gateways',
          query: {},
        };
      }
      return { fullPath: path, matched: [], name: undefined, query: {} };
    }),
  },
}));

vi.mock('@/router', () => ({ default: router }));
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }));

import { handleUnauthorized } from './handle-unauthorized';

describe('handleUnauthorized', () => {
  beforeEach(() => {
    authStore.clearToken.mockReset();
    router.replace.mockClear();
  });

  it('replaces a protected route with login while preserving its destination', () => {
    router.currentRoute.value = {
      fullPath: '/gateways?status=healthy',
      name: 'Gateways',
      query: {},
    };

    handleUnauthorized();

    expect(authStore.clearToken).toHaveBeenCalledTimes(1);
    expect(router.replace).toHaveBeenCalledWith({
      name: 'Login',
      query: { redirect: '/gateways?status=healthy' },
    });
  });

  it('normalizes a nested login redirect instead of appending another redirect layer', () => {
    router.currentRoute.value = {
      fullPath: '/login?redirect=/login?redirect=/gateways',
      name: 'Login',
      query: { redirect: '/login?redirect=/gateways' },
    };

    handleUnauthorized();

    expect(authStore.clearToken).toHaveBeenCalledTimes(1);
    expect(router.replace).toHaveBeenCalledWith({
      name: 'Login',
      query: { redirect: '/gateways' },
    });
  });

  it('does not navigate when login already has a valid redirect target', () => {
    router.currentRoute.value = {
      fullPath: '/login?redirect=/gateways',
      name: 'Login',
      query: { redirect: '/gateways' },
    };

    handleUnauthorized();

    expect(authStore.clearToken).toHaveBeenCalledTimes(1);
    expect(router.replace).not.toHaveBeenCalled();
  });
});
