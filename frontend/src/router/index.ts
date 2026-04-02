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
		// CD
		{
			path: '/cd/applications',
			name: 'Applications',
			component: () => import('@/views/cd/ApplicationPage.vue'),
			meta: { title: '应用管理', menuKey: 'applications' },
		},
		{
			path: '/cd/applications/:id',
			name: 'ApplicationDetail',
			component: () => import('@/views/cd/ApplicationDetail.vue'),
			meta: { title: '应用详情', menuKey: 'applications' },
		},
		{
			path: '/cd/deployments',
			name: 'Deployments',
			component: () => import('@/views/cd/DeploymentPage.vue'),
			meta: { title: '部署记录', menuKey: 'deployments' },
		},
		{
			path: '/cd/deployments/:id',
			name: 'DeploymentDetail',
			component: () => import('@/views/cd/DeploymentDetail.vue'),
			meta: { title: '部署详情', menuKey: 'deployments' },
		},
		{
			path: '/cd/routes',
			name: 'Route',
			component: () => import('@/views/cd/RoutePage.vue'),
			meta: { title: '路由配置', menuKey: 'route' },
		},
		{
			path: '/cd/routes/:id',
			name: 'RouteDetail',
			component: () => import('@/views/cd/RouteDetail.vue'),
			meta: { title: '路由详情', menuKey: 'route' },
		},
		{
			path: '/cd/traefik-http-routers',
			name: 'TraefikRoute',
			component: () => import('@/views/cd/TraefikRoutePage.vue'),
			meta: { title: 'Traefik HTTP Routers', menuKey: 'traefik-http-routers' },
		},
		// CI
		{
			path: '/ci/credentials',
			name: 'Credentials',
			component: () => import('@/views/ci/CredentialPage.vue'),
			meta: { title: '凭据管理', menuKey: 'credentials' },
		},
		{
			path: '/ci/templates',
			name: 'PipelineTemplates',
			component: () => import('@/views/ci/PipelineTemplatePage.vue'),
			meta: { title: '流水线模板', menuKey: 'pipelinetemplates' },
		},
		{
			path: '/ci/templates/:id',
			name: 'PipelineTemplateDetail',
			component: () => import('@/views/ci/PipelineTemplateDetail.vue'),
			meta: { title: '模板详情', menuKey: 'pipelinetemplates' },
		},
		{
			path: '/ci/projects',
			name: 'Projects',
			component: () => import('@/views/ci/ProjectPage.vue'),
			meta: { title: '项目管理', menuKey: 'projects' },
		},
		{
			path: '/ci/projects/:id',
			name: 'ProjectDetail',
			component: () => import('@/views/ci/ProjectDetail.vue'),
			meta: { title: '项目详情', menuKey: 'projects' },
		},
		{
			path: '/ci/runs',
			name: 'PipelineRuns',
			component: () => import('@/views/ci/PipelineRunPage.vue'),
			meta: { title: 'Pipeline Runs', menuKey: 'pipelineruns' },
		},
		{
			path: '/ci/runs/:id',
			name: 'PipelineRunDetail',
			component: () => import('@/views/ci/PipelineRunDetail.vue'),
			meta: { title: 'Run 详情', menuKey: 'pipelineruns' },
		},
		// Shared
		{
			path: '/login-history',
			name: 'LoginHistory',
			component: () => import('@/views/LoginHistoryPage.vue'),
			meta: { title: '登录历史', menuKey: 'loginhistory' },
		},
		{
			path: '/settings',
			name: 'Settings',
			component: () => import('@/views/Settings.vue'),
			meta: { title: '系统设置', menuKey: 'settings' },
		},
	],
});

// 路由守卫 - 检查登录状态
router.beforeEach(async (to, _from, next) => {
	document.title = `${to.meta.title || 'Pomelo Orbit'} - Pomelo Orbit`;

	if (to.meta.public) {
		next();
		return;
	}

	const authStore = useAuthStore();
	if (!authStore.isAuthenticated) {
		next({ name: 'Login', query: { redirect: to.fullPath } });
		return;
	}

	if (!authStore.user) {
		try {
			await authStore.fetchUser();
		} catch (error) {
			console.error('Failed to fetch user in router guard:', error);
			authStore.clearToken();
			next({ name: 'Login', query: { redirect: to.fullPath } });
			return;
		}
	}

	next();
});

export default router;
