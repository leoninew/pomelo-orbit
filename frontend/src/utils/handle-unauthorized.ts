import router from '@/router';
import { useAuthStore } from '@/stores/auth';

/**
 * 处理 401 未授权响应
 * 清除 token 并跳转到登录页
 */
export function handleUnauthorized(): void {
	const authStore = useAuthStore();
	authStore.clearToken();

	router.push({
		name: 'Login',
		query: { redirect: router.currentRoute.value.fullPath },
	});
}
