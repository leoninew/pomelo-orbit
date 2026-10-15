import { defineStore } from 'pinia';
import { computed, onScopeDispose, ref } from 'vue';
import { authApi } from '@/api/auth/auth';
import type { UserInfoResp } from '@/gen/proto/orbit/v1/auth/auth';

const TOKEN_KEY = 'pomelo_orbit_token';

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY));
  const user = ref<UserInfoResp>();

  const isAuthenticated = computed(() => !!token.value);

  // Read storage before requests as well: a suspended tab may not have handled
  // the storage event yet when the user resumes it.
  function syncToken() {
    const storedToken = localStorage.getItem(TOKEN_KEY);
    if (token.value !== storedToken) {
      token.value = storedToken;
      user.value = undefined;
    }
    return token.value;
  }

  function onStorage(event: StorageEvent) {
    if (event.storageArea === localStorage && (event.key === TOKEN_KEY || event.key === null)) {
      syncToken();
    }
  }
  window.addEventListener('storage', onStorage);
  onScopeDispose(() => window.removeEventListener('storage', onStorage));

  function hasPermission(permission: string) {
    return user.value?.permissions.includes(permission) ?? false;
  }

  // 设置 token
  function setToken(newToken: string) {
    localStorage.setItem(TOKEN_KEY, newToken);
    syncToken();
  }

  // 清除 token
  function clearToken() {
    token.value = null;
    user.value = undefined;
    localStorage.removeItem(TOKEN_KEY);
  }

  // 登录
  async function login(
    email: string,
    password: string,
    csrfToken: string,
    turnstileToken?: string
  ) {
    const response = await authApi.login({
      email,
      password,
      csrf_token: csrfToken,
      turnstile_token: turnstileToken ?? '',
    });
    setToken(response.access_token);
    if (!(await fetchUser())) {
      throw new Error('登录状态已变更，请重试');
    }
    return response;
  }

  // 登出
  async function logout() {
    const requestToken = syncToken();
    try {
      await authApi.logout({});
    } catch (error) {
      console.error('Failed to logout:', error);
    } finally {
      if (syncToken() === requestToken) {
        clearToken();
      }
    }
  }

  // 获取当前用户信息
  async function fetchUser() {
    const requestToken = syncToken();
    if (!requestToken) {
      return undefined;
    }
    // Authentication failures are handled by the request boundary. Network and
    // server failures must propagate without deleting a valid session.
    const userInfo = await authApi.getCurrentUser();
    if (syncToken() === requestToken) {
      user.value = userInfo;
      return userInfo;
    }
    return undefined;
  }

  // 修改密码
  async function changePassword(oldPassword: string, newPassword: string) {
    await authApi.changePassword({
      old_password: oldPassword,
      new_password: newPassword,
    });
  }

  return {
    token,
    user,
    isAuthenticated,
    hasPermission,
    setToken,
    syncToken,
    clearToken,
    login,
    logout,
    fetchUser,
    changePassword,
  };
});
