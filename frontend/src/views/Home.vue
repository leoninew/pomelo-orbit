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
			<a-col :span="8">
				<a-card :loading="loading">
					<a-statistic title="应用总数" :value="stats.projectCount">
						<template #prefix><AppstoreOutlined /></template>
					</a-statistic>
				</a-card>
			</a-col>
			<a-col :span="8">
				<a-card :loading="loading">
					<a-statistic title="今日部署" :value="stats.todayDeploys">
						<template #prefix><CloudServerOutlined /></template>
					</a-statistic>
				</a-card>
			</a-col>
			<a-col :span="8">
				<a-card :loading="loading">
					<a-statistic
						title="运行中"
						:value="stats.runningDeploys"
						:value-style="{ color: '#1890ff' }"
					>
						<template #prefix><SyncOutlined spin /></template>
					</a-statistic>
				</a-card>
			</a-col>
		</a-row>

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
						<a-tag :color="deploymentStatusColors[record.status] || 'default'">
							{{ record.status }}
						</a-tag>
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
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { formatTime, getTodayStart } from '@/utils/time';
import { onMounted, reactive, ref } from 'vue';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { type Deployment, deploymentStatusColors } from '@/types/api';

const { loading, execute } = useStatusAsync();

const stats = reactive({
	projectCount: 0,
	todayDeploys: 0,
	runningDeploys: 0,
});

const recentDeploys = ref<Deployment[]>([]);

const deployColumns = [
	{ title: '状态', key: 'status', width: 80 },
	{ title: '触发', dataIndex: 'trigger_type', width: 70 },
	{ title: '时间', key: 'started_at', width: 160 },
	{ title: '操作', key: 'actions', width: 60 },
];

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

			const recentDeploysRes = await deploymentApi.list({ per_page: 5 });
			recentDeploys.value = recentDeploysRes.items;
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
	min-height: 32px;
}

.page-header h2 {
	margin: 0;
}
</style>
