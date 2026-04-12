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
					class="btn btn-sm btn-ghost gap-1 text-primary"
				>
					<Loader2 class="size-4 animate-spin" />
					自动刷新中
				</button>
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/run')">
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
							<dt class="text-base-content/70 w-24 shrink-0">Repository</dt>
							<dd>
								<router-link
									:to="`/ci/repository/${run.repository_id}`"
									class="link link-primary text-xs"
								>
									{{ run.repository_name }}
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
							<dt class="text-base-content/70 w-24 shrink-0">模板</dt>
							<dd class="flex items-center gap-2">
								<router-link
									:to="`/ci/template/${run.template_id}`"
									class="link link-primary text-xs"
								>
									{{ run.template_name }} v{{ run.template_version }}
								</router-link>
								<router-link
									v-if="run.snapshot_id"
									:to="`/ci/snapshot/${run.snapshot_id}`"
									class="link link-primary text-xs"
								>
									查看快照
								</router-link>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">重试自</dt>
							<dd>
								<router-link
									v-if="run.retry_of"
									:to="`/ci/run/${run.retry_of}`"
									class="link link-primary text-xs"
								>
									{{ run.retry_of }}
								</router-link>
								<span v-else class="text-base-content/60">—</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
							<dd>{{ formatTime(run.created_at) }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">开始时间</dt>
							<dd>{{ run.started_at ? formatTime(run.started_at) : '—' }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">结束时间</dt>
							<dd>{{ run.finished_at ? formatTime(run.finished_at) : '—' }}</dd>
						</div>
						<div v-if="run.error_message" class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">错误信息</dt>
							<dd>
								<span class="tooltip tooltip-top cursor-help" :data-tip="run.error_message">
									<span class="text-xs truncate block max-w-xs">{{ run.error_message }}</span>
								</span>
							</dd>
						</div>
					</dl>
				</div>
			</div>

			<!-- Stages 编排与变量快照 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<div class="flex items-center gap-3">
							<h2 class="font-semibold">Stages 编排</h2>
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
					</div>

					<!-- Stages 编排内容 -->
					<table v-if="stagesView === 'list'" class="table w-full">
						<thead>
							<tr class="text-base-content/60 text-xs">
								<th class="w-8">#</th>
								<th>Stage 名称</th>
								<th class="w-20">版本</th>
								<th>依赖</th>
								<th class="w-24">状态</th>
								<th>错误信息</th>
								<th class="w-16">日志</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="!snapshot && run.snapshot_id">
								<td colspan="7" class="text-center py-8 text-base-content/60">
									<span class="loading loading-spinner loading-sm" />
								</td>
							</tr>
							<tr v-else-if="!snapshot || snapshot.stages_snapshot.length === 0">
								<td colspan="7" class="text-center py-8 text-base-content/60">暂无数据</td>
							</tr>
							<tr
								v-for="(stage, idx) in snapshot?.stages_snapshot ?? []"
								:key="stage.id"
								class="hover"
							>
								<td class="text-base-content/40 text-xs">{{ idx + 1 }}</td>
								<td class="text-xs">{{ stage.name }}</td>
								<td class="text-center">
									<span class="badge badge-sm badge-ghost">v{{ stage.version }}</span>
								</td>
								<td>
									<div v-if="stage.depends_on.length" class="flex items-center gap-1 flex-wrap">
										<span
											v-for="depId in stage.depends_on"
											:key="depId"
											class="text-xs bg-base-200 rounded px-2 py-0.5 text-base-content/70"
										>
											{{ snapshotStageMap[depId]?.name ?? depId }}
										</span>
									</div>
									<span v-else class="text-base-content/40 text-xs">—</span>
								</td>
								<td>
									<span
										v-if="stageRuns.find((sr) => sr.stage_id === stage.id)"
										class="badge badge-xs"
										:class="
											statusBadgeClass(stageRuns.find((sr) => sr.stage_id === stage.id)!.status)
										"
									>
										{{ statusLabel(stageRuns.find((sr) => sr.stage_id === stage.id)!.status) }}
									</span>
									<span v-else class="text-base-content/40 text-xs">—</span>
								</td>
								<td class="max-w-xs">
									<span
										v-if="stageRuns.find((sr) => sr.stage_id === stage.id)?.error_message"
										class="tooltip tooltip-top cursor-help"
										:data-tip="stageRuns.find((sr) => sr.stage_id === stage.id)?.error_message"
									>
										<span class="text-xs truncate block max-w-xs">
											{{ stageRuns.find((sr) => sr.stage_id === stage.id)?.error_message }}
										</span>
									</span>
									<span v-else class="text-base-content/40 text-xs">—</span>
								</td>
								<td>
									<button
										v-if="stageRuns.find((sr) => sr.stage_id === stage.id)"
										class="link link-primary text-xs"
										@click="openLogDrawer(stageRuns.find((sr) => sr.stage_id === stage.id)!)"
									>
										查看
									</button>
									<span v-else class="text-base-content/40 text-xs">—</span>
								</td>
							</tr>
						</tbody>
					</table>

					<div v-else class="min-h-[300px]">
						<div v-if="!snapshot" class="flex justify-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</div>
						<template v-else-if="snapshot.stages_snapshot.length > 0">
							<StageDAGView
								:key="snapshot.id"
								:stages="snapshot.stages_snapshot"
								:stage-runs="stageRuns"
								:show-minimap="true"
								@view-stage="openLogDrawer"
							/>
							<p class="text-xs text-base-content/50 mt-2">点击节点查看日志</p>
						</template>
						<div v-else class="text-center py-8 text-base-content/60 text-sm">暂无数据</div>
					</div>

					<!-- 变量快照内容 -->
					<div class="mt-6 pt-6 border-t border-base-300">
						<h3 class="font-semibold mb-4">变量快照</h3>
						<VariableDeclarationsTable :declarations="runVariableDeclarations" :readonly="true" />
					</div>
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
						暂无数据
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
							Stage: {{ currentStageRun?.stage_name }}
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
						<div v-else class="flex-1 bg-[#1a202c] rounded-box p-4 overflow-auto min-h-64">
							<pre
								v-if="logsText"
								class="text-gray-300 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all"
								>{{ logsText }}</pre
							>
							<div v-else class="flex flex-col items-center gap-2 py-8 text-gray-500">
								<FileX class="size-8" />
								<span class="text-sm">暂无数据</span>
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
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineRunApi, pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Artifact } from '@/types/ci/stage_run';
import type { PipelineSnapshot, SnapshotStage } from '@/types/ci/snapshot';
import type { PipelineRun } from '@/types/ci/run';
import type { StageRun } from '@/types/ci/stage_run';
import { isTerminalStatus, statusBadgeClass, statusLabel } from '@/utils/status';
import { delayAsync, formatTime } from '@/utils/time';
import StageDAGView from './components/StageDAGView.vue';
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

const route = useRoute();
const router = useRouter();
const runId = computed(() => route.params.id as string);
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: artifactsLoading, execute: executeArtifacts } = useStatusAsync();
const { loading: retrying, execute: executeRetry } = useStatusAsync();
const { loading: canceling, execute: executeCancel } = useStatusAsync();

