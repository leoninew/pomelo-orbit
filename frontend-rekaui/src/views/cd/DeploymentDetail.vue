<script setup lang="ts">
import { Loader2, RefreshCw } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { deploymentApi } from '@/api/cd/deployments';
import AppDialog from '@/components/AppDialog.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { useAuthStore } from '@/stores/auth';
import type { DeploymentDetail } from '@/types/cd/deployment';
import { isTerminalStatus } from '@/utils/status';
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
const isCancelDialogOpen = ref(false);
let logAbort: AbortController | null = null;

const statusBadgeClass = computed(() => {
	const status = deployment.value?.status;
	if (!status) {return 'bg-muted/50 text-muted-foreground';}
	const map: Record<string, string> = {
		waiting_to_run: 'bg-muted/50 text-muted-foreground',
		running: 'bg-blue-50 text-blue-700 border-blue-200',
		ran_to_completion: 'bg-green-50 text-green-700 border-green-200',
		faulted: 'bg-red-50 text-red-700 border-red-200',
		canceled: 'bg-gray-50 text-gray-700 border-gray-200',
	};
	return map[status] || 'bg-muted/50 text-muted-foreground';
});

const statusLabel = computed(() => {
	const status = deployment.value?.status;
	if (!status) {return '';}
	const map: Record<string, string> = {
		waiting_to_run: '等待运行',
		running: '运行中',
		ran_to_completion: '成功',
		faulted: '失败',
		canceled: '已取消',
	};
	return map[status] || status;
});

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
		isCancelDialogOpen.value = false;
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
			await fetchLogs();
		} else {
			startLogPolling();
		}
	}
});
onUnmounted(stopLog);
</script>

<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">部署详情</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="deployment && !isTerminalStatus(deployment.status)"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isCancelDialogOpen = true"
				>
					取消部署
				</button>
				<button
					class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-input bg-background text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
					@click="refreshDeployment"
				>
					<RefreshCw class="h-4 w-4" />
				</button>
				<button
					class="h-9 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="router.push('/cd/deployments')"
				>
					返回
				</button>
			</div>
		</div>

		<!-- 加载状态 -->
		<div v-if="loading" class="flex items-center justify-center py-12">
			<div class="h-8 w-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary"></div>
		</div>

		<!-- 内容 -->
		<div v-else-if="deployment" class="flex flex-col gap-4">
				<!-- 基本信息卡片 -->
				<div class="rounded-lg border border-border bg-card shadow-sm">
					<div class="border-b border-border px-5 py-4">
						<h2 class="font-semibold text-foreground">基本信息</h2>
					</div>
					<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">部署 ID</dt>
							<dd class="min-w-0 text-foreground">{{ deploymentId }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">应用</dt>
							<dd class="text-foreground">{{ deployment.application_name }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">状态</dt>
							<dd>
								<span
									class="inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium"
									:class="statusBadgeClass"
								>
									{{ statusLabel }}
								</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">操作类型</dt>
							<dd class="text-foreground">
								{{ deployment.operation_type === 'deploy' ? '部署' : deployment.operation_type === 'stop' ? '停止' : '重启' }}
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
							<dd class="text-muted-foreground">{{ formatTime(deployment.created_at) }}</dd>
						</div>
						<div v-if="deployment.started_at" class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">开始时间</dt>
							<dd class="text-muted-foreground">{{ formatTime(deployment.started_at) }}</dd>
						</div>
						<div v-if="deployment.finished_at" class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">完成时间</dt>
							<dd class="text-muted-foreground">{{ formatTime(deployment.finished_at) }}</dd>
						</div>
					</dl>
				</div>

				<!-- 日志卡片 -->
				<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
					<div class="border-b border-border px-5 py-4">
						<h2 class="font-semibold text-foreground">部署日志</h2>
					</div>
					<div
						ref="logContainerRef"
						class="h-[600px] overflow-y-auto bg-muted/30 p-4 font-mono text-xs text-foreground"
					>
						<pre v-if="logText" class="whitespace-pre-wrap">{{ logText }}</pre>
						<div v-else class="flex h-full items-center justify-center text-muted-foreground">
							<div class="text-center">
								<Loader2 class="mx-auto h-8 w-8 animate-spin" />
								<p class="mt-2">加载日志中...</p>
							</div>
						</div>
					</div>
				</div>
		</div>

		<AppDialog
			v-model:open="isCancelDialogOpen"
			title="确认取消"
			description="确定要取消此部署吗？"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button
					type="button"
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isCancelDialogOpen = false"
				>
					取消
				</button>
				<button
					type="button"
					class="rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90"
					@click="handleCancel"
				>
					确认取消
				</button>
			</template>
		</AppDialog>
	</div>
</template>
