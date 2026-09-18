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
      if (target.pathname === '/gateway') {
        return {
          fullPath: `${target.pathname}${target.search}${target.hash}`,
          matched: [{ meta: {} }],
          name: 'Gateway',
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
      fullPath: '/gateway?status=healthy',
      name: 'Gateway',
      query: {},
    };

    handleUnauthorized();

    expect(authStore.clearToken).toHaveBeenCalledTimes(1);
    expect(router.replace).toHaveBeenCalledWith({
      name: 'Login',
      query: { redirect: '/gateway?status=healthy' },
    });
  });

  it('normalizes a nested login redirect instead of appending another redirect layer', () => {
    router.currentRoute.value = {
      fullPath: '/login?redirect=/login?redirect=/gateway',
      name: 'Login',
      query: { redirect: '/login?redirect=/gateway' },
    };

    handleUnauthorized();

    expect(authStore.clearToken).toHaveBeenCalledTimes(1);
    expect(router.replace).toHaveBeenCalledWith({
      name: 'Login',
      query: { redirect: '/gateway' },
    });
  });

  it('does not navigate when login already has a valid redirect target', () => {
    router.currentRoute.value = {
      fullPath: '/login?redirect=/gateway',
      name: 'Login',
      query: { redirect: '/gateway' },
    };

    handleUnauthorized();

    expect(authStore.clearToken).toHaveBeenCalledTimes(1);
    expect(router.replace).not.toHaveBeenCalled();
  });
});
