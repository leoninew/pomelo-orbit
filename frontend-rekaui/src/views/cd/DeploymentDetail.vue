<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">部署详情</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="deployment && !isTerminalStatus(deployment.status)"
					class="app-button h-9 px-3"
					@click="isCancelDialogOpen = true"
				>
					<X class="size-4" />
					取消部署
				</button>
				<button class="app-button h-9 px-4" @click="router.push('/cd/deployments')">
					<ArrowLeft class="size-4" />
					返回
				</button>
			</div>
		</div>

		<!-- 加载状态 -->
		<AppSpinner v-if="status === 'loading'" class="py-12" />

		<!-- 内容 -->
		<div v-else-if="deployment" class="flex flex-col gap-4">
			<!-- 基本信息卡片 -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">部署 ID</dt>
						<dd class="min-w-0 break-all text-foreground">{{ deploymentId }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">应用</dt>
						<dd>
							<router-link :to="`/cd/applications/${deployment.application_id}`" class="app-link">
								{{ deployment.application_name || deployment.application_id }}
							</router-link>
						</dd>
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
							{{
								deployment.operation_type === 'deploy'
									? '部署'
									: deployment.operation_type === 'stop'
										? '停止'
										: '重启'
							}}
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">触发方式</dt>
						<dd class="text-foreground">{{ deployment.trigger_type }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">环境</dt>
						<dd class="text-foreground">{{ deployment.environment || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">环境文件</dt>
						<dd class="text-foreground">{{ deployment.env_file || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">耗时</dt>
						<dd class="text-muted-foreground">{{ formatDuration(deployment.started_at, deployment.finished_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(deployment.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">开始时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(deployment.started_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">完成时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(deployment.finished_at) }}</dd>
					</div>
					<div v-if="deployment.error_message" class="flex gap-2 sm:col-span-2">
						<dt class="w-24 shrink-0 text-muted-foreground">错误信息</dt>
						<dd class="min-w-0 text-destructive">
							<span class="block truncate" :title="deployment.error_message">
								{{ deployment.error_message }}
							</span>
						</dd>
					</div>
				</dl>
			</div>

			<!-- 日志卡片 -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="font-semibold text-foreground">部署日志</h2>
					<button class="app-button h-8 px-3" @click="refreshDeployment">
						<RefreshCw class="size-4" />
						刷新
					</button>
				</div>
				<div
					ref="logContainerRef"
					class="h-[600px] overflow-y-auto bg-muted/30 p-4 font-mono text-xs text-foreground"
				>
					<pre v-if="logText" class="whitespace-pre-wrap">{{ logText }}</pre>
					<div v-else class="flex h-full items-center justify-center text-muted-foreground">
						<div class="text-center">
							<AppSpinner v-if="logStatus === 'loading' || logStatus === 'streaming'" />
							<p v-if="logStatus === 'loading'" class="mt-2 text-sm">加载日志中...</p>
							<p v-else-if="logStatus === 'streaming'" class="mt-2 text-sm">日志流传输中...</p>
							<p v-else-if="logStatus === 'empty'" class="text-sm">暂无日志输出</p>
							<div v-else-if="logStatus === 'error'">
								<p class="text-sm text-destructive">日志加载失败</p>
								<button class="app-link mt-2 text-sm" @click="refreshDeployment">重试</button>
							</div>
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
				<button type="button" class="app-button" @click="isCancelDialogOpen = false">取消</button>
				<button type="button" class="app-button-destructive" @click="handleCancel">确认取消</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { ArrowLeft, RefreshCw, X } from 'lucide-vue-next';
	import { computed, onMounted, onUnmounted, ref } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { deploymentApi } from '@/api/cd/deployments';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useAuthStore } from '@/stores/auth';
	import type { DeploymentDetail } from '@/types/cd/deployment';
	import { isTerminalStatus } from '@/utils/status';
	import { delayAsync, formatDuration, formatTime } from '@/utils/time';
	import config from '@/config';

	const route = useRoute();
	const router = useRouter();
	const deploymentId = route.params.id as string;
	const toast = useToast();
	const { status, execute } = useStatusAsync();
	const authStore = useAuthStore();

	const deployment = ref<DeploymentDetail>();
	const logText = ref('');
	const logOffset = ref(0);
	const logContainerRef = ref<HTMLElement>();
	const isCancelDialogOpen = ref(false);
	const logStatus = ref<'loading' | 'streaming' | 'done' | 'empty' | 'error'>('loading');
	let logAbort: AbortController | null = null;

	const statusBadgeClass = computed(() => {
		const s = deployment.value?.status;
		if (!s) {
			return 'bg-muted/50 text-muted-foreground';
		}
		const map: Record<string, string> = {
			waiting_to_run: 'bg-muted/50 text-muted-foreground',
			running: 'bg-blue-50 text-blue-700 border-blue-200',
			ran_to_completion: 'bg-green-50 text-green-700 border-green-200',
			faulted: 'bg-red-50 text-red-700 border-red-200',
			canceled: 'bg-gray-50 text-gray-700 border-gray-200',
		};
		return map[s] || 'bg-muted/50 text-muted-foreground';
	});

	const statusLabel = computed(() => {
		const s = deployment.value?.status;
		if (!s) {
			return '';
		}
		const map: Record<string, string> = {
			waiting_to_run: '等待运行',
			running: '运行中',
			ran_to_completion: '成功',
			faulted: '失败',
			canceled: '已取消',
		};
		return map[s] || s;
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
				logStatus.value = logText.value ? 'done' : 'empty';
				deployment.value = await deploymentApi.get(deploymentId);
			} else {
				logStatus.value = 'streaming';
			}
		} catch (error) {
			console.error('获取日志失败:', error);
			logStatus.value = 'error';
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
				try {
					await fetchLogs();
				} catch {
					logStatus.value = 'error';
					break;
				}
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
		let reader: ReadableStreamDefaultReader<string> | null = null;

		try {
			const response = await deploymentApi.streamLogs(deploymentId, authStore.token, signal);
			if (!response.body) {
				logStatus.value = logText.value ? 'done' : 'empty';
				return;
			}

			reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
			let buffer = '';
			logStatus.value = 'streaming';

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
						logStatus.value = logText.value ? 'done' : 'empty';
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
			logStatus.value = logText.value ? 'done' : 'empty';
		} catch (error) {
			if (!signal.aborted) {
				console.error('日志流读取失败:', error);
				logStatus.value = 'error';
			}
		} finally {
			if (reader) {
				try {
					await reader.cancel();
				} catch {
					// ignore cleanup errors
				}
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
