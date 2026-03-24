<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>部署记录</h2>
			<a-input-search
				v-model:value="searchText"
				placeholder="搜索应用名称"
				style="width: 200px"
				@search="handleSearch"
			/>
		</div>

		<a-table
			:columns="columns"
			:data-source="deployments"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'application'">
					<router-link :to="`/applications/${record.application_id}`">
						{{ record.application_name || record.application_id }}
					</router-link>
				</template>
				<template v-else-if="column.key === 'status'">
					<a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
				</template>
				<template v-else-if="column.key === 'started_at'">
					{{ formatTime(record.started_at) }}
				</template>
				<template v-else-if="column.key === 'duration'">
					{{ formatDuration(record.duration_ms) }}
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-button-group>
						<a-button
							type="link"
							size="small"
							block
							@click="$router.push(`/deployments/${record.id}`)"
						>
							查看
						</a-button>
						<a-button
							v-if="record.status === 'running' || record.status === 'queued'"
							type="link"
							size="small"
							danger
							block
							@click="handleCancel(record.id)"
						>
							取消
						</a-button>
					</a-button-group>
				</template>
			</template>
		</a-table>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { formatTime } from '@/utils/time';
import { type Deployment, deploymentStatusColors } from '@/types/api';

const route = useRoute();
const { loading, execute } = useStatusAsync();
const deployments = ref<Deployment[]>([]);
const searchText = ref('');
const applicationId = ref<string | undefined>(route.query.application_id as string | undefined);

const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: '应用', key: 'application', width: 150 },
	{ title: '操作类型', dataIndex: 'operation_type', width: 80 },
	{ title: '触发方式', dataIndex: 'trigger_type', width: 80 },
	{ title: '分支/Tag', dataIndex: 'trigger_ref', width: 100 },
	{ title: '环境文件', dataIndex: 'env_file', width: 100 },
	{ title: '状态', key: 'status', width: 80 },
	{ title: '开始时间', key: 'started_at', width: 160 },
	{ title: '耗时', key: 'duration', width: 80 },
	{ title: '操作', key: 'actions', width: 100 },
];

function formatDuration(ms?: number | null) {
	if (!ms) return '-';
	if (ms < 1000) return `${ms}ms`;
	if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
	return `${(ms / 60000).toFixed(1)}min`;
}

function getStatusColor(status: string) {
	return deploymentStatusColors[status] || 'default';
}

async function fetchDeployments() {
	try {
		await execute(async () => {
			const res = await deploymentApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined,
				application_id: applicationId.value,
			});
			deployments.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error('获取部署记录列表失败\n' + error);
	}
}

function handleSearch() {
	pagination.current = 1;
	fetchDeployments();
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 20;
	fetchDeployments();
}

async function handleCancel(id: string) {
	try {
		await execute(async () => {
			await deploymentApi.cancel(id);
			message.success('已取消部署');
			fetchDeployments();
		});
	} catch (error) {
		message.error('取消失败\n' + error);
	}
}

onMounted(() => {
	fetchDeployments();
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
