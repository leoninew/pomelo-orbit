<script setup lang="ts">
import { Loader2, X } from 'lucide-vue-next';
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineRunApi, pipelineTemplateApi } from '@/api/ci';
import AppDialog from '@/components/AppDialog.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineSnapshot } from '@/types/ci/snapshot';
import type { PipelineRun } from '@/types/ci/run';
import type { StageRun } from '@/types/ci/stage_run';
import { isTerminalStatus } from '@/utils/status';
import { delayAsync, formatTime } from '@/utils/time';
import StageDAGView from './components/StageDAGView.vue';

const route = useRoute();
const router = useRouter();
const runId = computed(() => route.params.id as string);
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: retrying, execute: executeRetry } = useStatusAsync();
const { loading: canceling, execute: executeCancel } = useStatusAsync();

const run = ref<PipelineRun>();
const snapshot = ref<PipelineSnapshot>();
const currentStageRun = ref<StageRun>();
const showLogsDrawer = ref(false);
const isCancelDialogOpen = ref(false);
const stagesView = ref<'list' | 'dag'>('list');

const stageRuns = computed(() => run.value?.stage_runs ?? []);

// 日志 drawer 状态
const logsText = ref('');
const logsLoading = ref(false);
const logContainer = ref<HTMLDivElement>();
let logPollAbort: AbortController | null = null;

let pollAbort: AbortController | null = null;
const isPolling = ref(false);