const run = ref<PipelineRun>();
const snapshot = ref<PipelineSnapshot>();
const artifacts = ref<Artifact[]>([]);
const currentStageRun = ref<StageRun>();
const showLogsDrawer = ref(false);
const cancelModalRef = ref<HTMLDialogElement>();
const stagesView = ref<'list' | 'dag'>('list');

const runVariableDeclarations = computed(() => run.value?.variables_snapshot ?? []);
const stageRuns = computed(() => run.value?.stage_runs ?? []);
const snapshotStageMap = computed<Record<string, SnapshotStage>>(() => {
	const map: Record<string, SnapshotStage> = {};
	for (const s of snapshot.value?.stages_snapshot ?? []) {
		map[s.id] = s;
	}
	return map;
});

// 日志 drawer 状态
const logsText = ref('');
const logsLoading = ref(false);
let logPollAbort: AbortController | null = null;

let pollAbort: AbortController | null = null;

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
			const resp = await pipelineRunApi.getStageLog(runId.value, stageRunId, offset);
			// 如果用户已切换到其他 stage，丢弃过期响应
			if (currentStageRun.value?.id !== stageRunId) {
				break;
			}
			if (resp.logs) {
				logsText.value += resp.logs;
				offset = resp.offset;
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

async function fetchArtifacts() {
	try {
		await executeArtifacts(async () => {
			artifacts.value = await pipelineRunApi.listArtifacts(runId.value);
		});
	} catch {
		/* silent */
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
			cancelModalRef.value?.close();
			await fetchRun();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '取消失败');
	}
}

async function startPolling() {
	pollAbort = new AbortController();
	const signal = pollAbort.signal;
	while (!signal.aborted) {
		try {
			run.value = await pipelineRunApi.get(runId.value);
			if (isTerminalStatus(run.value.status)) {
				fetchArtifacts();
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
}

async function init() {
	stopPolling();
	run.value = undefined;
	snapshot.value = undefined;
	artifacts.value = [];
	await fetchRun();
	fetchArtifacts();
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
