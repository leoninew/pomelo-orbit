<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>Traefik HTTP Routers</h2>
			<a-space>
				<a-button type="primary" @click="openDashboard">
					<template #icon><LinkOutlined /></template>
					打开 Dashboard
				</a-button>
				<a-button :loading="loading" @click="fetchRoutes">
					<template #icon><ReloadOutlined /></template>
					刷新
				</a-button>
			</a-space>
		</div>

		<a-table
			:columns="columns"
			:data-source="routes"
			:loading="loading"
			:pagination="false"
			row-key="name"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'rule'">
					<a
						v-if="buildRouteUrl(record.rule, record.tls)"
						:href="buildRouteUrl(record.rule, record.tls)!"
						target="_blank"
					>
						{{ record.rule }}
						<LinkOutlined />
					</a>
					<span v-else>{{ record.rule }}</span>
				</template>
				<template v-if="column.key === 'status'">
					<a-tag :color="record.status === 'enabled' ? 'success' : 'default'">
						{{ record.status }}
					</a-tag>
				</template>
				<template v-if="column.key === 'entrypoints'">
					<a-tag v-for="ep in record.entrypoints" :key="ep">{{ ep }}</a-tag>
				</template>
				<template v-if="column.key === 'tls'">
					<a-tag :color="record.tls ? 'blue' : 'default'">
						{{ record.tls ? 'HTTPS' : 'HTTP' }}
					</a-tag>
				</template>
			</template>
		</a-table>
	</a-space>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import { traefikRouteApi, type TraefikRouter } from '@/api/traefik-route';
import { useStatusAsync } from '@/composables/useStatusAsync';

const routes = ref<TraefikRouter[]>([]);
const { loading, execute } = useStatusAsync();

const columns = [
	{ title: '名称', dataIndex: 'name', key: 'name' },
	{ title: '提供者', dataIndex: 'provider', key: 'provider', width: 100 },
	{ title: '状态', dataIndex: 'status', key: 'status', width: 100 },
	{ title: '规则', dataIndex: 'rule', key: 'rule' },
	{ title: '服务', dataIndex: 'service', key: 'service' },
	{ title: '入口点', dataIndex: 'entrypoints', key: 'entrypoints', width: 150 },
	{ title: '协议', dataIndex: 'tls', key: 'tls', width: 80 },
];

async function fetchRoutes() {
	try {
		await execute(async () => {
			const data = await traefikRouteApi.list();
			routes.value = data.items;
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '获取路由失败');
	}
}

function buildRouteUrl(rule: string, tls: boolean): string | null {
	const match = rule.match(/Host\(`([^`]+)`\)/);
	if (!match) return null;
	const protocol = tls ? 'https' : 'http';
	return `${protocol}://${match[1]}`;
}

async function openDashboard() {
	try {
		const config = await traefikRouteApi.getConfig();
		const protocol = config.https_enabled ? 'https' : 'http';
		window.open(`${protocol}://${config.dashboard_domain}/dashboard/`, '_blank');
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '打开 Dashboard 失败');
	}
}

onMounted(() => {
	fetchRoutes();
});
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	min-height: 32px;
}
</style>
