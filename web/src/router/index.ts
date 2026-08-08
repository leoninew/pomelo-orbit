import { createRouter, createWebHistory } from 'vue-router';
import i18n from '@/i18n';
import { getNavigationScope, getSecondaryNavigationTitle } from '@/navigation';
import { useAuthStore } from '@/stores/auth';
import { PERMISSIONS } from '@/constants/permissions';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/auth/Login.vue'),
      meta: { title: '登录', public: true },
    },
    {
      path: '/google/callback',
      name: 'GoogleCallback',
      component: () => import('@/views/auth/GoogleCallback.vue'),
      meta: { title: 'Google 登录', public: true },
    },

    {
      path: '/mcp-authorize',
      name: 'MCPAuthorize',
      component: () => import('@/views/auth/MCPAuthorize.vue'),
      meta: { title: 'MCP 授权' },
    },
    {
      path: '/403',
      name: 'Forbidden',
      component: () => import('@/views/auth/ForbiddenPage.vue'),
      meta: { title: '无权访问' },
    },
    {
      path: '/',
      name: 'Home',
      component: () => import('@/views/home/Home.vue'),
      meta: { title: '首页', menuKey: 'home' },
    },
    {
      path: '/home',
      redirect: '/',
    },
    {
      path: '/projects',
      name: 'Projects',
      component: () => import('@/views/project/ProjectPage.vue'),
      meta: { title: '项目管理', menuKey: 'projects' },
    },
    {
      path: '/project/:id',
      name: 'ProjectDetail',
      component: () => import('@/views/project/ProjectDetail.vue'),
      props: true,
      meta: { title: '项目详情', menuKey: 'projects' },
    },
    {
      path: '/users',
      name: 'Users',
      component: () => import('@/views/user/UserPage.vue'),
      meta: { title: '用户管理', menuKey: 'users', permission: PERMISSIONS.USER_READ },
    },
    {
      path: '/user/:id',
      name: 'UserDetail',
      component: () => import('@/views/user/UserDetail.vue'),
      props: true,
      meta: { title: '用户详情', menuKey: 'users', permission: PERMISSIONS.USER_READ },
    },
    {
      path: '/roles',
      name: 'Roles',
      component: () => import('@/views/role/RolePage.vue'),
      meta: { title: '角色管理', menuKey: 'roles', permission: PERMISSIONS.ROLE_READ },
    },
    {
      path: '/role/:id',
      name: 'RoleDetail',
      component: () => import('@/views/role/RoleDetail.vue'),
      props: true,
      meta: { title: '角色详情', menuKey: 'roles', permission: PERMISSIONS.ROLE_READ },
    },
    // application / service / deployment / gateway / route
    {
      path: '/applications',
      name: 'Applications',
      component: () => import('@/views/application/ApplicationPage.vue'),
      meta: { title: '应用', menuKey: 'applications' },
    },
    {
      path: '/application/:id',
      name: 'ApplicationDetail',
      component: () => import('@/views/application/ApplicationDetail.vue'),
      meta: { title: '应用详情', menuKey: 'applications' },
    },
    {
      path: '/versions',
      name: 'Versions',
      component: () => import('@/views/application/VersionsPage.vue'),
      meta: { title: '版本', menuKey: 'applications' },
    },
    {
      path: '/version/:versionId/component/:componentId',
      name: 'VersionComponentDetail',
      component: () => import('@/views/application/VersionComponentDetail.vue'),
      meta: { title: '组件详情', menuKey: 'applications' },
    },
    {
      path: '/version/:id',
      name: 'VersionDetail',
      component: () => import('@/views/application/VersionDetail.vue'),
      meta: { title: '版本详情', menuKey: 'applications' },
    },
    {
      path: '/services',
      name: 'Services',
      component: () => import('@/views/service/ServicePage.vue'),
      meta: { title: '服务', menuKey: 'services' },
    },
    {
      path: '/service/:id',
      name: 'ServiceDetail',
      component: () => import('@/views/service/ServiceDetail.vue'),
      meta: { title: '服务详情', menuKey: 'services' },
    },
    {
      path: '/service/:id/component/:componentId',
      name: 'ServiceComponentDetail',
      component: () => import('@/views/service/ServiceComponentDetail.vue'),
      meta: { title: '服务组件配置', menuKey: 'services' },
    },
    {
      path: '/gateways',
      name: 'Gateways',
      component: () => import('@/views/gateway/GatewayPage.vue'),
      meta: { title: '网关', menuKey: 'gateways' },
    },
    {
      path: '/gateway/edit/:id',
      name: 'GatewayEdit',
      component: () => import('@/views/gateway/GatewayEdit.vue'),
      meta: { title: '编辑网关', menuKey: 'gateways' },
    },
    {
      path: '/gateway/:id',
      name: 'GatewayDetail',
      component: () => import('@/views/gateway/GatewayDetail.vue'),
      meta: { title: '网关详情', menuKey: 'gateways' },
    },
    {
      path: '/deployments',
      name: 'Deployments',
      component: () => import('@/views/deployment/DeploymentPage.vue'),
      meta: { title: '部署记录', menuKey: 'deployments' },
    },
    {
      path: '/dialogue',
      name: 'DeploymentDialogue',
      component: () => import('@/views/deployment/DeploymentDialoguePage.vue'),
      meta: { title: '对话', menuKey: 'deployment-dialogue' },
    },
    {
      path: '/deployment/:id',
      name: 'DeploymentDetail',
      component: () => import('@/views/deployment/DeploymentDetail.vue'),
      meta: { title: '部署详情', menuKey: 'deployments' },
    },
    {
      path: '/routes',
      name: 'Route',
      component: () => import('@/views/route/RoutePage.vue'),
      meta: { title: '路由', menuKey: 'route' },
    },
    {
      path: '/route/:id',
      name: 'RouteDetail',
      component: () => import('@/views/route/RouteDetail.vue'),
      meta: { title: '路由详情', menuKey: 'route' },
    },
    {
      path: '/route/traefik',
      redirect: '/routes',
    },
    // credential / repository / pipeline / pipeline_run
    {
      path: '/credential',
      name: 'Credentials',
      component: () => import('@/views/credential/CredentialPage.vue'),
      meta: { title: '凭据管理', menuKey: 'credentials' },
    },
    {
      path: '/credential/:id',
      name: 'CredentialDetail',
      component: () => import('@/views/credential/CredentialDetail.vue'),
      props: true,
      meta: { title: '凭据详情', menuKey: 'credentials' },
    },
    {
      path: '/pipeline',
      name: 'Pipelines',
      component: () => import('@/views/pipeline/PipelinePage.vue'),
      meta: { title: '流水线', menuKey: 'pipelines' },
    },
    {
      path: '/pipeline-stage',
      name: 'PipelineStages',
      component: () => import('@/views/pipeline/PipelineStagePage.vue'),
      meta: { title: '阶段', menuKey: 'pipelinestages' },
    },
    {
      path: '/pipeline-stage/:id',
      name: 'PipelineStageDetail',
      component: () => import('@/views/pipeline/PipelineStageDetail.vue'),
      meta: { title: '阶段详情', menuKey: 'pipelinestages' },
    },
    {
      path: '/pipeline/snapshot/:id',
      name: 'PipelineSnapshotDetail',
      component: () => import('@/views/pipeline/PipelineSnapshotDetail.vue'),
      meta: { title: '快照详情', menuKey: 'pipelines' },
    },
    {
      path: '/pipeline/:id',
      name: 'PipelineDetail',
      component: () => import('@/views/pipeline/PipelineDetail.vue'),
      meta: { title: '流水线详情', menuKey: 'pipelines' },
    },
    {
      path: '/repository',
      name: 'Repositories',
      component: () => import('@/views/repository/RepositoryPage.vue'),
      meta: { title: '代码仓库', menuKey: 'repository' },
    },
    {
      path: '/repository/:id',
      name: 'RepositoryDetail',
      component: () => import('@/views/repository/RepositoryDetail.vue'),
      meta: { title: '仓库详情', menuKey: 'repository' },
    },
    {
      path: '/pipeline-run',
      name: 'PipelineRuns',
      component: () => import('@/views/pipeline_run/PipelineRunPage.vue'),
      meta: { title: '流水线记录', menuKey: 'pipelineruns' },
    },
    {
      path: '/pipeline-run/artifact',
      name: 'Artifacts',
      component: () => import('@/views/pipeline_run/ArtifactPage.vue'),
      meta: { title: '制品记录', menuKey: 'artifacts' },
    },
    {
      path: '/pipeline-run/artifact/:id',
      name: 'ArtifactDetail',
      component: () => import('@/views/pipeline_run/ArtifactDetail.vue'),
      meta: { title: '制品详情', menuKey: 'artifacts' },
    },
    {
      path: '/pipeline-run/:id',
      name: 'PipelineRunDetail',
      component: () => import('@/views/pipeline_run/PipelineRunDetail.vue'),
      meta: { title: 'Run 详情', menuKey: 'pipelineruns' },
    },
    // auth / settings
    {
      path: '/login-history',
      name: 'LoginHistory',
      component: () => import('@/views/auth/LoginHistoryPage.vue'),
      meta: { title: '登录历史', menuKey: 'loginhistory', permission: PERMISSIONS.LOGIN_READ },
    },
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('@/views/settings/Settings.vue'),
      meta: { title: '系统设置', menuKey: 'settings', permission: PERMISSIONS.SETTING_READ },
    },
  ],
});

// 路由守卫 - 检查登录状态
router.beforeEach(async (to, _from, next) => {
  const scope = getNavigationScope(to.path);
  const menuKey = typeof to.meta.menuKey === 'string' ? to.meta.menuKey : '';
  const navigationTitle =
    scope === 'pipeline' || scope === 'deployment' || scope === 'settings'
      ? getSecondaryNavigationTitle(scope, menuKey, (key) => i18n.global.t(key))
      : null;
  document.title = `${navigationTitle || to.meta.title || 'Pomelo Orbit'} - Pomelo Orbit`;

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
