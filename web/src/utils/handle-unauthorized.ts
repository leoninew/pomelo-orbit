import router from '@/router';
import { useAuthStore } from '@/stores/auth';
import { resolveLoginRedirect } from '@/utils/login-redirect';

/**
 * 处理 401 未授权响应
 * 仅当前请求使用的 token 仍为当前会话时，清除登录状态并跳转。
 */
export function handleUnauthorized(requestToken: string | null): boolean {
  const authStore = useAuthStore();
  if (!requestToken || authStore.syncToken() !== requestToken) {
    return false;
  }
  authStore.clearToken();

  const currentRoute = router.currentRoute.value;
  const redirect = resolveLoginRedirect(router, currentRoute.fullPath);
  const query = redirect === '/' ? {} : { redirect };

  if (currentRoute.name === 'Login') {
    if (currentRoute.query.redirect !== query.redirect) {
      void router.replace({ name: 'Login', query });
    }
    return true;
  }

  void router.replace({
    name: 'Login',
    query,
  });
  return true;
}
