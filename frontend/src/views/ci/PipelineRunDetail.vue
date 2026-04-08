<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				流水线详情
				<span v-if="run" class="badge badge-sm" :class="statusBadgeClass(run.status)">
					{{ statusLabel(run.status) }}
				</span>
			</h1>
			<div class="flex items-center gap-2">
				<button
					v-if="run && !isTerminalStatus(run.status)"
					class="btn btn-sm btn-ghost gap-1"
					:class="{ 'text-primary': isPolling }"
					@click="togglePolling"
				>
					<Loader2 class="size-4" :class="{ 'animate-spin': isPolling }" />
					{{ isPolling ? '自动刷新中' : '自动刷新' }}
				</button>
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/runs')">
					<ArrowLeft class="size-4" />
					返回
				</button>
			</div>
		</div>

		<!-- Loading -->
		<div v-if="loading" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>

		<template v-else-if="run">
			<!-- Basic info -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-3">
						<h2 class="font-semibold">基本信息</h2>
						<div class="flex items-center gap-2">
							<button
								v-if="run.status === 'faulted'"
								class="btn btn-sm btn-primary"
								:disabled="retrying"
								@click="handleRetry"
							>
								<span v-if="retrying" class="loading loading-spinner loading-xs" />
								重试
							</button>
							<button
								v-if="run.status === 'waiting_to_run' || run.status === 'running'"
								class="btn btn-sm btn-error btn-ghost"
								:disabled="canceling"
								@click="cancelModalRef?.showModal()"
							>
								取消
							</button>
						</div>
					</div>
					<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">Run ID</dt>
							<dd class="font-mono text-xs">{{ run.id }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">Project</dt>
							<dd>
								<router-link
									:to="`/ci/projects/${run.project_id}`"
									class="link link-primary text-xs"
								>
									{{ run.project_id }}
								</router-link>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">触发方式</dt>
							<dd>
								<span class="badge badge-sm badge-ghost">{{ run.trigger }}</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">Ref</dt>
							<dd class="text-base-content/60">{{ run.trigger_ref }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">快照</dt>
							<dd>
								<router-link
									v-if="snapshot"
									:to="`/ci/snapshots/${run.pipeline_snapshot_id}`"
									class="link link-primary text-xs"
								>
									{{ snapshot.template_id }} v{{ snapshot.version }}
								</router-link>
								<span v-else class="text-base-content/60">—</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">重试自</dt>
							<dd>
								<router-link
									v-if="run.retry_of"
									:to="`/ci/runs/${run.retry_of}`"
									class="link link-primary text-xs"
								>
									{{ run.retry_of }}
								</router-link>
								<span v-else class="text-base-content/60">—</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">开始时间</dt>
							<dd>{{ run.started_at ? formatTime(run.started_at) : '—' }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">结束时间</dt>
							<dd>{{ run.finished_at ? formatTime(run.finished_at) : '—' }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
							<dd>{{ formatTime(run.created_at) }}</dd>
						</div>
					</dl>
				</div>
			</div>

			<!-- Stages -->
			<div
				v-if="snapshot && snapshot.stages_snapshot.length > 0"
				class="card bg-base-100 shadow-sm"
			>
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-3">
						<h2 class="font-semibold">Stages</h2>
						<div class="join">
							<button
								class="btn btn-xs join-item"
								:class="stagesView === 'list' ? 'btn-active' : 'btn-ghost'"
								@click="stagesView = 'list'"
							>
								列表
							</button>
							<button
								class="btn btn-xs join-item"
								:class="stagesView === 'dag' ? 'btn-active' : 'btn-ghost'"
								@click="stagesView = 'dag'"
							>
								DAG
							</button>
						</div>
					</div>
					<StageListView
						v-if="stagesView === 'list'"
						:stages="snapshot.stages_snapshot"
						:orchestration="snapshotStagesAsOrch"
						:stage-statuses="stageStatuses"
						:readonly="true"
						@view-log="onViewLog"
					/>
					<template v-else>
						<StageDAGView
							:stages="snapshot.stages_snapshot"
							:stage-statuses="stageStatuses"
							:show-minimap="true"
							:readonly="true"
							@view-stage="onViewStage"
						/>
						<p class="text-xs text-base-content/50 mt-2">点击节点查看日志</p>
					</template>
				</div>
			</div>

			<!-- Artifacts -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<h2 class="font-semibold mb-3">制品</h2>
					<div v-if="artifactsLoading" class="flex justify-center py-6">
						<span class="loading loading-spinner loading-md text-primary" />
					</div>
					<div
						v-else-if="artifacts.length === 0"
						class="text-sm text-base-content/60 py-4 text-center"
					>
						暂无制品
					</div>
					<table v-else class="table">
						<thead>
							<tr class="text-base-content/60">
								<th>Stage</th>
								<th>类型</th>
								<th>名称</th>
								<th>路径</th>
								<th>创建时间</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="a in artifacts" :key="a.id" class="hover">
								<td>{{ a.stage_name }}</td>
								<td>
									<span class="badge badge-sm badge-ghost">{{ a.type }}</span>
								</td>
								<td>{{ a.name }}</td>
								<td class="cell-muted max-w-xs truncate">{{ a.path }}</td>
								<td class="cell-muted">{{ formatTime(a.created_at) }}</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</template>

		<!-- Job logs drawer -->
		<Teleport to="body">
			<Transition
				enter-active-class="transition-transform duration-300 ease-out"
				enter-from-class="translate-x-full"
				enter-to-class="translate-x-0"
				leave-active-class="transition-transform duration-300 ease-in"
				leave-from-class="translate-x-0"
				leave-to-class="translate-x-full"
			>
				<div
					v-if="showLogsDrawer"
					class="fixed inset-y-0 right-0 z-50 w-[800px] max-w-full bg-base-100 shadow-2xl flex flex-col border-l border-base-200"
				>
					<div class="flex items-center justify-between px-5 py-4 border-b border-base-200">
						<h3 class="font-semibold flex items-center gap-2">
							Stage: {{ currentStageRun?.name }}
							<span
								v-if="currentStageRun"
								class="badge badge-sm"
								:class="statusBadgeClass(currentStageRun.status)"
							>
								{{ statusLabel(currentStageRun.status) }}
							</span>
						</h3>
						<button class="btn btn-sm btn-ghost btn-circle" @click="closeLogDrawer">
							<X class="size-4" />
						</button>
					</div>
					<div class="flex-1 overflow-auto p-5 flex flex-col gap-4">
						<div v-if="currentStageRun?.error_message" class="alert alert-error py-2 text-xs">
							{{ currentStageRun.error_message }}
						</div>
						<div v-if="logsLoading" class="flex justify-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</div>
						<div v-else class="flex-1 bg-base-200 rounded-box p-4 overflow-auto min-h-64">
							<pre
								v-if="logsText"
								class="text-base-content font-mono text-xs leading-relaxed whitespace-pre-wrap break-all"
								>{{ logsText }}</pre
							>
							<div v-else class="flex flex-col items-center gap-2 py-8 text-base-content/60">
								<FileX class="size-8" />
								<span class="text-sm">暂无日志</span>
							</div>
						</div>
					</div>
				</div>
			</Transition>
			<Transition
				enter-active-class="transition-opacity duration-300"
				enter-from-class="opacity-0"
				enter-to-class="opacity-100"
				leave-active-class="transition-opacity duration-300"
				leave-from-class="opacity-100"
				leave-to-class="opacity-0"
			>
				<div v-if="showLogsDrawer" class="fixed inset-0 z-40 bg-black/30" @click="closeLogDrawer" />
			</Transition>
		</Teleport>

		<!-- Cancel confirm modal -->
		<dialog ref="cancelModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">取消 Run</h3>
				<p class="py-4 text-sm">确定取消此 Run？</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="canceling" @click="handleCancel">
						<span v-if="canceling" class="loading loading-spinner loading-xs" />
						确定
					</button>
					<button class="btn btn-ghost" @click="cancelModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft, FileX, Loader2, X } from 'lucide-vue-next';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineRunApi, pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Artifact, StageRun, PipelineRun, PipelineSnapshot } from '@/types/api';
import type { TaskStatus } from '@/types/common';
import type { SnapshotStage } from '@/types/ci/snapshot';
import { statusBadgeClass, statusLabel, isTerminalStatus } from '@/utils/status';
import { delayAsync, formatTime } from '@/utils/time';
import StageDAGView from './components/StageDAGView.vue';
import StageListView from './components/StageListView.vue';

const route = useRoute();
const router = useRouter();
const runId = route.params.id as string;
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: artifactsLoading, execute: executeArtifacts } = useStatusAsync();
const { execute: executeSnapshot } = useStatusAsync();
const { loading: retrying, execute: executeRetry } = useStatusAsync();
const { loading: canceling, execute: executeCancel } = useStatusAsync();

const run = ref<PipelineRun>();
const snapshot = ref<PipelineSnapshot>();
const artifacts = ref<Artifact[]>([]);
const currentStageRun = ref<StageRun>();
const showLogsDrawer = ref(false);
const cancelModalRef = ref<HTMLDialogElement>();
const stagesView = ref<'list' | 'dag'>('list');

// 日志 drawer 状态
const logsText = ref('');
const logsLoading = ref(false);
let logPollAbort: AbortController | null = null;

let pollAbort: AbortController | null = null;
const isPolling = ref(false);

const stageRuns = computed<StageRun[]>(() => run.value?.stage_runs ?? []);

const snapshotStagesAsOrch = computed(() =>
	(snapshot.value?.stages_snapshot ?? []).map((s, i) => ({
		stage_id: s.id,
		stage_key: s.name,
		depends_on: s.depends_on,
		sort_order: i,
	}))
);

const stageStatuses = computed<Map<string, TaskStatus>>(() => {
	const map = new Map<string, TaskStatus>();
	if (!snapshot.value) return map;

	for (const stage of snapshot.value.stages_snapshot) {
		const sr = stageRuns.value.find((r) => r.name === stage.name);
		if (!sr) {
			map.set(stage.name, 'waiting_to_run');
			continue;
		}
		map.set(stage.name, sr.status);
	}
	return map;
});

async function onViewStage(stage: SnapshotStage) {
	const sr = stageRuns.value.find((r) => r.name === stage.name);
	if (!sr) {
		toast.error('该 Stage 尚未执行');
		return;
	}
	openLogDrawer(sr);
}

async function onViewLog(stageKey: string) {
	const sr = stageRuns.value.find((r) => r.name === stageKey);
	if (!sr) {
		toast.error('该 Stage 尚未执行');
		return;
	}
	openLogDrawer(sr);
}

function openLogDrawer(sr: StageRun) {
	// 停止上一个日志轮询
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
			const resp = await pipelineRunApi.getStageLog(runId, stageRunId, offset);
			// 如果用户已切换到其他 stage，丢弃过期响应
			if (currentStageRun.value?.id !== stageRunId) break;
			if (resp.logs) {
				logsText.value += resp.logs;
				offset = resp.offset;
			}
			logsLoading.value = false;
			if (resp.is_complete) break;
		} catch {
			logsLoading.value = false;
			break;
		}
		await delayAsync(1500);
	}
}

async function fetchRun() {
	try {
		await execute(async () => {
			const data = await pipelineRunApi.get(runId);
			run.value = data;
			if (data.pipeline_snapshot_id) {
				await fetchSnapshot(data.pipeline_snapshot_id);
			}
		});
	} catch {
		toast.error('获取 Run 信息失败');
		router.push('/ci/runs');
	}
}

async function fetchSnapshot(snapshotId: string) {
	try {
		await executeSnapshot(async () => {
			snapshot.value = await pipelineTemplateApi.getSnapshot(snapshotId);
		});
	} catch {
		// snapshot 加载失败不影响主流程
	}
}

async function fetchArtifacts() {
	try {
		await executeArtifacts(async () => {
			artifacts.value = await pipelineRunApi.listArtifacts(runId);
		});
	} catch {
		/* silent */
	}
}

async function handleRetry() {
	try {
		await executeRetry(async () => {
			const newRun = await pipelineRunApi.retry(runId);
			toast.success('重试成功');
			router.push(`/ci/runs/${newRun.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '重试失败');
	}
}

async function handleCancel() {
	try {
		await executeCancel(async () => {
			await pipelineRunApi.cancel(runId);
			toast.success('已取消');
			cancelModalRef.value?.close();
			fetchRun();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '取消失败');
	}
}

async function startPolling() {
	if (isPolling.value) return;
	isPolling.value = true;
	pollAbort = new AbortController();
	const signal = pollAbort.signal;
	while (!signal.aborted) {
		run.value = await pipelineRunApi.get(runId);
		if (isTerminalStatus(run.value.status)) {
			fetchArtifacts();
			break;
		}
		await delayAsync(2000);
	}
	isPolling.value = false;
}

function stopPolling() {
	pollAbort?.abort();
	pollAbort = null;
	isPolling.value = false;
}

function togglePolling() {
	if (isPolling.value) stopPolling();
	else startPolling();
}

onMounted(async () => {
	await fetchRun();
	fetchArtifacts();
});

onUnmounted(() => {
	stopPolling();
	logPollAbort?.abort();
});
</script>
