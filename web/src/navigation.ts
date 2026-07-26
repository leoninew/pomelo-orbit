import { FolderGit2, Layers, LayoutGrid, Network, Play, Wrench } from 'lucide-vue-next';
import type { Component } from 'vue';
import { PERMISSIONS } from '@/constants/permissions';

export type NavigationScope = 'home' | 'settings' | 'pipeline' | 'deployment';
export type PrimaryNavigationKey = 'pipeline' | 'deployment' | 'settings';

/** Level-2 leaf (navigable route). */
export interface NavigationLeaf {
  key: string;
  label: string;
  labelKey: string;
  path: string;
  permission?: string;
}

/** Level-1 parent (expandable; not a route by itself). */
export interface NavigationBranch {
  key: string;
  labelKey: string;
  icon: Component;
  children: NavigationLeaf[];
}

export interface PrimaryNavigationEntry {
  key: PrimaryNavigationKey;
  label: string;
  labelKey: string;
  path: string;
}

export interface ResolvedNavigationLeaf {
  key: string;
  label: string;
  path: string;
}

export interface ResolvedNavigationBranch {
  key: string;
  label: string;
  icon: Component;
  /** First child path; used when the sidebar is collapsed. */
  path: string;
  children: ResolvedNavigationLeaf[];
}

export type ResolveNavigationOptions = {
  hasPermission: (permission: string) => boolean;
  t: (key: string) => string;
};

const homeNavigation: NavigationBranch[] = [
  {
    key: 'workspace',
    labelKey: 'nav.groups.workspace',
    icon: LayoutGrid,
    children: [
      { key: 'home', label: '项目概述', labelKey: 'nav.home', path: '/' },
      {
        key: 'projects',
        label: '项目管理',
        labelKey: 'nav.projects',
        path: '/projects',
      },
    ],
  },
];

const settingsNavigation: NavigationBranch[] = [
  {
    key: 'admin',
    labelKey: 'nav.groups.admin',
    icon: Wrench,
    children: [
      {
        key: 'users',
        label: '用户',
        labelKey: 'nav.users',
        path: '/users',
        permission: PERMISSIONS.USER_READ,
      },
      {
        key: 'roles',
        label: '角色',
        labelKey: 'nav.roles',
        path: '/roles',
        permission: PERMISSIONS.ROLE_READ,
      },
      {
        key: 'loginhistory',
        label: '登录历史',
        labelKey: 'nav.loginHistory',
        path: '/login-history',
        permission: PERMISSIONS.LOGIN_READ,
      },
      {
        key: 'settings',
        label: '系统设置',
        labelKey: 'nav.settings',
        path: '/settings',
        permission: PERMISSIONS.SETTING_READ,
      },
    ],
  },
];

const deploymentNavigation: NavigationBranch[] = [
  {
    key: 'runtime',
    labelKey: 'nav.groups.runtime',
    icon: Network,
    children: [
      {
        key: 'gateways',
        label: '网关',
        labelKey: 'nav.gateways',
        path: '/gateways',
      },
      {
        key: 'services',
        label: '服务',
        labelKey: 'nav.services',
        path: '/services',
      },
      {
        key: 'route',
        label: '路由配置',
        labelKey: 'nav.routes',
        path: '/routes',
      },
      {
        key: 'traefik-http-routers',
        label: 'Traefik Routers',
        labelKey: 'nav.traefikRoutes',
        path: '/route/traefik',
      },
    ],
  },
  {
    key: 'workload',
    labelKey: 'nav.groups.workload',
    icon: Layers,
    children: [
      {
        key: 'applications',
        label: '应用',
        labelKey: 'nav.applications',
        path: '/applications',
      },
      { key: 'versions', label: '版本', labelKey: 'nav.versions', path: '/versions' },
      {
        key: 'deployments',
        label: '部署记录',
        labelKey: 'nav.deployments',
        path: '/deployments',
      },
    ],
  },
];

