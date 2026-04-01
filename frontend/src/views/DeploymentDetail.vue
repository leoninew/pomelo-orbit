<template>
	<div class="page-wrapper">
		<div v-if="deployment" class="page-header">
			<h2>部署记录 #{{ deploymentId }}</h2>
			<a-space>
				<a-button @click="() => $router.push('/deployments')">返回列表</a-button>
				<a-button @click="() => $router.push(`/applications/${deployment.application_id}`)">
					返回应用
				</a-button>
				<a-button
					v-if="deployment.status === 'running' || deployment.status === 'queued'"
					danger
					@click="handleCancel"
				>
					取消部署
				</a-button>
			</a-space>
		</div>

		<a-card title="基本信息" :loading="loading">
			<a-descriptions v-if="deployment" :column="3" bordered size="small">
				<a-descriptions-item label="应用">
					<router-link :to="`/applications/${deployment.application_id}`">
						{{ deployment.application_name || deployment.application_id }}
					</router-link>
				</a-descriptions-item>
				<a-descriptions-item label="状态">
					<a-tag :color="getStatusColor(deployment.status)">
						{{ deployment.status }}
					</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="触发方式">
					{{ deployment.trigger_type }}
				</a-descriptions-item>
				<a-descriptions-item label="分支/Tag">
					<span v-if="deployment.trigger_ref">{{ deployment.trigger_ref }}</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="环境文件">
					<span v-if="deployment.env_file">{{ deployment.env_file }}</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="开始时间">
					{{ formatTime(deployment.started_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="结束时间">
					{{ formatTime(deployment.finished_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="耗时">
					{{ formatDuration(deployment.duration_ms) }}
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<a-card v-if="deployment?.error_message" title="部署异常">
			<a-alert type="error" :message="deployment.error_message" show-icon />
		</a-card>

		<a-card title="部署日志" :loading="loading" class="log-card">
			<template v-if="deployment" #extra>
				<a-space>
					<a-button size="small" @click="refreshDeployment">刷新</a-button>
					<a-tooltip title="滚动到底部">
						<a-button size="small" :disabled="!logText" @click="scrollToBottom">
							<template #icon><VerticalAlignBottomOutlined /></template>
						</a-button>
					</a-tooltip>
				</a-space>
			</template>
			<div v-if="deployment" ref="logContainerRef" class="log-container">
				<pre class="log-content">{{ logText || '暂无日志' }}</pre>
			</div>
		</a-card>
	</div>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { VerticalAlignBottomOutlined } from '@ant-design/icons-vue';
import { formatTime } from '@/utils/time';
import { formatDuration } from '@/utils/status';
import { onMounted, onUnmounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { type DeploymentDetail, deploymentStatusColors } from '@/types/api';

const route = useRoute();
const router = useRouter();
const deploymentId = route.params.id as string;

const { loading, execute } = useStatusAsync();
const deployment = ref<DeploymentDetail>();
const logText = ref('');
const logOffset = ref(0);
const logContainerRef = ref<HTMLElement>();

let pollTimer: number | null = null;

function getStatusColor(status: string) {
	return deploymentStatusColors[status] || 'default';
}

async function fetchDeployment() {
	try {
		await execute(async () => {
			const data = await deploymentApi.get(deploymentId);
			deployment.value = data;
		});
	} catch (error) {
		message.error('获取部署记录详情失败\n' + error);
		router.push('/deployments');
	}
}

async function fetchLogs() {
	try {
		const data = await deploymentApi.getLogs(deploymentId, logOffset.value);
		if (data.logs) {
			logText.value += data.logs;
			logOffset.value = data.offset;
		}

		// 如果部署完成，停止轮询
		if (data.is_complete) {
			stopLogPolling();
			// 刷新部署状态
			await fetchDeployment();
		}
	} catch (error) {
		console.error('获取日志失败:', error);
	}
}

function startLogPolling() {
	// 立即获取一次日志
	fetchLogs();

	// 已完成的部署不需要轮询
	if (deployment.value && ['success', 'failed'].includes(deployment.value.status)) {
		return;
	}

	// 每2秒轮询一次
	pollTimer = window.setInterval(() => {
		fetchLogs();
	}, 2000);
}

function stopLogPolling() {
	if (pollTimer) {
		clearInterval(pollTimer);
		pollTimer = null;
	}
}

async function handleCancel() {
	try {
		await deploymentApi.cancel(deploymentId);
		message.success('已取消部署');
		stopLogPolling();
		fetchDeployment();
	} catch (error) {
		message.error('取消失败\n' + error);
	}
}

async function refreshDeployment() {
	await fetchDeployment();
	scrollToBottom();
}

function scrollToBottom() {
	if (logContainerRef.value) {
		logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight;
	}
}

onMounted(async () => {
	await fetchDeployment();
	startLogPolling();
});

onUnmounted(() => {
	stopLogPolling();
});
</script>

<style scoped>
.page-wrapper {
	display: flex;
	flex-direction: column;
	gap: 16px;
	height: 100%;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	min-height: 32px;
}

.page-header h2 {
	margin: 0;
}

.log-card {
	flex: 1;
	min-height: 0;
	display: flex;
	flex-direction: column;
}

.log-card :deep(.ant-card-body) {
	flex: 1;
	min-height: 0;
	display: flex;
	flex-direction: column;
	padding: 12px;
}

.log-container {
	flex: 1;
	min-height: 0;
	background: #1e1e1e;
	border-radius: 4px;
	padding: 12px;
	overflow: auto;
}

.log-content {
	color: #d4d4d4;
	font-family: 'Consolas', 'Monaco', monospace;
	font-size: 13px;
	line-height: 1.5;
	margin: 0;
	white-space: pre-wrap;
	word-break: break-all;
}
</style>
