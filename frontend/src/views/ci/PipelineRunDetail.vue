<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">流水线详情</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="run?.status === 'faulted'"
					:disabled="retrying"
					class="app-button-primary h-9 px-3"
					@click="handleRetry"
				>
					<RotateCcw class="size-4" />
					重试
				</button>
				<button
					v-if="run?.status === 'waiting_to_run' || run?.status === 'running'"
					:disabled="canceling"
					class="app-button-destructive h-9 px-3"
					@click="isCancelDialogOpen = true"
				>
					<X class="size-4" />
					取消
				</button>
				<button
					v-if="run && !isTerminalStatus(run.status)"
					class="app-button inline-flex h-9 items-center gap-2 px-3"
					:class="isPolling ? 'text-primary' : ''"
					@click="togglePolling"
				>
					<Loader2 class="h-4 w-4" :class="isPolling ? 'animate-spin' : ''" />
					{{ isPolling ? '自动刷新中' : '已暂停刷新' }}
				</button>
				<button class="app-button h-9 px-4" @click="router.push('/ci/run')">
					<ArrowLeft class="size-4" />
					返回
				</button>
			</div>
		</div>

		<!-- 加载状态 -->
		<AppSpinner v-if="loading" class="py-12" />

		<!-- 内容 -->
		<div v-else-if="run" class="flex flex-col gap-4">
			<!-- 基本信息卡片 -->
			<div class="app-surface">
				<div class="app-section-header">
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
							<AppBadge variant="pill" :tone="pipelineStatusTone">
								{{ pipelineStatusLabel }}
							</AppBadge>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">代码仓库</dt>
						<dd>
							<router-link :to="`/ci/repository/${run.repository_id}`" class="app-link">
								{{ run.repository_name }}
							</router-link>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">触发方式</dt>
						<dd>
							<AppBadge variant="pill">
								{{ run.trigger }}
							</AppBadge>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">触发分支</dt>
						<dd class="text-foreground">{{ run.trigger_ref }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">模板</dt>
						<dd>
							<router-link :to="`/ci/template/${run.template_id}`" class="app-link">
								{{ run.template_name }} v{{ run.template_version }}
							</router-link>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">快照</dt>
						<dd>
							<router-link
								v-if="run.snapshot_id"
								:to="`/ci/snapshot/${run.snapshot_id}`"
								class="app-link"
							>
								查看
							</router-link>
							<span v-else class="text-muted-foreground">—</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">重试自</dt>
						<dd>
							<router-link v-if="run.retry_of" :to="`/ci/run/${run.retry_of}`" class="app-link">
								查看
							</router-link>
							<span v-else class="text-muted-foreground">—</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(run.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">开始时间</dt>
						<dd class="text-muted-foreground">
							{{ run.started_at ? formatTime(run.started_at) : '—' }}
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">结束时间</dt>
						<dd class="text-muted-foreground">
							{{ run.finished_at ? formatTime(run.finished_at) : '—' }}
						</dd>
					</div>
					<div v-if="run.error_message" class="flex gap-2 sm:col-span-2">
						<dt class="w-24 shrink-0 text-muted-foreground">错误信息</dt>
						<dd class="min-w-0 text-destructive">
							<span class="block truncate" :title="run.error_message">
								{{ run.error_message }}
							</span>
						</dd>
					</div>
				</dl>
			</div>

			<!-- Stage 列表 -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="font-semibold text-foreground">阶段编排</h2>
					<ViewModeToggle v-model="stagesView" />
				</div>

				<!-- 列表视图 -->
				<div v-if="stagesView === 'list'">
					<div class="overflow-x-auto">
						<table class="app-table-detail min-w-[960px]">
							<thead>
								<tr>
									<th>#</th>
									<th>阶段</th>
									<th>版本</th>
									<th>依赖</th>
									<th>制品</th>
									<th>状态</th>
									<th>错误信息</th>
									<th>操作</th>
								</tr>
							</thead>
							<tbody>
								<tr v-if="run.snapshot_id && !snapshot">
									<td colspan="8" class="text-center text-muted-foreground">
										<AppSpinner />
									</td>
								</tr>
								<tr v-else-if="!snapshot || snapshot.stages_snapshot.length === 0">
									<td colspan="8" class="text-center text-muted-foreground">暂无阶段记录</td>
								</tr>
								<tr v-for="(stage, index) in snapshot?.stages_snapshot ?? []" :key="stage.id">
									<td class="text-muted-foreground">{{ index + 1 }}</td>
									<td>
										<router-link :to="`/ci/build-stage/${stage.id}`" class="app-link">
											{{ stage.name }}
										</router-link>
									</td>
									<td class="text-foreground">v{{ stage.version }}</td>
									<td>
										<div v-if="stage.depends_on.length" class="flex flex-wrap gap-1">
											<AppBadge v-for="depId in stage.depends_on" :key="depId">
												{{ snapshotStageMap[depId]?.name ?? depId }}
											</AppBadge>
										</div>
										<span v-else class="text-muted-foreground">—</span>
									</td>
									<td class="text-foreground">
										{{ stage.artifacts?.length ? stage.artifacts.length : '—' }}
									</td>
									<td>
										<AppBadge
											variant="pill"
											:tone="stageStatusTone(stageRunMap[stage.id]?.status ?? 'waiting_to_run')"
										>
											{{ stageStatusLabel(stageRunMap[stage.id]?.status ?? 'waiting_to_run') }}
										</AppBadge>
									</td>
									<td class="max-w-xs">
										<span
											v-if="stageRunMap[stage.id]?.error_message"
											class="block truncate text-destructive"
											:title="stageRunMap[stage.id]?.error_message"
										>
											{{ stageRunMap[stage.id]?.error_message }}
										</span>
										<span v-else class="text-muted-foreground">—</span>
									</td>
									<td>
										<button
											v-if="stageRunMap[stage.id]"
											class="app-link"
											@click="openStageLog(stage.id)"
										>
											查看日志
										</button>
										<span v-else class="text-muted-foreground">—</span>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>

				<!-- DAG 视图 -->
				<div v-else-if="stagesView === 'dag'" class="p-6">
					<div
						v-if="!snapshot?.stages_snapshot || snapshot.stages_snapshot.length === 0"
						class="text-center text-muted-foreground"
					>
						暂无阶段记录
					</div>
					<div v-else class="h-[500px]">
						<StageDAGView
							:stages="snapshot.stages_snapshot"
							:stage-runs="stageRuns"
							:animated="true"
							@view-stage="openLogDrawer"
						/>
					</div>
				</div>
				<!-- 无效状态 -->
				<div v-else class="p-6 text-center text-destructive">无效的视图模式</div>
			</div>

			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">变量快照</h2>
				</div>
				<VariableDeclarationsTable :declarations="runVariableDeclarations" :readonly="true" />
			</div>

			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">制品</h2>
				</div>
				<AppSpinner v-if="artifactsLoading" class="py-16" />
				<AppEmptyState
					v-else-if="!isTerminalStatus(run.status)"
					message="运行完成后展示"
					size="compact"
				/>
				<AppEmptyState v-else-if="artifacts.length === 0" size="compact" />
				<div v-else class="overflow-x-auto">
					<table class="app-table-detail min-w-[760px]">
						<thead>
							<tr>
								<th>阶段</th>
								<th>类型</th>
								<th>名称</th>
								<th>路径</th>
								<th>创建时间</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="artifact in artifacts" :key="artifact.id">
								<td class="text-foreground">{{ artifact.stage_name }}</td>
								<td>
									<AppBadge>
										{{ artifact.type }}
									</AppBadge>
								</td>
								<td class="text-foreground">{{ artifact.name }}</td>
								<td class="max-w-md truncate text-muted-foreground">
									{{ artifact.path || '—' }}
								</td>
								<td class="text-muted-foreground">
									{{ formatTime(artifact.created_at) }}
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<AppDrawer
			:open="showLogsDrawer"
			:title="`${currentStageRun?.stage_name ?? ''} - 日志`"
			width-class="w-[min(960px,100vw)]"
			body-class="min-h-0 flex-1 overflow-hidden p-0"
			@update:open="handleLogDrawerOpenChange"
		>
			<div class="flex h-full flex-col gap-3 p-6">
				<div v-if="logsText" class="min-h-0 flex-1">
					<MonacoEditor
						:model-value="logsText"
						language="plaintext"
						height="100%"
						:readonly="true"
					/>
				</div>
				<div v-else class="flex flex-1 items-center justify-center text-muted-foreground">
					<div class="text-center">
						<AppSpinner v-if="stageLogStatus === 'loading' || stageLogStatus === 'streaming'" />
						<p v-if="stageLogStatus === 'loading'" class="mt-2">加载日志中...</p>
						<p v-else-if="stageLogStatus === 'streaming'" class="mt-2">等待日志输出...</p>
						<p v-else-if="stageLogStatus === 'empty'">暂无日志输出</p>
						<div v-else-if="stageLogStatus === 'error'">
							<p class="text-destructive">{{ stageLogError || '日志加载失败' }}</p>
							<button type="button" class="app-link mt-2 text-sm" @click="retryStageLog">
								重试
							</button>
						</div>
					</div>
				</div>
			</div>
		</AppDrawer>

		<AppDialog
			v-model:open="isCancelDialogOpen"
			title="确认取消"
			width-class="w-[min(420px,calc(100vw-32px))]"
		>
			<p class="text-sm text-foreground">确定要取消此流水线运行吗？</p>
			<template #footer>
				<button type="button" class="app-button" @click="isCancelDialogOpen = false">取消</button>
				<button
					type="button"
					:disabled="canceling"
					class="app-button-destructive"
					@click="handleCancel"
				>
					确认
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { ArrowLeft, Loader2, RotateCcw, X } from 'lucide-vue-next';
	import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { pipelineRunApi, pipelineTemplateApi } from '@/api/ci';
	import AppDialog from '@/components/AppDialog.vue';
	import AppBadge from '@/components/AppBadge.vue';
	import AppEmptyState from '@/components/AppEmptyState.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import AppDrawer from '@/components/AppDrawer.vue';
	import ViewModeToggle from '@/components/ViewModeToggle.vue';
	import MonacoEditor from '@/components/MonacoEditor.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { PipelineSnapshot, SnapshotStage } from '@/types/ci/snapshot';
	import type { PipelineRun } from '@/types/ci/run';
	import type { Artifact, StageRun } from '@/types/ci/stage_run';
	import { isTerminalStatus, statusTone } from '@/utils/status';
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
	const isCancelDialogOpen = ref(false);
	const stagesView = ref<'list' | 'dag'>('list');
	const stageLogStatus = ref<'loading' | 'streaming' | 'done' | 'empty' | 'error'>('loading');
	const stageLogError = ref('');

	const runVariableDeclarations = computed(() => run.value?.variables_snapshot ?? []);
	const stageRuns = computed(() => run.value?.stage_runs ?? []);
	const stageRunMap = computed<Record<string, StageRun>>(() => {
		const map: Record<string, StageRun> = {};
		for (const sr of stageRuns.value) {
			map[sr.stage_id] = sr;
		}
		return map;
	});
	const snapshotStageMap = computed<Record<string, SnapshotStage>>(() => {
		const map: Record<string, SnapshotStage> = {};
		for (const stage of snapshot.value?.stages_snapshot ?? []) {
			map[stage.id] = stage;
		}
		return map;
	});

	// 日志 drawer 状态
	const logsText = ref('');
	let logPollAbort: AbortController | null = null;

	let pollAbort: AbortController | null = null;
	const isPolling = ref(false);

	const pipelineStatusTone = computed(() => (run.value ? statusTone(run.value.status) : 'default'));
	const pipelineStatusLabel = computed(() => (run.value ? run.value.status : ''));

	function stageStatusTone(status: string) {
		return status === 'skipped' ? 'default' : statusTone(status);
	}

	function stageStatusLabel(status: string) {
		if (status === 'skipped') {
			return '跳过';
		}
		return status;
	}

	function openLogDrawer(sr: StageRun) {
		logPollAbort?.abort();
		currentStageRun.value = sr;
		logsText.value = '';
		stageLogStatus.value = 'loading';
		stageLogError.value = '';
		showLogsDrawer.value = true;
		startLogPolling(sr.id);
	}

	function openStageLog(stageId: string) {
		const stageRun = stageRunMap.value[stageId];
		if (stageRun) {
			openLogDrawer(stageRun);
		}
	}

	function closeLogDrawer() {
		showLogsDrawer.value = false;
		logPollAbort?.abort();
		logPollAbort = null;
	}

	function handleLogDrawerOpenChange(open: boolean) {
		if (open) {
			showLogsDrawer.value = true;
			return;
		}
		closeLogDrawer();
	}

	function retryStageLog() {
		if (!currentStageRun.value) {
			return;
		}
		logPollAbort?.abort();
		logsText.value = '';
		stageLogStatus.value = 'loading';
		stageLogError.value = '';
		startLogPolling(currentStageRun.value.id);
	}

	async function startLogPolling(stageRunId: string) {
		logPollAbort = new AbortController();
		const signal = logPollAbort.signal;
		stageLogStatus.value = 'loading';
		let offset = 0;

		while (!signal.aborted) {
			try {
				const resp = await pipelineRunApi.getStageLog(runId.value, stageRunId, offset);
				if (signal.aborted) {
					break;
				}
				if (currentStageRun.value?.id !== stageRunId) {
					break;
				}
				if (resp.logs) {
					logsText.value += resp.logs;
					offset = resp.offset;
					stageLogStatus.value = resp.is_complete ? 'done' : 'streaming';
				} else if (resp.is_complete) {
					stageLogStatus.value = logsText.value ? 'done' : 'empty';
				} else {
					stageLogStatus.value = 'streaming';
				}
				if (resp.is_complete) {
					break;
				}
			} catch (error) {
				if (!signal.aborted) {
					stageLogStatus.value = 'error';
					stageLogError.value = error instanceof Error ? error.message : '日志加载失败';
				}
				break;
			}
			await delayAsync(1500);
		}
	}

	async function fetchRun() {
		try {
			return await execute(async () => {
				const data = await pipelineRunApi.get(runId.value);
				run.value = data;
				return data;
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
			// artifact 加载失败不影响主流程
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
				const currentRun = await fetchRun();
				if (currentRun && isTerminalStatus(currentRun.status)) {
					await fetchArtifacts();
				}
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
					void fetchArtifacts();
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

	function resetState() {
		stopPolling();
		run.value = undefined;
		snapshot.value = undefined;
		artifacts.value = [];
	}

	async function handleRunStatus(currentRun: PipelineRun | undefined) {
		if (!currentRun) {
			return;
		}
		if (isTerminalStatus(currentRun.status)) {
			await fetchArtifacts();
		} else {
			startPolling();
		}
	}

	async function init() {
		resetState();
		const currentRun = await fetchRun();
		if (currentRun?.snapshot_id) {
			await fetchSnapshot(currentRun.snapshot_id);
		}
		await handleRunStatus(currentRun);
	}

	watch(runId, init);

	onMounted(init);

	onUnmounted(() => {
		stopPolling();
		logPollAbort?.abort();
	});
</script>
