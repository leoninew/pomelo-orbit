<template>
	<a-config-provider>
		<div id="app">
			<template v-if="isLoginPage">
				<router-view />
			</template>
			<template v-else>
				<a-layout style="min-height: 100vh">
					<!-- 顶部固定 Header -->
					<div class="top-header">
						<router-link to="/" class="logo">{{ projectTitle }}</router-link>
						<div class="module-nav">
							<div
								class="module-tab"
								:class="{ active: currentModule === 'ci' }"
								@click="navigateToModule('ci')"
							>
								<CodeOutlined />
								持续集成
							</div>
							<div
								class="module-tab"
								:class="{ active: currentModule === 'cd' }"
								@click="navigateToModule('cd')"
							>
								<CloudServerOutlined />
								持续部署
							</div>
							<div
								class="module-tab"
								:class="{ active: currentModule === 'settings' }"
								@click="navigateToModule('settings')"
							>
								<SettingOutlined />
								系统设置
							</div>
						</div>
						<div class="header-right">
							<a-dropdown>
								<a-space style="cursor: pointer">
									<UserOutlined />
									<span>{{ authStore.user?.username || '用户' }}</span>
								</a-space>
								<template #overlay>
									<a-menu>
										<a-menu-item key="logout" @click="handleLogout">
											<LogoutOutlined />
											退出登录
										</a-menu-item>
									</a-menu>
								</template>
							</a-dropdown>
						</div>
					</div>

					<!-- Header 下方：Sider + Content -->
					<a-layout style="height: calc(100vh - 64px); overflow: hidden">
						<a-layout-sider
							v-model:collapsed="collapsed"
							:trigger="null"
							collapsible
							theme="light"
							:width="220"
							style="border-right: 1px solid #f0f0f0"
						>
							<a-menu
								v-if="currentModule === 'cd'"
								v-model:selected-keys="selectedKeys"
								mode="inline"
								theme="light"
								style="height: 100%; border-right: 0"
							>
								<a-menu-item key="applications" @click="navigate('/cd/applications')">
									<template #icon><AppstoreOutlined /></template>
									应用管理
								</a-menu-item>
								<a-menu-item key="deployments" @click="navigate('/cd/deployments')">
									<template #icon><DeploymentUnitOutlined /></template>
									部署记录
								</a-menu-item>
								<a-menu-item key="route" @click="navigate('/cd/routes')">
									<template #icon><GlobalOutlined /></template>
									路由配置
								</a-menu-item>
								<a-menu-item
									key="traefik-http-routers"
									@click="navigate('/cd/traefik-http-routers')"
								>
									<template #icon><ApiOutlined /></template>
									Traefik Routers
								</a-menu-item>
							</a-menu>

							<a-menu
								v-else-if="currentModule === 'ci'"
								v-model:selected-keys="selectedKeys"
								mode="inline"
								theme="light"
								style="height: 100%; border-right: 0"
							>
								<a-menu-item key="projects" @click="navigate('/ci/projects')">
									<template #icon><ProjectOutlined /></template>
									项目管理
								</a-menu-item>
								<a-menu-item key="pipelineruns" @click="navigate('/ci/runs')">
									<template #icon><PlayCircleOutlined /></template>
									流水线记录
								</a-menu-item>
								<a-menu-item key="pipelinetemplates" @click="navigate('/ci/templates')">
									<template #icon><FileTextOutlined /></template>
									流水线模板
								</a-menu-item>
								<a-menu-item key="credentials" @click="navigate('/ci/credentials')">
									<template #icon><KeyOutlined /></template>
									凭据管理
								</a-menu-item>
							</a-menu>

							<a-menu
								v-else-if="currentModule === 'settings'"
								v-model:selected-keys="selectedKeys"
								mode="inline"
								theme="light"
								style="height: 100%; border-right: 0"
							>
								<a-menu-item key="loginhistory" @click="navigate('/login-history')">
									<template #icon><HistoryOutlined /></template>
									登录历史
								</a-menu-item>
								<a-menu-item key="settings" @click="navigate('/settings')">
									<template #icon><SettingOutlined /></template>
									系统设置
								</a-menu-item>
							</a-menu>

							<div class="sider-collapse-btn" @click="collapsed = !collapsed">
								<LeftOutlined v-if="!collapsed" />
								<RightOutlined v-else />
							</div>
						</a-layout-sider>

						<a-layout-content
							style="
								padding: 24px;
								background: #f5f5f5;
								overflow-y: auto;
								display: flex;
								flex-direction: column;
							"
						>
							<router-view />
						</a-layout-content>
					</a-layout>
				</a-layout>
			</template>
		</div>
	</a-config-provider>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const collapsed = ref(false);
