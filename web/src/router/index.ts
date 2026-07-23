import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { PERMISSIONS } from '@/constants/permissions';

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
      path: '/403',
      name: 'Forbidden',
      component: () => import('@/views/ForbiddenPage.vue'),
      meta: { title: '无权访问' },
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
    {
      path: '/project/:id',
      name: 'ProjectDetail',
      component: () => import('@/views/ProjectDetail.vue'),
      props: true,
      meta: { title: '项目详情', menuKey: 'projects' },
    },
    {
      path: '/users',
      name: 'Users',
      component: () => import('@/views/UserPage.vue'),
      meta: { title: '用户管理', menuKey: 'users', permission: PERMISSIONS.USER_READ },
    },
    {
      path: '/user/:id',
      name: 'UserDetail',
      component: () => import('@/views/UserDetail.vue'),
      props: true,
      meta: { title: '用户详情', menuKey: 'users', permission: PERMISSIONS.USER_READ },
    },
    {
      path: '/roles',
      name: 'Roles',
      component: () => import('@/views/RolePage.vue'),
      meta: { title: '角色管理', menuKey: 'roles', permission: PERMISSIONS.ROLE_READ },
    },
    {
      path: '/role/:id',
      name: 'RoleDetail',
      component: () => import('@/views/RoleDetail.vue'),
      props: true,
      meta: { title: '角色详情', menuKey: 'roles', permission: PERMISSIONS.ROLE_READ },
    },
    // CD
    {
      path: '/cd/applications',
      name: 'Applications',
      component: () => import('@/views/cd/ApplicationPage.vue'),
      meta: { title: '应用', menuKey: 'applications' },
    },
    {
      path: '/cd/application/:id',
      name: 'ApplicationDetail',
      component: () => import('@/views/cd/ApplicationDetail.vue'),
      meta: { title: '应用详情', menuKey: 'applications' },
    },
    {
      path: '/cd/versions',
      name: 'Versions',
      component: () => import('@/views/cd/VersionsPage.vue'),
      meta: { title: '版本', menuKey: 'versions' },
    },
    {
      path: '/cd/version/:id',
      name: 'VersionDetail',
      component: () => import('@/views/cd/VersionDetail.vue'),
      meta: { title: '版本详情', menuKey: 'versions' },
    },
    {
      path: '/cd/services',
      name: 'Services',
      component: () => import('@/views/cd/ServicePage.vue'),
      meta: { title: '服务', menuKey: 'services' },
    },
    {
      path: '/cd/service/:id',
      name: 'ServiceDetail',
      component: () => import('@/views/cd/ServiceDetail.vue'),
      meta: { title: '服务详情', menuKey: 'services' },
    },
    {
      path: '/cd/gateways',
      name: 'Gateways',
      component: () => import('@/views/cd/GatewayPage.vue'),
      meta: { title: '网关', menuKey: 'gateways' },
    },
    {
      path: '/cd/gateway/create',
      name: 'GatewayCreate',
      component: () => import('@/views/cd/GatewayCreate.vue'),
      meta: { title: '创建网关', menuKey: 'gateways' },
    },
    {
      path: '/cd/gateway/:id/edit',
      name: 'GatewayEdit',
      component: () => import('@/views/cd/GatewayEdit.vue'),
      meta: { title: '编辑网关', menuKey: 'gateways' },
    },
    {
      path: '/cd/gateway/:id',
      name: 'GatewayDetail',
      component: () => import('@/views/cd/GatewayDetail.vue'),
      meta: { title: '网关详情', menuKey: 'gateways' },
    },
    {
      path: '/cd/deployments',
      name: 'Deployments',
      component: () => import('@/views/cd/DeploymentPage.vue'),
      meta: { title: '部署记录', menuKey: 'deployments' },
    },
    {
      path: '/cd/deployment/:id',
      name: 'DeploymentDetail',
      component: () => import('@/views/cd/DeploymentDetail.vue'),
      meta: { title: '部署详情', menuKey: 'deployments' },
    },
    {
      path: '/cd/environments',
      name: 'Environments',
      component: () => import('@/views/cd/EnvironmentPage.vue'),
      meta: { title: '环境管理', menuKey: 'environments' },
    },
    {
      path: '/cd/routes',
      name: 'Route',
      component: () => import('@/views/cd/RoutePage.vue'),
      meta: { title: '路由配置', menuKey: 'route' },
    },
    {
      path: '/cd/route/:id',
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
      meta: { title: '登录历史', menuKey: 'loginhistory', permission: PERMISSIONS.LOGIN_READ },
    },
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('@/views/Settings.vue'),
      meta: { title: '系统设置', menuKey: 'settings', permission: PERMISSIONS.SETTING_READ },
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
    await authStore.fetchUser();
    if (!authStore.user) {
      next({ name: 'Login', query: { redirect: to.fullPath } });
      return;
    }
  }

  const permission = to.meta.permission;
  if (typeof permission === 'string' && !authStore.hasPermission(permission)) {
    next({ name: 'Forbidden', query: { from: to.fullPath } });
    return;
  }

  next();
});

export default router;
