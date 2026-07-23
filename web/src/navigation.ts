import { FolderGit2, KeyRound, Layers, LayoutGrid, Network, Play, Wrench } from 'lucide-vue-next';
import type { Component } from 'vue';
import { PERMISSIONS } from '@/constants/permissions';

export type NavigationScope = 'home' | 'ci' | 'cd';
export type PrimaryNavigationKey = 'ci' | 'cd';

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
  {
    key: 'admin',
    labelKey: 'nav.groups.admin',
    icon: Wrench,
    children: [
      {
        key: 'users',
        label: '用户管理',
        labelKey: 'nav.users',
        path: '/users',
        permission: PERMISSIONS.USER_READ,
      },
      {
        key: 'roles',
        label: '角色管理',
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

const cdNavigation: NavigationBranch[] = [
  {
    key: 'workload',
    labelKey: 'nav.groups.workload',
    icon: Layers,
    children: [
      {
        key: 'applications',
        label: '应用',
        labelKey: 'nav.applications',
        path: '/cd/applications',
      },
      { key: 'versions', label: '版本', labelKey: 'nav.versions', path: '/cd/versions' },
      {
        key: 'services',
        label: '服务',
        labelKey: 'nav.services',
        path: '/cd/services',
      },
      {
        key: 'deployments',
        label: '部署记录',
        labelKey: 'nav.deployments',
        path: '/cd/deployments',
      },
      {
        key: 'environments',
        label: '环境管理',
        labelKey: 'nav.environments',
        path: '/cd/environments',
      },
    ],
  },
  {
    key: 'access',
    labelKey: 'nav.groups.access',
    icon: Network,
    children: [
      {
        key: 'gateways',
        label: '网关',
        labelKey: 'nav.gateways',
        path: '/cd/gateways',
      },
      {
        key: 'route',
        label: '路由配置',
        labelKey: 'nav.routes',
        path: '/cd/routes',
      },
      {
        key: 'traefik-http-routers',
        label: 'Traefik Routers',
        labelKey: 'nav.traefikRoutes',
        path: '/cd/traefik-http-routers',
      },
    ],
  },
];

const ciNavigation: NavigationBranch[] = [
  {
    key: 'pipeline',
    labelKey: 'nav.groups.pipeline',
    icon: FolderGit2,
    children: [
      {
        key: 'repository',
        label: '代码仓库',
        labelKey: 'nav.repositories',
        path: '/ci/repository',
      },
      {
        key: 'buildstages',
        label: '构建阶段',
        labelKey: 'nav.buildStages',
        path: '/ci/build-stage',
      },
      {
        key: 'pipelinetemplates',
        label: '流水线模板',
        labelKey: 'nav.pipelineTemplates',
        path: '/ci/template',
      },
    ],
  },
  {
    key: 'runtime',
    labelKey: 'nav.groups.runtime',
    icon: Play,
    children: [
      {
        key: 'pipelineruns',
        label: '流水线记录',
        labelKey: 'nav.pipelineRuns',
        path: '/ci/run',
      },
      {
        key: 'artifacts',
        label: '制品记录',
        labelKey: 'nav.artifacts',
        path: '/ci/artifact',
      },
    ],
  },
  {
    key: 'security',
    labelKey: 'nav.groups.security',
    icon: KeyRound,
    children: [
      {
        key: 'credentials',
        label: '凭据管理',
        labelKey: 'nav.credentials',
        path: '/ci/credential',
      },
    ],
  },
];

/** Two-level secondary navigation keyed by scope. */
export const secondaryNavigation: Record<NavigationScope, NavigationBranch[]> = {
  home: homeNavigation,
  cd: cdNavigation,
  ci: ciNavigation,
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
    key: 'ci',
    label: '持续集成',
    labelKey: 'nav.ci',
    path: firstNavigationPath(secondaryNavigation.ci),
  },
  {
    key: 'cd',
    label: '持续部署',
    labelKey: 'nav.cd',
    path: firstNavigationPath(secondaryNavigation.cd),
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
    return 'home';
  }
  if (path === '/ci' || path.startsWith('/ci/')) {
    return 'ci';
  }
  if (path === '/cd' || path.startsWith('/cd/')) {
    return 'cd';
  }
  return null;
}

export function getPrimaryNavigationKey(path: string): PrimaryNavigationKey | null {
  const scope = getNavigationScope(path);
  return scope === 'ci' || scope === 'cd' ? scope : null;
}