const selectedKeys = ref<string[]>([]);
const currentModule = ref<'cd' | 'ci' | 'settings' | null>(null);
const projectTitle = 'Pomelo Orbit';

const isLoginPage = computed(() => route.name === 'Login');

function syncModule() {
	if (route.path.startsWith('/ci/')) {
		currentModule.value = 'ci';
	} else if (route.path.startsWith('/cd/')) {
		currentModule.value = 'cd';
	} else if (route.path === '/login-history' || route.path === '/settings') {
		currentModule.value = 'settings';
	} else {
		currentModule.value = 'cd';
	}
	selectedKeys.value = route.meta.menuKey ? [route.meta.menuKey as string] : [];
}

// 等路由就绪后初始化，避免刷新时短暂显示错误模块
router.isReady().then(syncModule);

watch(() => route.path, syncModule);

function navigateToModule(module: 'cd' | 'ci' | 'settings') {
	if (module === 'ci') router.push('/ci/projects');
	else if (module === 'cd') router.push('/cd/applications');
	else router.push('/login-history');
}

function navigate(path: string) {
	router.push(path);
}

async function handleLogout() {
	await authStore.logout();
	router.push('/login');
}
</script>

<style>
#app {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.top-header {
	height: 64px;
	padding: 0 20px;
	background: #fff;
	border-bottom: 1px solid #f0f0f0;
	display: flex;
	align-items: center;
	gap: 8px;
	position: sticky;
	top: 0;
	z-index: 100;
	flex-shrink: 0;
}

.logo {
	color: rgba(0, 0, 0, 0.88);
	font-size: 20px;
	font-weight: 600;
	text-decoration: none;
	white-space: nowrap;
	margin-right: 24px;
	padding-right: 24px;
	border-right: 1px solid #f0f0f0;
}

.logo:hover {
	color: #1677ff;
}

.module-nav {
	display: flex;
	align-items: stretch;
	gap: 4px;
	height: 100%;
}

.module-tab {
	display: flex;
	align-items: center;
	gap: 6px;
	padding: 0 14px;
	color: rgba(0, 0, 0, 0.88);
	cursor: pointer;
	font-size: 14px;
	border-bottom: 2px solid transparent;
	margin-bottom: -1px;
	transition:
		color 0.15s,
		border-color 0.15s;
}

.module-tab:hover {
	color: #1677ff;
}

.module-tab.active {
	color: #1677ff;
	border-bottom-color: #1677ff;
}

.header-right {
	display: flex;
	align-items: center;
	gap: 16px;
	color: rgba(0, 0, 0, 0.65);
	margin-left: auto;
	font-size: 13px;
}

.header-right .anticon,
.header-right span {
	color: rgba(0, 0, 0, 0.65);
}

/* sider 选中项：去圆角，水平充满，左侧竖线，蓝色 */
.ant-layout-sider-light .ant-menu-light .ant-menu-item-selected {
	border-radius: 0 !important;
	margin-inline: 0 !important;
	width: 100% !important;
	background-color: #f5f5f5 !important;
	color: #1677ff !important;
	border-left: 2px solid #1677ff !important;
}

.ant-layout-sider-light .ant-menu-light .ant-menu-item-selected .anticon {
	color: #1677ff !important;
}

/* sider hover：只加背景 */
.ant-layout-sider-light .ant-menu-light .ant-menu-item:not(.ant-menu-item-selected):hover {
	border-radius: 0 !important;
	margin-inline: 0 !important;
	width: 100% !important;
	background-color: #fafafa !important;
}

.sider-collapse-btn {
	position: absolute;
	bottom: 0;
	width: 100%;
	height: 40px;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	color: rgba(0, 0, 0, 0.35);
	border-top: 1px solid #f0f0f0;
	transition:
		color 0.15s,
		background 0.15s;
}

.sider-collapse-btn:hover {
	color: rgba(0, 0, 0, 0.65);
	background: #fafafa;
}
</style>
