<template>
	<div class="flex flex-col gap-4 h-full">
		<!-- Header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				部署记录
				<span v-if="deployment" class="text-base-content/60 text-base font-normal">
					{{ deployment.application_name }}
				</span>
				<span v-if="deployment" class="badge badge-sm" :class="statusBadgeClass(deployment.status)">
					{{ statusLabel(deployment.status) }}
				</span>
			</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/cd/deployments')">
					<ArrowLeft class="size-4" />
					返回列表
				</button>
				<button
					v-if="deployment"
					class="btn btn-sm btn-ghost gap-1"
					@click="$router.push(`/cd/applications/${deployment.application_id}`)"
				>
					返回应用
				</button>
				<button
					v-if="deployment?.status === 'running' || deployment?.status === 'waiting_to_run'"
					class="btn btn-sm btn-error btn-ghost gap-1"
					@click="handleCancel"
				>
					取消部署
				</button>
			</div>
		</div>

		<!-- Basic info -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-4">基本信息</h2>
				<div v-if="loading" class="flex justify-center py-6">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<dl v-else-if="deployment" class="grid grid-cols-1 sm:grid-cols-3 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-20 shrink-0">应用</dt>
						<dd>
							<router-link
								:to="`/cd/applications/${deployment.application_id}`"
								class="link link-primary"
							>
								{{ deployment.application_name || deployment.application_id }}
							</router-link>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-20 shrink-0">触发方式</dt>
						<dd class="text-base-content/70">{{ deployment.trigger_type }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-20 shrink-0">环境文件</dt>
						<dd class="text-base-content/70">{{ deployment.env_file || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-20 shrink-0">耗时</dt>
						<dd class="text-base-content/70">{{ formatDuration(deployment.duration_ms) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-20 shrink-0">开始时间</dt>
						<dd class="text-base-content/60">{{ formatTime(deployment.started_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-20 shrink-0">结束时间</dt>
						<dd class="text-base-content/60">{{ formatTime(deployment.finished_at) }}</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- Error message -->
		<div v-if="deployment?.error_message" role="alert" class="alert alert-error">
			<span class="text-sm">{{ deployment.error_message }}</span>
		</div>

		<!-- Log card -->
		<div class="card bg-base-100 shadow-sm flex-1 flex flex-col min-h-0">
			<div class="flex items-center justify-between px-5 py-3 border-b border-base-200 shrink-0">
				<h2 class="font-semibold">部署日志</h2>
				<div class="flex items-center gap-2">
					<button
						v-if="deployment && !isTerminalStatus(deployment.status)"
						class="btn btn-xs btn-ghost gap-1 text-primary"
					>
						<Loader2 class="size-3.5 animate-spin" />
						自动刷新中
					</button>
					<button class="btn btn-xs btn-ghost gap-1" @click="refreshDeployment">
						<RefreshCw class="size-3.5" />
						刷新
					</button>
					<button
						title="滚动到底部"
						class="btn btn-xs btn-ghost"
						:disabled="!logText"
						@click="scrollToBottom"
					>
						<ArrowDown class="size-3.5" />
					</button>
				</div>
			</div>
			<div
				ref="logContainerRef"
				class="flex-1 overflow-auto p-4 bg-base-200 rounded-b-box min-h-64"
			>
				<pre
					class="text-base-content font-mono text-xs leading-relaxed whitespace-pre-wrap break-all"
					>{{ logText || '暂无日志' }}</pre
				>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { ArrowDown, ArrowLeft, Loader2, RefreshCw } from 'lucide-vue-next';
import { onMounted, onUnmounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { deploymentApi } from '@/api/cd/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { useAuthStore } from '@/stores/auth';
import type { DeploymentDetail } from '@/types/cd/deployment';
import { formatDuration, isTerminalStatus, statusBadgeClass, statusLabel } from '@/utils/status';
import { delayAsync, formatTime } from '@/utils/time';
import config from '@/config';

const route = useRoute();
const router = useRouter();
const deploymentId = route.params.id as string;
const toast = useToast();
const { loading, execute } = useStatusAsync();
const authStore = useAuthStore();

const deployment = ref<DeploymentDetail>();
const logText = ref('');
const logOffset = ref(0);
const logContainerRef = ref<HTMLElement>();
let logAbort: AbortController | null = null;
async function fetchDeployment() {
	try {
		await execute(async () => {
			const data = await deploymentApi.get(deploymentId);
			deployment.value = data;
		});
	} catch {
		toast.error('获取部署详情失败');
		router.push('/cd/deployments');
	}
}

async function fetchLogs() {
	try {
		const data = await deploymentApi.getLogs(deploymentId, logOffset.value);
		if (data.logs) {
			logText.value += data.logs;
			logOffset.value = data.offset;
		}
		if (data.is_complete) {
			logAbort?.abort();
			deployment.value = await deploymentApi.get(deploymentId);
		}
	} catch (error) {
		console.error('获取日志失败:', error);
	}
}

function startLogPolling() {
	logAbort = new AbortController();
	const signal = logAbort.signal;
	(async () => {
		await fetchLogs();
		while (!signal.aborted) {
			if (deployment.value && isTerminalStatus(deployment.value.status)) {
				break;
			}
			await delayAsync(2000);
			if (signal.aborted) {
				break;
			}
			await fetchLogs();
		}
	})();
}

function stopLog() {
	logAbort?.abort();
	logAbort = null;
}

async function startLogStream(refreshOnComplete = true) {
	logAbort = new AbortController();
	const signal = logAbort.signal;

	try {
		const response = await deploymentApi.streamLogs(deploymentId, authStore.token, signal);
		if (!response.body) {
			return;
		}

		const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
		let buffer = '';

		while (!signal.aborted) {
			const { done, value } = await reader.read();
			if (done) {
				break;
			}

			buffer += value;
			const parts = buffer.split('\n\n');
			buffer = parts.pop() ?? '';

			for (const part of parts) {
				if (part.startsWith('event: complete')) {
					if (refreshOnComplete) {
						deployment.value = await deploymentApi.get(deploymentId);
					}
					return;
				}
				const dataLine = part.split('\n').find((l) => l.startsWith('data: '));
				if (dataLine) {
					const payload = JSON.parse(dataLine.slice(6));
					if (payload.logs) {
						logText.value += payload.logs;
						scrollToBottom();
					}
				}
			}
		}
	} catch (error) {
		if (!signal.aborted) {
			console.error('日志流读取失败:', error);
		}
	}
}

async function handleCancel() {
	try {
		await deploymentApi.cancel(deploymentId);
		toast.success('已取消部署');
		stopLog();
		deployment.value = await deploymentApi.get(deploymentId);
	} catch {
		toast.error('取消失败');
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
	if (!deployment.value) {
		return;
	}
	if (config.features.sseDeploymentLog) {
		if (isTerminalStatus(deployment.value.status)) {
			startLogStream(false);
		} else {
			startLogStream();
		}
	} else {
		if (isTerminalStatus(deployment.value.status)) {
			// 已完成的部署直接拉取完整日志
			await fetchLogs();
		} else {
			startLogPolling();
		}
	}
});
onUnmounted(stopLog);
</script>
