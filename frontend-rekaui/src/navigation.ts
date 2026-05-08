import {
	FileCode2,
	FolderGit2,
	Globe,
	History,
	KeyRound,
	Layers,
	LayoutGrid,
	Network,
	Package,
	Play,
	Rocket,
	Settings,
} from 'lucide-vue-next';
import type { Component } from 'vue';

export type NavigationScope = 'home' | 'ci' | 'cd';
export type PrimaryNavigationKey = 'ci' | 'cd';

export interface NavigationEntry {
	key: string
	label: string
	path: string
	icon: Component
}

export interface PrimaryNavigationEntry {
	key: PrimaryNavigationKey
	label: string
	path: string
}

export const secondaryNavigation = {
	home: [
		{ key: 'home', label: '项目概述', path: '/', icon: LayoutGrid },
		{ key: 'loginhistory', label: '登录历史', path: '/login-history', icon: History },
		{ key: 'settings', label: '系统设置', path: '/settings', icon: Settings },
	],
	cd: [
		{ key: 'applications', label: '应用管理', path: '/cd/applications', icon: LayoutGrid },
		{ key: 'deployments', label: '部署记录', path: '/cd/deployments', icon: Rocket },
		{ key: 'route', label: '路由配置', path: '/cd/routes', icon: Globe },
		{
			key: 'traefik-http-routers',
			label: 'Traefik Routers',
			path: '/cd/traefik-http-routers',
			icon: Network,
		},
	],
	ci: [
		{ key: 'repository', label: '代码仓库', path: '/ci/repository', icon: FolderGit2 },
		{ key: 'buildstages', label: '构建阶段', path: '/ci/build-stage', icon: Layers },
		{ key: 'pipelinetemplates', label: '流水线模板', path: '/ci/template', icon: FileCode2 },
		{ key: 'pipelineruns', label: '流水线记录', path: '/ci/run', icon: Play },
		{ key: 'artifacts', label: '制品记录', path: '/ci/artifact', icon: Package },
		{ key: 'credentials', label: '凭据管理', path: '/ci/credential', icon: KeyRound },
	],
} satisfies Record<NavigationScope, NavigationEntry[]>;

export const primaryNavigation = [
	{ key: 'ci', label: '持续集成', path: secondaryNavigation.ci[0].path },
	{ key: 'cd', label: '持续部署', path: secondaryNavigation.cd[0].path },
] satisfies PrimaryNavigationEntry[];

export function getNavigationScope(path: string): NavigationScope | null {
	if (path === '/' || path === '/login-history' || path === '/settings') {
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
