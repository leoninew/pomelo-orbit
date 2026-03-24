<template>
	<a-config-provider>
		<div id="app">
			<!-- 登录页不显示布局 -->
			<template v-if="isLoginPage">
				<router-view />
			</template>
			<template v-else>
				<a-layout style="min-height: 100vh">
					<a-layout-sider
						v-model:collapsed="collapsed"
						collapsible
						:style="{ background: '#001529' }"
					>
						<router-link to="/" class="logo">
							<span v-if="!collapsed">{{ projectTitle }}</span>
							<span v-else>{{ projectShort }}</span>
						</router-link>
						<a-menu v-model:selected-keys="selectedKeys" mode="inline" theme="dark">
							<a-menu-item key="applications" @click="navigate('/applications')">
								<template #icon><AppstoreOutlined /></template>
								应用管理
							</a-menu-item>
							<a-menu-item key="deployments" @click="navigate('/deployments')">
								<template #icon><CloudServerOutlined /></template>
								部署记录
							</a-menu-item>
							<a-sub-menu key="routes">
								<template #icon><GlobalOutlined /></template>
								<template #title>路由配置</template>
								<a-menu-item key="route" @click="navigate('/routes')">路由配置</a-menu-item>
								<a-menu-item key="traefik-http-routers" @click="navigate('/traefik-http-routers')">
									Traefik HTTP Routers
								</a-menu-item>
							</a-sub-menu>
							<a-menu-item key="events" @click="navigate('/events')">
								<template #icon><ApiOutlined /></template>
								回调事件
							</a-menu-item>
							<!-- <a-menu-item key="credentials" @click="navigate('/credentials')">
								<template #icon><KeyOutlined /></template>
								凭据管理
							</a-menu-item> -->
							<a-menu-item key="loginhistory" @click="navigate('/login-history')">
								<template #icon><HistoryOutlined /></template>
								登录历史
							</a-menu-item>
							<a-menu-item key="settings" @click="navigate('/settings')">
								<template #icon><SettingOutlined /></template>
								系统设置
							</a-menu-item>
						</a-menu>
					</a-layout-sider>
					<a-layout>
						<a-layout-header
							:style="{ background: '#fff', color: 'rgba(0, 0, 0, 0.88)' }"
							class="header"
						>
							<div class="header-right">
								<a-dropdown>
									<a-space style="cursor: pointer">
										<UserOutlined />
										<span>{{ authStore.user?.username || '用户' }}</span>
									</a-space>
									<template #overlay>
										<a-menu>
											<a-menu-item key="settings" @click="navigate('/settings')">
												<SettingOutlined />
												系统设置
											</a-menu-item>
											<a-menu-divider />
											<a-menu-item key="logout" @click="handleLogout">
												<LogoutOutlined />
												退出登录
											</a-menu-item>
										</a-menu>
									</template>
								</a-dropdown>
							</div>
						</a-layout-header>
						<a-layout-content style="padding: 24px; background: #f0f2f5">
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
const selectedKeys = ref<string[]>(['applications']);
const projectTitle = 'Pomelo Orbit';
const projectShort = 'PO';

const isLoginPage = computed(() => route.name === 'Login');

// 根据路由更新选中菜单
watch(
	() => route.name,
	(name) => {
		if (name && typeof name === 'string') {
			const key = name.toLowerCase();
			if (key === 'home') {
				selectedKeys.value = [];
			} else if (
				[
					'applications',
					'deployments',
					'events',
					'route',
					'traefik-http-routers',
					// 'credentials',
					'loginhistory',
					'settings',
				].includes(key)
			) {
				selectedKeys.value = [key];
			} else if (key === 'applicationdetail') {
				selectedKeys.value = ['applications'];
			} else if (key === 'deploymentdetail') {
				selectedKeys.value = ['deployments'];
			} else if (key === 'eventdetail') {
				selectedKeys.value = ['events'];
			} else if (key === 'routedetail') {
				selectedKeys.value = ['route'];
			}
		}
	},
	{ immediate: true }
);

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

.logo {
	height: 64px;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 20px;
	font-weight: bold;
	background: rgba(255, 255, 255, 0.1);
	text-decoration: none;
}

.logo:hover {
	color: white;
}

.header {
	background: #fff;
	padding: 0 24px;
	display: flex;
	justify-content: flex-end;
	align-items: center;
	box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
	color: rgba(0, 0, 0, 0.88);
}

.header-right {
	display: flex;
	align-items: center;
	color: rgba(0, 0, 0, 0.88);
}

.header-right .anticon,
.header-right span {
	color: rgba(0, 0, 0, 0.88);
}
</style>
