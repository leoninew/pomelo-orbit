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
			path: '/google/callback',
			name: 'GoogleCallback',
			component: () => import('@/views/GoogleCallback.vue'),
			meta: { title: 'Google 登录', public: true },
		},
		{
			path: '/',
			name: 'Home',
			component: () => import('@/views/Home.vue'),
			meta: { title: '首页', menuKey: 'home' },
		},
		{
			path: '/home',
			redirect: '/',
		},
		{
			path: '/projects',
			name: 'Projects',
			component: () => import('@/views/ProjectPage.vue'),
			meta: { title: '项目管理', menuKey: 'projects' },
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
			path: '/ci/credential',
			name: 'Credentials',
			component: () => import('@/views/ci/CredentialPage.vue'),
			meta: { title: '凭据管理', menuKey: 'credentials' },
		},
		{
			path: '/ci/credential/:id',
			name: 'CredentialDetail',
			component: () => import('@/views/ci/CredentialDetail.vue'),
			props: true,
			meta: { title: '凭据详情', menuKey: 'credentials' },
		},
		{
			path: '/ci/build-stage',
			name: 'BuildStagePage',
			component: () => import('@/views/ci/BuildStagePage.vue'),
			meta: { title: '构建阶段', menuKey: 'buildstages' },
		},
		{
			path: '/ci/build-stage/:id',
			name: 'BuildStageDetail',
			component: () => import('@/views/ci/BuildStageDetail.vue'),
			meta: { title: '构建阶段详情', menuKey: 'buildstages' },
		},
		{
			path: '/ci/template',
			name: 'PipelineTemplates',
			component: () => import('@/views/ci/PipelineTemplatePage.vue'),
			meta: { title: '流水线模板', menuKey: 'pipelinetemplates' },
		},
		{
			path: '/ci/template/:id',
			name: 'PipelineTemplateDetail',
			component: () => import('@/views/ci/PipelineTemplateDetail.vue'),
			meta: { title: '模板详情', menuKey: 'pipelinetemplates' },
		},
		{
			path: '/ci/snapshot/:id',
			name: 'PipelineSnapshotDetail',
			component: () => import('@/views/ci/PipelineSnapshotDetail.vue'),
			meta: { title: '快照详情', menuKey: 'pipelinetemplates' },
		},
		{
			path: '/ci/repository',
			name: 'Repositories',
			component: () => import('@/views/ci/RepositoryPage.vue'),
			meta: { title: '代码仓库', menuKey: 'repository' },
		},
		{
			path: '/ci/repository/:id',
			name: 'RepositoryDetail',
			component: () => import('@/views/ci/RepositoryDetail.vue'),
			meta: { title: '仓库详情', menuKey: 'repository' },
		},
		{
			path: '/ci/run',
			name: 'PipelineRuns',
			component: () => import('@/views/ci/PipelineRunPage.vue'),
			meta: { title: '流水线记录', menuKey: 'pipelineruns' },
		},
		{
			path: '/ci/run/:id',
			name: 'PipelineRunDetail',
			component: () => import('@/views/ci/PipelineRunDetail.vue'),
			meta: { title: 'Run 详情', menuKey: 'pipelineruns' },
		},
		{
			path: '/ci/artifact',
			name: 'Artifacts',
			component: () => import('@/views/ci/ArtifactPage.vue'),
			meta: { title: '制品记录', menuKey: 'artifacts' },
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
		} catch {
			authStore.clearToken();
			next({ name: 'Login', query: { redirect: to.fullPath } });
			return;
		}
	}

	next();
});

export default router;