const statusBadgeClass = computed(() => {
	const status = run.value?.status;
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

const statusText = computed(() => {
	const status = run.value?.status;
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

function getStageStatusBadge(status: string) {
	const map: Record<string, string> = {
		waiting_to_run: 'bg-muted/50 text-muted-foreground',
		running: 'bg-blue-50 text-blue-700 border-blue-200',
		ran_to_completion: 'bg-green-50 text-green-700 border-green-200',
		faulted: 'bg-red-50 text-red-700 border-red-200',
		canceled: 'bg-gray-50 text-gray-700 border-gray-200',
		skipped: 'bg-gray-50 text-gray-600 border-gray-200',
	};
	return map[status] || 'bg-muted/50 text-muted-foreground';
}

function getStageStatusText(status: string) {
	const map: Record<string, string> = {
		waiting_to_run: '等待',
		running: '运行中',
		ran_to_completion: '成功',
		faulted: '失败',
		canceled: '已取消',
		skipped: '跳过',
	};
	return map[status] || status;
}

function openLogDrawer(sr: StageRun) {
	logPollAbort?.abort();
	currentStageRun.value = sr;
	logsText.value = '';
	showLogsDrawer.value = true;
	startLogPolling(sr.id);
}

function closeLogDrawer() {
	showLogsDrawer.value = false;
	logPollAbort?.abort();
	logPollAbort = null;
}

async function startLogPolling(stageRunId: string) {
	logPollAbort = new AbortController();
	const signal = logPollAbort.signal;
	logsLoading.value = true;
	let offset = 0;

	while (!signal.aborted) {
		try {
			const resp = await pipelineRunApi.getStageLog(runId.value, stageRunId, offset);
			if (currentStageRun.value?.id !== stageRunId) {
				break;
			}
			if (resp.logs) {
				logsText.value += resp.logs;
				offset = resp.offset;
				await nextTick();
				scrollToBottom();
			}
			logsLoading.value = false;
			if (resp.is_complete) {
				break;
			}
		} catch {
			logsLoading.value = false;
			break;
		}
		await delayAsync(1500);
	}
}

function scrollToBottom() {
	if (logContainer.value) {
		logContainer.value.scrollTop = logContainer.value.scrollHeight;
	}
}

async function fetchRun() {
	try {
		await execute(async () => {
			const data = await pipelineRunApi.get(runId.value);
			run.value = data;
		});
	} catch {
		toast.error('获取 Run 信息失败');
		router.push('/ci/run');
	}
}

async function fetchSnapshot(snapshotId: string) {
	try {
		const data = await pipelineTemplateApi.getSnapshot(snapshotId);
		snapshot.value = data;
	} catch {
		// snapshot 加载失败不影响主流程
	}
}

async function handleRetry() {
	try {
		await executeRetry(async () => {
			const newRun = await pipelineRunApi.retry(runId.value);
			toast.success('重试成功');
			router.push(`/ci/run/${newRun.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '重试失败');
	}
}

async function handleCancel() {
	try {
		await executeCancel(async () => {
			await pipelineRunApi.cancel(runId.value);
			toast.success('已取消');
			isCancelDialogOpen.value = false;
			await fetchRun();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '取消失败');
	}
}

async function startPolling() {
	pollAbort = new AbortController();
	const signal = pollAbort.signal;
	isPolling.value = true;
	while (!signal.aborted) {
		try {
			run.value = await pipelineRunApi.get(runId.value);
			if (isTerminalStatus(run.value.status)) {
				isPolling.value = false;
				break;
			}
		} catch {
			// 网络抖动时静默重试，不中断轮询
		}
		await delayAsync(2000);
	}
}

function stopPolling() {
	pollAbort?.abort();
	pollAbort = null;
	isPolling.value = false;
}

function togglePolling() {
	if (isPolling.value) {
		stopPolling();
	} else {
		startPolling();
	}
}

async function init() {
	stopPolling();
	run.value = undefined;
	snapshot.value = undefined;
	await fetchRun();
	if (run.value?.snapshot_id) {
		await fetchSnapshot(run.value.snapshot_id);
	}
	if (run.value && !isTerminalStatus(run.value.status)) {
		startPolling();
	}
}

watch(runId, init);

onMounted(init);

onUnmounted(() => {
	stopPolling();
	logPollAbort?.abort();
});
</script>

<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">流水线详情</h1>
			<div class="flex flex-wrap items-center gap-2">
					<button
						v-if="run?.status === 'faulted'"
						:disabled="retrying"
						class="h-9 rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
						@click="handleRetry"
					>
						{{ retrying ? '重试中...' : '重试' }}
					</button>
					<button
						v-if="run?.status === 'waiting_to_run' || run?.status === 'running'"
						:disabled="canceling"
						class="h-9 rounded-md border border-destructive/50 bg-background px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-50"
						@click="isCancelDialogOpen = true"
					>
						取消
					</button>
					<button
						v-if="run && !isTerminalStatus(run.status)"
						class="inline-flex h-9 items-center gap-2 rounded-md border border-input bg-background px-3 text-sm font-medium transition-colors hover:bg-muted/50"
						:class="isPolling ? 'text-primary' : 'text-muted-foreground'"
						@click="togglePolling"
					>
						<Loader2 class="h-4 w-4" :class="isPolling ? 'animate-spin' : ''" />
						{{ isPolling ? '自动刷新中' : '已暂停刷新' }}
					</button>
					<button
						class="h-9 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
						@click="router.push('/ci/run')"
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
		<div v-else-if="run" class="flex flex-col gap-4">
				<!-- 基本信息卡片 -->
				<div class="rounded-lg border border-border bg-card shadow-sm">
					<div class="border-b border-border px-5 py-4">
						<h2 class="font-semibold text-foreground">基本信息</h2>
					</div>
					<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">运行 ID</dt>
							<dd class="min-w-0 text-foreground">{{ runId }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">状态</dt>
							<dd>
								<span
									class="inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium"
									:class="statusBadgeClass"
								>
									{{ statusText }}
								</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">代码仓库</dt>
							<dd>
								<router-link
									:to="`/ci/repository/${run.repository_id}`"
									class="text-primary hover:underline"
								>
									{{ run.repository_name }}
								</router-link>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">触发方式</dt>
							<dd>
								<span class="inline-block rounded-full bg-muted/50 px-2 py-0.5 text-xs font-medium text-foreground">
									{{ run.trigger }}
								</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">触发分支</dt>
							<dd class="text-foreground">{{ run.trigger_ref }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">模板</dt>
							<dd>
								<router-link
									:to="`/ci/template/${run.template_id}`"
									class="text-primary hover:underline"
								>
									{{ run.template_name }} v{{ run.template_version }}
								</router-link>
							</dd>
						</div>
						<div v-if="run.created_at" class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
							<dd class="text-muted-foreground">{{ formatTime(run.created_at) }}</dd>
						</div>
						<div v-if="run.started_at" class="flex gap-2">
							<dt class="w-24 shrink-0 text-muted-foreground">开始时间</dt>
							<dd class="text-muted-foreground">{{ formatTime(run.started_at) }}</dd>
						</div>
					</dl>
				</div>

				<!-- Stage 列表 -->
				<div class="rounded-lg border border-border bg-card shadow-sm">
					<div class="flex items-center justify-between border-b border-border px-5 py-4">
						<h2 class="font-semibold text-foreground">Stage 执行</h2>
						<div v-if="snapshot?.stages_snapshot && snapshot.stages_snapshot.length > 0" class="flex gap-1 rounded-md border border-border bg-background p-1">
							<button
								class="rounded px-3 py-1 text-xs font-medium transition-colors"
								:class="stagesView === 'list' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'"
								@click="stagesView = 'list'"
							>
								列表
							</button>
							<button
								class="rounded px-3 py-1 text-xs font-medium transition-colors"
								:class="stagesView === 'dag' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'"
								@click="stagesView = 'dag'"
							>
								DAG
							</button>
						</div>
					</div>
					
					<!-- 列表视图 -->
					<div v-if="stagesView === 'list'" class="divide-y divide-border">
						<div
							v-for="sr in stageRuns"
							:key="sr.id"
							class="flex items-center justify-between px-5 py-4 transition-colors hover:bg-muted/30"
						>
							<div class="flex items-center gap-4">
								<span
									class="inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium"
									:class="getStageStatusBadge(sr.status)"
								>
									{{ getStageStatusText(sr.status) }}
								</span>
								<div>
									<p class="text-sm text-foreground">{{ sr.stage_name }}</p>
									<p v-if="sr.started_at" class="text-xs text-muted-foreground">
										{{ formatTime(sr.started_at) }}
									</p>
								</div>
							</div>
							<button
								v-if="sr.status !== 'waiting_to_run'"
								class="rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-muted/50"
								@click="openLogDrawer(sr)"
							>
								查看日志
							</button>
						</div>
					</div>
					
					<!-- DAG 视图 -->
					<div v-else-if="snapshot?.stages_snapshot && snapshot.stages_snapshot.length > 0" class="p-6">
						<div class="h-[500px]">
							<StageDAGView
								:stages="snapshot.stages_snapshot"
								:stage-runs="stageRuns"
								:animated="true"
								@view-stage="openLogDrawer"
							/>
						</div>
					</div>
				</div>
		</div>

		<!-- 日志抽屉 -->
		<div
			v-if="showLogsDrawer"
			class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
			@click.self="closeLogDrawer"
		>
			<div class="relative h-[80vh] w-[90vw] max-w-5xl rounded-lg border border-border bg-card">
				<div class="flex items-center justify-between border-b border-border px-6 py-4">
					<h3 class="text-lg font-semibold text-foreground">
						{{ currentStageRun?.stage_name }} - 日志
					</h3>
					<button
						class="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
						@click="closeLogDrawer"
					>
						<X class="h-5 w-5" />
					</button>
				</div>
				<div
					ref="logContainer"
					class="h-[calc(80vh-4rem)] overflow-y-auto bg-muted/30 p-4 font-mono text-xs text-foreground"
				>
					<pre v-if="logsText" class="whitespace-pre-wrap">{{ logsText }}</pre>
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
			description="确定要取消此流水线运行吗？"
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
					:disabled="canceling"
					class="rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90 disabled:cursor-not-allowed disabled:opacity-50"
					@click="handleCancel"
				>
					{{ canceling ? '取消中...' : '确认取消' }}
				</button>
			</template>
		</AppDialog>
	</div>
</template>
