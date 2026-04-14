<template>
	<div class="h-screen flex flex-col font-sans">
		<!-- Login page: no layout chrome -->
		<template v-if="isLoginPage">
			<router-view />
		</template>

		<template v-else>
			<!-- ── Top navbar ── -->
			<header class="navbar bg-base-100 border-b border-base-200 h-12 min-h-12 shrink-0 px-4 gap-0">
				<!-- Logo -->
				<router-link
					to="/"
					class="text-base font-bold text-base-content hover:text-primary transition-colors mr-6 pr-6 border-r border-base-200"
				>
					Pomelo Orbit
				</router-link>

				<!-- Module tabs -->
				<nav class="flex items-stretch h-full gap-0.5">
					<button
						v-for="mod in modules"
						:key="mod.key"
						class="flex items-center gap-1.5 px-3 text-sm transition-colors cursor-pointer border-b-2"
						:class="
							currentModule === mod.key
								? 'border-primary text-primary'
								: 'border-transparent text-base-content/60 hover:text-base-content'
						"
						@click="navigateToModule(mod.key)"
					>
						<component :is="mod.icon" class="size-3.5" />
						{{ mod.label }}
					</button>
				</nav>

				<!-- Right: user dropdown -->
				<div class="ml-auto">
					<div class="dropdown dropdown-end">
						<div
							tabindex="0"
							role="button"
							class="flex items-center gap-2 cursor-pointer text-sm text-base-content/70 hover:text-base-content transition-colors px-2 py-1 rounded-btn"
						>
							<UserRound class="size-4" />
							<span>{{ authStore.user?.username || '用户' }}</span>
							<ChevronDown class="size-3" />
						</div>
						<ul
							tabindex="0"
							class="dropdown-content menu bg-base-100 rounded-box shadow-lg border border-base-200 w-36 mt-1 p-1 z-50"
						>
							<li>
								<button class="flex items-center gap-2 text-error" @click="handleLogout">
									<LogOut class="size-4" />
									退出登录
								</button>
							</li>
						</ul>
					</div>
				</div>
			</header>

			<!-- ── Body: sider + content ── -->
			<div class="flex flex-1 overflow-hidden">
				<!-- Sidebar -->
				<aside
					v-if="currentModule"
					class="shrink-0 bg-base-100 border-r border-base-200 flex flex-col transition-all duration-200 overflow-hidden"
					:class="collapsed ? 'w-14' : 'w-44'"
				>
					<ul class="menu flex-1 p-2 gap-1">
						<li v-for="item in sidebarItems" :key="item.key">
							<router-link
								:to="item.path"
								class="flex items-center gap-3 rounded-btn"
								:class="
									selectedKey === item.key
										? 'bg-primary/10 text-primary hover:bg-primary/15'
										: 'text-base-content/70 hover:text-base-content hover:bg-base-200'
								"
								:title="collapsed ? item.label : undefined"
							>
								<component :is="item.icon" class="size-4 shrink-0" />
								<span v-if="!collapsed" class="text-sm">{{ item.label }}</span>
							</router-link>
						</li>
					</ul>

					<!-- Collapse toggle -->
					<button
						class="flex items-center justify-center h-9 border-t border-base-200 text-base-content/30 hover:text-base-content/60 hover:bg-base-200 transition-colors cursor-pointer"
						@click="collapsed = !collapsed"
					>
						<PanelLeftClose v-if="!collapsed" class="size-3.5" />
						<PanelLeftOpen v-else class="size-3.5" />
					</button>
				</aside>

				<!-- Main content -->
				<main class="flex-1 overflow-y-auto bg-base-200/50 p-6">
					<router-view />
				</main>
			</div>
		</template>

		<!-- ── Global toast ── -->
		<div class="toast toast-top toast-end z-[9999]">
			<div
				v-for="t in toasts"
				:key="t.id"
				role="alert"
				class="alert shadow-md text-sm"
				:class="{
					'alert-success': t.type === 'success',
					'alert-error': t.type === 'error',
					'alert-warning': t.type === 'warning',
					'alert-info': t.type === 'info',
				}"
			>
				<span>{{ t.text }}</span>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import {
	ChevronDown,
	CloudCog,
	Code2,
	FileCode2,
	FolderGit2,
	Globe,
	History,
	KeyRound,
	Layers,
	LayoutGrid,
	LogOut,
	Network,
	Package,
	PanelLeftClose,
	PanelLeftOpen,
	Play,
	Rocket,
	Settings,
	UserRound,
} from 'lucide-vue-next';
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useToast } from '@/composables/useToast';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { toasts } = useToast();

const collapsed = ref(false);
const currentModule = ref<string | null>(null);
const selectedKey = ref('');

// ── Module definitions ──
const modules = [
	{ key: 'ci' as const, label: '持续集成', icon: Code2 },
	{ key: 'cd' as const, label: '持续部署', icon: CloudCog },
	{ key: 'settings' as const, label: '系统设置', icon: Settings },
];

// ── Sidebar items per module ──
const sidebarMap = {
	home: [{ key: 'home', label: '项目概述', path: '/', icon: LayoutGrid }],
	cd: [
		{
			key: 'applications',
			label: '应用管理',
			path: '/cd/applications',
			icon: LayoutGrid,
		},
		{
			key: 'deployments',
			label: '部署记录',
			path: '/cd/deployments',
			icon: Rocket,
		},
		{ key: 'route', label: '路由配置', path: '/cd/routes', icon: Globe },
		{
			key: 'traefik-http-routers',
			label: 'Traefik Routers',
			path: '/cd/traefik-http-routers',
			icon: Network,
		},
	],
	ci: [
		{
			key: 'repository',
			label: '代码仓库',
			path: '/ci/repository',
			icon: FolderGit2,
		},
		{
			key: 'buildstages',
			label: '构建阶段',
			path: '/ci/build-stage',
			icon: Layers,
		},
		{
			key: 'pipelinetemplates',
			label: '流水线模板',
			path: '/ci/template',
			icon: FileCode2,
		},
		{ key: 'pipelineruns', label: '流水线记录', path: '/ci/run', icon: Play },
		{
			key: 'artifacts',
			label: '制品记录',
			path: '/ci/artifact',
			icon: Package,
		},
		{
			key: 'credentials',
			label: '凭据管理',
			path: '/ci/credential',
			icon: KeyRound,
		},
	],
	settings: [
		{
			key: 'loginhistory',
			label: '登录历史',
			path: '/login-history',
			icon: History,
		},
		{ key: 'settings', label: '系统设置', path: '/settings', icon: Settings },
	],
};

const sidebarItems = computed(() => {
	if (!currentModule.value) {
		return [];
	}
	return sidebarMap[currentModule.value] ?? [];
});

const isLoginPage = computed(() => route.name === 'Login');

function syncFromRoute() {
	const path = route.path;
	if (path === '/') {
		currentModule.value = 'home';
	} else if (path.startsWith('/ci/')) {
		currentModule.value = 'ci';
	} else if (path.startsWith('/cd/')) {
		currentModule.value = 'cd';
	} else if (path === '/login-history' || path === '/settings') {
		currentModule.value = 'settings';
	} else {
		currentModule.value = null;
	}

	selectedKey.value = (route.meta.menuKey as string) ?? '';
}

router.isReady().then(syncFromRoute);
watch(() => route.path, syncFromRoute);

function navigateToModule(mod: 'ci' | 'cd' | 'settings') {
	const first = sidebarMap[mod][0];
	if (first) {
		router.push(first.path);
	}
}

async function handleLogout() {
	await authStore.logout();
	router.push('/login');
}
</script>
