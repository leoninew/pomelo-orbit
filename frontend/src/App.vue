<template>
	<div class="h-screen flex flex-col font-sans">
		<!-- Login page: no layout chrome -->
		<template v-if="isLoginPage">
			<router-view />
		</template>

		<template v-else>
			<!-- ── Top navbar ── -->
			<header class="navbar bg-base-100 border-b border-base-200 h-16 min-h-16 shrink-0 px-4 gap-0">
				<!-- Logo -->
				<router-link
					to="/"
					class="text-lg font-bold text-base-content hover:text-primary transition-colors mr-6 pr-6 border-r border-base-200"
				>
					Pomelo Orbit
				</router-link>

				<!-- Module tabs -->
				<nav class="flex items-stretch h-full gap-1">
					<button
						v-for="mod in modules"
						:key="mod.key"
						class="flex items-center gap-1.5 px-4 text-sm border-b-2 transition-colors cursor-pointer"
						:class="
							currentModule === mod.key
								? 'border-primary text-primary'
								: 'border-transparent text-base-content/70 hover:text-base-content'
						"
						@click="navigateToModule(mod.key)"
					>
						<component :is="mod.icon" class="size-4" />
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
					class="shrink-0 bg-base-100 border-r border-base-200 flex flex-col transition-all duration-200 overflow-hidden"
					:class="collapsed ? 'w-14' : 'w-52'"
				>
					<ul class="menu flex-1 p-2 gap-0.5">
						<li v-for="item in sidebarItems" :key="item.key">
							<router-link
								:to="item.path"
								class="flex items-center gap-3 rounded-btn"
								:class="selectedKey === item.key ? 'active' : ''"
								:title="collapsed ? item.label : undefined"
							>
								<component :is="item.icon" class="size-4 shrink-0" />
								<span v-if="!collapsed" class="text-sm">{{ item.label }}</span>
							</router-link>
						</li>
					</ul>

					<!-- Collapse toggle -->
					<button
						class="flex items-center justify-center h-10 border-t border-base-200 text-base-content/40 hover:text-base-content hover:bg-base-200 transition-colors cursor-pointer"
						@click="collapsed = !collapsed"
					>
						<ChevronLeft v-if="!collapsed" class="size-4" />
						<ChevronRight v-else class="size-4" />
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
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
	ChevronDown,
	ChevronLeft,
	ChevronRight,
	Code2,
	CloudCog,
	Settings,
	LayoutGrid,
	Rocket,
	Globe,
	Network,
	FolderGit2,
	Play,
	FileCode2,
	KeyRound,
	History,
	UserRound,
	LogOut,
} from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';
import { useToast } from '@/composables/useToast';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { toasts } = useToast();

const collapsed = ref(false);
const currentModule = ref<'ci' | 'cd' | 'settings' | null>(null);
const selectedKey = ref('');

// ── Module definitions ──
const modules = [
	{ key: 'ci' as const, label: '持续集成', icon: Code2 },
	{ key: 'cd' as const, label: '持续部署', icon: CloudCog },
	{ key: 'settings' as const, label: '系统设置', icon: Settings },
];

// ── Sidebar items per module ──
const sidebarMap = {
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
		{ key: 'projects', label: '项目管理', path: '/ci/projects', icon: FolderGit2 },
		{ key: 'pipelineruns', label: '流水线记录', path: '/ci/runs', icon: Play },
		{ key: 'pipelinetemplates', label: '流水线模板', path: '/ci/templates', icon: FileCode2 },
		{ key: 'credentials', label: '凭据管理', path: '/ci/credentials', icon: KeyRound },
	],
	settings: [
		{ key: 'loginhistory', label: '登录历史', path: '/login-history', icon: History },
		{ key: 'settings', label: '系统设置', path: '/settings', icon: Settings },
	],
};

const sidebarItems = computed(() => {
	if (!currentModule.value) return [];
	return sidebarMap[currentModule.value] ?? [];
});

const isLoginPage = computed(() => route.name === 'Login');

function syncFromRoute() {
	const path = route.path;
	if (path.startsWith('/ci/')) currentModule.value = 'ci';
	else if (path.startsWith('/cd/')) currentModule.value = 'cd';
	else if (path === '/login-history' || path === '/settings') currentModule.value = 'settings';
	else currentModule.value = 'cd';

	selectedKey.value = (route.meta.menuKey as string) ?? '';
}

router.isReady().then(syncFromRoute);
watch(() => route.path, syncFromRoute);

function navigateToModule(mod: 'ci' | 'cd' | 'settings') {
	const first = sidebarMap[mod][0];
	if (first) router.push(first.path);
}

async function handleLogout() {
	await authStore.logout();
	router.push('/login');
}
</script>
