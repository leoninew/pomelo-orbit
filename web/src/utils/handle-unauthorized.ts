import router from '@/router';
import { useAuthStore } from '@/stores/auth';
import { resolveLoginRedirect } from '@/utils/login-redirect';

/**
 * 处理 401 未授权响应
 * 清除 token 并跳转到登录页
 */
export function handleUnauthorized(): void {
  const authStore = useAuthStore();
  authStore.clearToken();

  const currentRoute = router.currentRoute.value;
  const redirect = resolveLoginRedirect(router, currentRoute.fullPath);
  const query = redirect === '/' ? {} : { redirect };

  if (currentRoute.name === 'Login') {
    if (currentRoute.query.redirect !== query.redirect) {
      void router.replace({ name: 'Login', query });
    }
    return;
  }

  void router.replace({
    name: 'Login',
    query,
  });
}