const pipelineNavigation: NavigationBranch[] = [
  {
    key: 'code',
    labelKey: 'nav.groups.code',
    icon: FolderGit2,
    children: [
      {
        key: 'repository',
        label: '仓库',
        labelKey: 'nav.repositories',
        path: '/repository',
      },
      {
        key: 'credentials',
        label: '凭据',
        labelKey: 'nav.credentials',
        path: '/credential',
      },
    ],
  },
  {
    key: 'pipeline',
    labelKey: 'nav.groups.pipeline',
    icon: Play,
    children: [
      {
        key: 'pipelinetemplates',
        label: '模板',
        labelKey: 'nav.pipelineTemplates',
        path: '/pipeline/template',
      },
      {
        key: 'buildstages',
        label: '阶段',
        labelKey: 'nav.buildStages',
        path: '/pipeline/stage',
      },
      {
        key: 'pipelineruns',
        label: '记录',
        labelKey: 'nav.pipelineRuns',
        path: '/pipeline-run',
      },
      {
        key: 'artifacts',
        label: '制品',
        labelKey: 'nav.artifacts',
        path: '/pipeline-run/artifact',
      },
    ],
  },
];

/** Two-level secondary navigation keyed by scope. */
export const secondaryNavigation: Record<NavigationScope, NavigationBranch[]> = {
  home: homeNavigation,
  settings: settingsNavigation,
  pipeline: pipelineNavigation,
  deployment: deploymentNavigation,
};

export function firstNavigationPath(branches: NavigationBranch[]): string {
  for (const branch of branches) {
    const leaf = branch.children.find((entry) => entry.path);
    if (leaf) {
      return leaf.path;
    }
  }
  return '/';
}

export function resolveSecondaryNavigation(
  scope: NavigationScope,
  options: ResolveNavigationOptions
): ResolvedNavigationBranch[] {
  return secondaryNavigation[scope]
    .map((branch) => {
      const children = branch.children
        .filter((item) => !item.permission || options.hasPermission(item.permission))
        .map((item) => ({
          key: item.key,
          label: options.t(item.labelKey),
          path: item.path,
        }))
        .filter((item) => item.path);

      return {
        key: branch.key,
        label: options.t(branch.labelKey),
        icon: branch.icon,
        path: children[0]?.path ?? '',
        children,
      };
    })
    .filter((branch) => branch.children.length > 0);
}

export const primaryNavigation = [
  {
    key: 'pipeline',
    label: '持续集成',
    labelKey: 'nav.continuousIntegration',
    path: firstNavigationPath(secondaryNavigation.pipeline),
  },
  {
    key: 'deployment',
    label: '持续部署',
    labelKey: 'nav.continuousDeployment',
    path: firstNavigationPath(secondaryNavigation.deployment),
  },
  {
    key: 'settings',
    label: '系统管理',
    labelKey: 'nav.systemManagement',
    path: firstNavigationPath(secondaryNavigation.settings),
  },
] satisfies PrimaryNavigationEntry[];

export function getNavigationScope(path: string): NavigationScope | null {
  if (
    path === '/' ||
    path === '/home' ||
    path === '/projects' ||
    path.startsWith('/project/') ||
    path === '/users' ||
    path.startsWith('/user/') ||
    path === '/roles' ||
    path.startsWith('/role/') ||
    path === '/login-history' ||
    path === '/settings'
  ) {
    return path === '/' || path === '/home' || path === '/projects' || path.startsWith('/project/')
      ? 'home'
      : 'settings';
  }
  if (
    path === '/pipeline' ||
    path.startsWith('/pipeline/') ||
    path === '/pipeline-run' ||
    path.startsWith('/pipeline-run/') ||
    path === '/repository' ||
    path.startsWith('/repository/') ||
    path === '/credential' ||
    path.startsWith('/credential/')
  ) {
    return 'pipeline';
  }
  if (
    path === '/application' ||
    path.startsWith('/application/') ||
    path === '/applications' ||
    path === '/version' ||
    path.startsWith('/version/') ||
    path === '/versions' ||
    path === '/service' ||
    path.startsWith('/service/') ||
    path === '/services' ||
    path === '/deployment' ||
    path.startsWith('/deployment/') ||
    path === '/deployments' ||
    path === '/gateway' ||
    path.startsWith('/gateway/') ||
    path === '/gateways' ||
    path === '/route' ||
    path.startsWith('/route/') ||
    path === '/routes'
  ) {
    return 'deployment';
  }
  return null;
}

export function getPrimaryNavigationKey(path: string): PrimaryNavigationKey | null {
  const scope = getNavigationScope(path);
  return scope === 'pipeline' || scope === 'deployment' || scope === 'settings' ? scope : null;
}
