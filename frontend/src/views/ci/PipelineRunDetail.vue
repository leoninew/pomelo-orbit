<template>
	<a-space direction="vertical" style="width: 100%">
		<div v-if="run" class="page-header">
			<h2>
				Pipeline Run
				<a-tag :color="pipelineRunStatusColors[run.status]" style="margin-left: 8px">
					{{ run.status }}
				</a-tag>
			</h2>
			<a-space>
				<a-button @click="$router.push('/ci/runs')">返回</a-button>
				<a-button
					v-if="run.status === 'failed' || run.status === 'success'"
					type="primary"
					@click="handleRetry"
				>
					重试
				</a-button>
				<a-popconfirm
					v-if="run.status === 'waiting' || run.status === 'running'"
					title="确定取消此 Run？"
					@confirm="handleCancel"
				>
					<a-button danger>取消</a-button>
				</a-popconfirm>
			</a-space>
		</div>

		<!-- 基本信息卡片 -->
		<a-card title="基本信息" :loading="loading">
			<a-descriptions v-if="run" :column="2" bordered size="small">
				<a-descriptions-item label="Run ID">
					{{ run.id }}
				</a-descriptions-item>
				<a-descriptions-item label="Project">
					<router-link :to="`/ci/projects/${run.project_id}`">
						{{ run.project_id }}
					</router-link>
				</a-descriptions-item>
				<a-descriptions-item label="触发方式">
					<a-tag>{{ run.trigger }}</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="Ref">
					{{ run.trigger_ref }}
				</a-descriptions-item>
				<a-descriptions-item label="状态">
					<a-tag :color="pipelineRunStatusColors[run.status]">
						{{ run.status }}
					</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="重试自">
					<router-link v-if="run.retry_of" :to="`/ci/runs/${run.retry_of}`">
						{{ run.retry_of }}
					</router-link>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="开始时间">
					{{ run.started_at ? formatTime(run.started_at) : '-' }}
				</a-descriptions-item>
				<a-descriptions-item label="结束时间">
					{{ run.finished_at ? formatTime(run.finished_at) : '-' }}
				</a-descriptions-item>
				<a-descriptions-item label="创建时间" :span="2">
					{{ formatTime(run.created_at) }}
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<!-- Jobs 卡片 -->
		<a-card title="Jobs" :loading="jobsLoading">
			<a-table
				v-if="jobs.length > 0"
				:columns="jobColumns"
				:data-source="jobs"
				:pagination="false"
				row-key="id"
			>
				<template #bodyCell="{ column, record }">
					<template v-if="column.key === 'name'">
						<a @click="showJobLogs(record)">{{ record.name }}</a>
					</template>
					<template v-else-if="column.key === 'status'">
						<a-tag :color="jobStatusColors[record.status]">
							{{ record.status }}
						</a-tag>
					</template>
					<template v-else-if="column.key === 'started_at'">
						{{ record.started_at ? formatTime(record.started_at) : '-' }}
					</template>
					<template v-else-if="column.key === 'finished_at'">
						{{ record.finished_at ? formatTime(record.finished_at) : '-' }}
					</template>
				</template>
			</a-table>
			<a-empty v-else description="暂无 Job 记录" />
		</a-card>

		<!-- 制品卡片 -->
		<a-card title="制品" :loading="artifactsLoading">
			<a-table
				v-if="artifacts.length > 0"
				:columns="artifactColumns"
				:data-source="artifacts"
				:pagination="false"
				row-key="id"
			>
				<template #bodyCell="{ column, record }">
					<template v-if="column.key === 'type'">
						<a-tag>{{ record.type }}</a-tag>
					</template>
					<template v-else-if="column.key === 'created_at'">
						{{ formatTime(record.created_at) }}
					</template>
				</template>
			</a-table>
			<a-empty v-else description="暂无制品" />
		</a-card>

		<!-- Job 日志抽屉 -->
		<a-drawer
			v-model:open="showLogsDrawer"
			:title="`Job: ${currentJob?.name || ''}`"
			:width="800"
			:loading="logsLoading"
		>
			<div v-if="currentJob">
				<a-descriptions :column="1" bordered size="small" style="margin-bottom: 16px">
					<a-descriptions-item label="状态">
						<a-tag :color="jobStatusColors[currentJob.status]">
							{{ currentJob.status }}
						</a-tag>
					</a-descriptions-item>
					<a-descriptions-item label="开始时间">
						{{ currentJob.started_at ? formatTime(currentJob.started_at) : '-' }}
					</a-descriptions-item>
					<a-descriptions-item label="结束时间">
						{{ currentJob.finished_at ? formatTime(currentJob.finished_at) : '-' }}
					</a-descriptions-item>
					<a-descriptions-item v-if="currentJob.error_message" label="错误信息">
						<pre style="margin: 0; white-space: pre-wrap; color: #ff4d4f">{{
							currentJob.error_message
						}}</pre>
					</a-descriptions-item>
				</a-descriptions>

				<div class="log-container">
					<pre v-if="logs?.content" class="log-content">{{ logsText }}</pre>
					<a-empty v-else description="暂无日志" />
				</div>
			</div>
		</a-drawer>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { jobApi, pipelineRunApi } from '@/api/ci';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Artifact, Job, JobLog, PipelineRun } from '@/types/api';
