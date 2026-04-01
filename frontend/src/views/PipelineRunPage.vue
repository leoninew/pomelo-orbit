<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>Pipeline Runs</h2>
		</div>

		<a-table
			:columns="columns"
			:data-source="runs"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'id'">
					<router-link :to="`/ci/runs/${record.id}`">
						{{ record.id.substring(0, 12) }}
					</router-link>
				</template>
				<template v-else-if="column.key === 'project_id'">
					<router-link :to="`/ci/projects/${record.project_id}`">
						{{ record.project_id.substring(0, 8) }}
					</router-link>
				</template>
				<template v-else-if="column.key === 'trigger'">
					<a-tag>{{ record.trigger }}</a-tag>
				</template>
				<template v-else-if="column.key === 'status'">
					<a-tag :color="pipelineRunStatusColors[record.status]">
						{{ record.status }}
					</a-tag>
				</template>
				<template v-else-if="column.key === 'retry_of'">
					<router-link v-if="record.retry_of" :to="`/ci/runs/${record.retry_of}`">
						{{ record.retry_of.substring(0, 8) }}
					</router-link>
					<span v-else>-</span>
				</template>
				<template v-else-if="column.key === 'created_at'">
					{{ formatTime(record.created_at) }}
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-space>
						<a @click="$router.push(`/ci/runs/${record.id}`)">查看</a>
						<a
							v-if="record.status === 'failed' || record.status === 'success'"
							@click="handleRetry(record.id)"
						>
							重试
						</a>
					</a-space>
				</template>
			</template>
		</a-table>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineRunApi } from '@/api/ci';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { PipelineRun } from '@/types/api';
import { pipelineRunStatusColors } from '@/types/api';

const route = useRoute();
const router = useRouter();

const { loading, execute } = useStatusAsync();

const runs = ref<PipelineRun[]>([]);
const pagination = reactive({
	current: 1,
	pageSize: 20,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: 'Run ID', key: 'id', width: 140 },
	{ title: 'Project', key: 'project_id', width: 120 },
	{ title: '触发方式', key: 'trigger', width: 100 },
	{ title: 'Ref', key: 'trigger_ref', dataIndex: 'trigger_ref', width: 150 },
	{ title: '状态', key: 'status', width: 100 },
	{ title: '重试自', key: 'retry_of', width: 120 },
	{ title: '创建时间', key: 'created_at', width: 180 },
	{ title: '操作', key: 'actions', width: 150 },
];

async function fetchRuns() {
	try {
		await execute(async () => {
			const projectId = route.query.project_id as string | undefined;
			const res = await pipelineRunApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				project_id: projectId,
			});
			runs.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取运行记录失败');
	}
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 20;
	fetchRuns();
}

async function handleRetry(runId: string) {
	try {
		const newRun = await pipelineRunApi.retry(runId);
		message.success(`重试成功，新 Run ID: ${newRun.id}`);
		router.push(`/ci/runs/${newRun.id}`);
	} catch (error) {
		message.error(error instanceof Error ? error.message : '重试失败');
	}
}

onMounted(() => {
	fetchRuns();
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
