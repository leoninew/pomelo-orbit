// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { authApi } from '@/api/auth/auth';
import { useAuthStore } from './auth';
import type { UserInfoResp } from '@/gen/proto/orbit/v1/auth/auth';

vi.mock('@/api/auth/auth', () => ({
  authApi: { getCurrentUser: vi.fn(), login: vi.fn() },
}));

const profile: UserInfoResp = {
  id: 'user-1',
  username: 'user',
  email: 'user@example.com',
  auth_source: 'local',
  created_at: '',
  roles: [],
  permissions: ['user:read'],
};
let store: ReturnType<typeof useAuthStore>;

beforeEach(() => {
  localStorage.clear();
  setActivePinia(createPinia());
  vi.resetAllMocks();
  store = useAuthStore();
});
afterEach(() => store.$dispose());

describe('auth session', () => {
  it('loads the profile for the signed-in session', async () => {
    store.setToken('session-token');
    vi.mocked(authApi.getCurrentUser).mockResolvedValue(profile);
    expect(await store.fetchUser()).toEqual(profile);
    expect(store.hasPermission('user:read')).toBe(true);
  });

  it('reads a new stored session and invalidates the previous user permissions', () => {
    store.setToken('previous-token');
    store.user = profile;
    localStorage.setItem('pomelo_orbit_token', 'new-token');
    expect(store.syncToken()).toBe('new-token');
    expect(store.user).toBeUndefined();
    expect(store.hasPermission('user:read')).toBe(false);
  });

  it('observes login and logout notifications from other tabs', () => {
    localStorage.setItem('pomelo_orbit_token', 'other-tab-token');
    window.dispatchEvent(
      new StorageEvent('storage', {
        key: 'pomelo_orbit_token',
        storageArea: localStorage,
      })
    );
    expect(store.token).toBe('other-tab-token');
    localStorage.clear();
    window.dispatchEvent(new StorageEvent('storage', { key: null, storageArea: localStorage }));
    expect(store.isAuthenticated).toBe(false);
  });

  it('preserves the token and cached profile when loading the user fails', async () => {
    store.setToken('valid-token');
    store.user = profile;
    const error = new Error('Network unavailable');
    vi.mocked(authApi.getCurrentUser).mockRejectedValue(error);
    await expect(store.fetchUser()).rejects.toBe(error);
    expect(store.token).toBe('valid-token');
    expect(localStorage.getItem('pomelo_orbit_token')).toBe('valid-token');
    expect(store.user).toEqual(profile);
  });

  it('reports profile loading failure instead of reporting login success', async () => {
    vi.mocked(authApi.login).mockResolvedValue({ access_token: 'new-token', token_type: 'bearer' });
    vi.mocked(authApi.getCurrentUser).mockRejectedValue(new Error('Service unavailable'));
    await expect(store.login('user@example.com', 'password', 'csrf')).rejects.toThrow(
      'Service unavailable'
    );
    expect(store.token).toBe('new-token');
  });
});