import { jobStatusColors, pipelineRunStatusColors } from '@/types/api';

const route = useRoute();
const router = useRouter();
const runId = route.params.id as string;

const { loading, execute } = useStatusAsync();
const { loading: jobsLoading, execute: executeJobs } = useStatusAsync();
const { loading: artifactsLoading, execute: executeArtifacts } = useStatusAsync();
const { loading: logsLoading, execute: executeLogs } = useStatusAsync();

const run = ref<PipelineRun>();
const jobs = ref<Job[]>([]);
const artifacts = ref<Artifact[]>([]);
const logs = ref<JobLog | null>(null);
const currentJob = ref<Job>();
const showLogsDrawer = ref(false);

const logsText = computed(() => logs.value?.content ?? '');

const jobColumns = [
	{ title: 'Job 名称', key: 'name', dataIndex: 'name' },
	{ title: '状态', key: 'status', width: 120 },
	{ title: '开始时间', key: 'started_at', width: 180 },
	{ title: '结束时间', key: 'finished_at', width: 180 },
];

const artifactColumns = [
	{ title: 'Job', key: 'job_name', dataIndex: 'job_name', width: 200 },
	{ title: '类型', key: 'type', width: 150 },
	{ title: '名称', key: 'name', dataIndex: 'name' },
	{ title: '路径', key: 'path', dataIndex: 'path', ellipsis: true },
	{ title: '创建时间', key: 'created_at', width: 180 },
];

async function fetchRun() {
	try {
		await execute(async () => {
			const data = await pipelineRunApi.get(runId);
			run.value = data;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取 Run 信息失败');
		router.push('/ci/runs');
	}
}

async function fetchJobs() {
	try {
		await executeJobs(async () => {
			const data = await pipelineRunApi.listJobs(runId);
			jobs.value = data;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取 Jobs 失败');
	}
}

async function fetchArtifacts() {
	try {
		await executeArtifacts(async () => {
			const data = await pipelineRunApi.listArtifacts(runId);
			artifacts.value = data;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取制品失败');
	}
}

async function showJobLogs(job: Job) {
	currentJob.value = job;
	showLogsDrawer.value = true;
	logs.value = null;

	try {
		await executeLogs(async () => {
			logs.value = await jobApi.listLogs(job.id);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取日志失败');
	}
}

async function handleRetry() {
	try {
		const newRun = await pipelineRunApi.retry(runId);
		message.success(`重试成功，新 Run ID: ${newRun.id}`);
		router.push(`/ci/runs/${newRun.id}`);
	} catch (error) {
		message.error(error instanceof Error ? error.message : '重试失败');
	}
}

async function handleCancel() {
	try {
		await pipelineRunApi.cancel(runId);
		message.success('已取消');
		fetchRun();
	} catch (error) {
		message.error(error instanceof Error ? error.message : '取消失败');
	}
}

onMounted(() => {
	fetchRun();
	fetchJobs();
	fetchArtifacts();
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
	display: flex;
	align-items: center;
}

.log-container {
	background: #1e1e1e;
	border-radius: 4px;
	padding: 16px;
	max-height: calc(100vh - 400px);
	overflow-y: auto;
}

.log-content {
	color: #d4d4d4;
	font-family: 'Consolas', 'Monaco', monospace;
	font-size: 13px;
	line-height: 1.5;
	margin: 0;
	white-space: pre-wrap;
	word-wrap: break-word;
}
</style>
