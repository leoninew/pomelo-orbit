import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const router = createRouter({
	history: createWebHistory(),
	routes: [
		{
			path: '/login',
			name: 'Login',
			component: () => import('@/views/Login.vue'),
			meta: { title: '登录', public: true },
		},
		{
			path: '/',
			name: 'Home',
			component: () => import('@/views/Home.vue'),
			meta: { title: '首页' },
		},
		{
			path: '/applications',
			name: 'Applications',
			component: () => import('@/views/ApplicationPage.vue'),
			meta: { title: '应用管理' },
		},
		{
			path: '/applications/:id',
			name: 'ApplicationDetail',
			component: () => import('@/views/ApplicationDetail.vue'),
			meta: { title: '应用详情' },
		},
		{
			path: '/deployments',
			name: 'Deployments',
			component: () => import('@/views/DeploymentPage.vue'),
			meta: { title: '部署记录' },
		},
		{
			path: '/deployments/:id',
			name: 'DeploymentDetail',
			component: () => import('@/views/DeploymentDetail.vue'),
			meta: { title: '部署详情' },
		},
		{
			path: '/events',
			name: 'Events',
			component: () => import('@/views/EventPage.vue'),
			meta: { title: '回调事件' },
		},
		{
			path: '/events/:id',
			name: 'EventDetail',
			component: () => import('@/views/EventDetail.vue'),
			meta: { title: '事件详情' },
		},
		// {
		// 	path: '/credentials',
		// 	name: 'Credentials',
		// 	component: () => import('@/views/CredentialPage.vue'),
		// 	meta: { title: '凭据管理' },
		// },
		// {
		// 	path: '/credentials/:id',
		// 	name: 'CredentialDetail',
		// 	component: () => import('@/views/CredentialDetail.vue'),
		// 	meta: { title: '凭据详情' },
		// },
		{
			path: '/routes',
			name: 'Route',
			component: () => import('@/views/RoutePage.vue'),
			meta: { title: '路由配置' },
		},
		{
			path: '/routes/:id',
			name: 'RouteDetail',
			component: () => import('@/views/RouteDetail.vue'),
			meta: { title: '路由详情' },
		},
		{
			path: '/traefik-http-routers',
			name: 'TraefikRoute',
			component: () => import('@/views/TraefikRoutePage.vue'),
			meta: { title: 'Traefik HTTP Routers' },
		},
		{
			path: '/login-history',
			name: 'LoginHistory',
			component: () => import('@/views/LoginHistoryPage.vue'),
			meta: { title: '登录历史' },
		},
		{
			path: '/settings',
			name: 'Settings',
			component: () => import('@/views/Settings.vue'),
			meta: { title: '系统设置' },
		},
	],
});

// 路由守卫 - 检查登录状态
router.beforeEach(async (to, _from, next) => {
	// 设置页面标题
	document.title = `${to.meta.title || 'Pomelo Orbit'} - Pomelo Orbit`;

	// 公开页面直接访问
	if (to.meta.public) {
		next();
		return;
	}

	// 检查登录状态
	const authStore = useAuthStore();
	if (!authStore.isAuthenticated) {
		next({ name: 'Login', query: { redirect: to.fullPath } });
		return;
	}

	// 获取用户信息，如果失败（401）则清除 token 并跳转登录
	if (!authStore.user) {
		try {
			await authStore.fetchUser();
		} catch (error) {
			console.error('Failed to fetch user in router guard:', error);
			// token 过期或无效，清除并跳转登录
			authStore.clearToken();
			next({ name: 'Login', query: { redirect: to.fullPath } });
			return;
		}
	}

	next();
});

export default router;
