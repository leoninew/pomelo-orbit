<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>仪表盘</h2>
			<a-button @click="refresh">
				<template #icon><ReloadOutlined /></template>
				刷新
			</a-button>
		</div>

		<a-row :gutter="16">
			<a-col :span="6">
				<a-card :loading="loading">
					<a-statistic title="应用总数" :value="stats.projectCount">
						<template #prefix>
							<AppstoreOutlined />
						</template>
					</a-statistic>
				</a-card>
			</a-col>
			<a-col :span="6">
				<a-card :loading="loading">
					<a-statistic title="今日部署" :value="stats.todayDeploys">
						<template #prefix>
							<CloudServerOutlined />
						</template>
					</a-statistic>
				</a-card>
			</a-col>
			<a-col :span="6">
				<a-card :loading="loading">
					<a-statistic
						title="运行中"
						:value="stats.runningDeploys"
						:value-style="{ color: '#1890ff' }"
					>
						<template #prefix>
							<SyncOutlined spin />
						</template>
					</a-statistic>
				</a-card>
			</a-col>
			<a-col :span="6">
				<a-card :loading="loading">
					<a-statistic title="今日事件" :value="stats.todayEvents">
						<template #prefix>
							<ApiOutlined />
						</template>
					</a-statistic>
				</a-card>
			</a-col>
		</a-row>

		<a-row :gutter="16">
			<a-col :span="12">
				<a-card title="最近部署" :loading="loading">
					<a-table
						:columns="deployColumns"
						:data-source="recentDeploys"
						:pagination="false"
						size="small"
						row-key="id"
					>
						<template #bodyCell="{ column, record }">
							<template v-if="column.key === 'status'">
								<a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
							</template>
							<template v-else-if="column.key === 'started_at'">
								{{ formatTime(record.started_at) }}
							</template>
							<template v-else-if="column.key === 'actions'">
								<router-link :to="`/deployments/${record.id}`">详情</router-link>
							</template>
						</template>
					</a-table>
				</a-card>
			</a-col>
			<a-col :span="12">
				<a-card title="最近事件" :loading="loading">
					<a-table
						:columns="eventColumns"
						:data-source="recentEvents"
						:pagination="false"
						size="small"
						row-key="id"
					>
						<template #bodyCell="{ column, record }">
							<template v-if="column.key === 'source'">
								<a-tag :color="record.source === 'github' ? 'blue' : 'orange'">
									{{ record.source }}
								</a-tag>
							</template>
							<template v-else-if="column.key === 'status'">
								<a-tag :color="getEventStatusColor(record.status)">{{ record.status }}</a-tag>
							</template>
							<template v-else-if="column.key === 'received_at'">
								{{ formatTime(record.received_at) }}
							</template>
							<template v-else-if="column.key === 'actions'">
								<router-link :to="`/events/${record.id}`">详情</router-link>
							</template>
						</template>
					</a-table>
				</a-card>
			</a-col>
		</a-row>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { formatTime, getTodayStart } from '@/utils/time';
import { onMounted, reactive, ref } from 'vue';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { eventApi } from '@/api/event';
import { useStatusAsync } from '@/composables/useStatusAsync';
import {
	type Deployment,
	deploymentStatusColors,
	eventStatusColors,
	type WebhookEvent,
} from '@/types/api';

const { loading, execute } = useStatusAsync();

const stats = reactive({
	projectCount: 0,
	todayDeploys: 0,
	runningDeploys: 0,
	todayEvents: 0,
});

const recentDeploys = ref<Deployment[]>([]);
const recentEvents = ref<WebhookEvent[]>([]);

const deployColumns = [
	{ title: '状态', key: 'status', width: 70 },
	{ title: '触发', dataIndex: 'trigger_type', width: 60 },
	{ title: '时间', key: 'started_at', width: 140 },
	{ title: '操作', key: 'actions', width: 50 },
];

const eventColumns = [
	{ title: '来源', key: 'source', width: 60 },
	{ title: '状态', key: 'status', width: 70 },
	{ title: '时间', key: 'received_at', width: 140 },
	{ title: '操作', key: 'actions', width: 50 },
];

function getStatusColor(status: string) {
	return deploymentStatusColors[status] || 'default';
}

function getEventStatusColor(status: string) {
	return eventStatusColors[status] || 'default';
}

async function fetchStats() {
	try {
		await execute(async () => {
			const todayStart = getTodayStart();
			const todayEnd = todayStart.add(1, 'day');

			const applicationsRes = await applicationApi.list({ per_page: 1 });
			stats.projectCount = applicationsRes.total;

			const deploysRes = await deploymentApi.list({
				per_page: 100,
				date_from: todayStart.toISOString(),
				date_to: todayEnd.toISOString(),
			});
			stats.todayDeploys = deploysRes.total;
			stats.runningDeploys = deploysRes.items.filter(
				(d) => d.status === 'running' || d.status === 'queued'
			).length;

			const eventsRes = await eventApi.list({
				per_page: 100,
				date_from: todayStart.toISOString(),
				date_to: todayEnd.toISOString(),
			});
			stats.todayEvents = eventsRes.total;

			const recentDeploysRes = await deploymentApi.list({ per_page: 5 });
			recentDeploys.value = recentDeploysRes.items;

			const recentEventsRes = await eventApi.list({ per_page: 5 });
			recentEvents.value = recentEventsRes.items;
		});
	} catch (error) {
		message.error('获取数据失败\n' + error);
	}
}

function refresh() {
	fetchStats();
}

onMounted(() => {
	refresh();
});
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.page-header h2 {
	margin: 0;
}
</style>
