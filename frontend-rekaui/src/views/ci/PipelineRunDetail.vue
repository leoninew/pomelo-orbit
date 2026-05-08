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
					{{ retrying ? '重试中...' : '重试' }}
				</button>
				<button
					v-if="run?.status === 'waiting_to_run' || run?.status === 'running'"
					:disabled="canceling"
					class="app-button-destructive h-9 px-3"
					@click="isCancelDialogOpen = true"
				>
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
				<button class="app-button h-9 px-4" @click="router.push('/ci/run')">返回</button>
			</div>
		</div>

		<!-- 加载状态 -->
		<div v-if="loading" class="flex items-center justify-center py-12">
			<div
				class="h-8 w-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary"
			></div>
		</div>

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
							<router-link :to="`/ci/repository/${run.repository_id}`" class="app-link">
								{{ run.repository_name }}
							</router-link>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">触发方式</dt>
						<dd>
							<span
								class="inline-block rounded-full bg-muted/50 px-2 py-0.5 text-xs font-medium text-foreground"
							>
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
					<div
						v-if="snapshot?.stages_snapshot && snapshot.stages_snapshot.length > 0"
						class="flex gap-1 rounded-md border border-border bg-background p-1"
					>
						<button
							class="rounded px-3 py-1 text-xs font-medium transition-colors"
							:class="
								stagesView === 'list'
									? 'bg-primary text-primary-foreground'
									: 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
							"
							@click="stagesView = 'list'"
						>
							列表
						</button>
						<button
							class="rounded px-3 py-1 text-xs font-medium transition-colors"
							:class="
								stagesView === 'dag'
									? 'bg-primary text-primary-foreground'
									: 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
							"
							@click="stagesView = 'dag'"
						>
							DAG
						</button>
					</div>
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
										<span
											class="inline-block size-5 animate-spin rounded-full border-2 border-primary/20 border-t-primary"
										/>
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
											<span
												v-for="depId in stage.depends_on"
												:key="depId"
												class="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground"
											>
												{{ snapshotStageMap[depId]?.name ?? depId }}
											</span>
										</div>
										<span v-else class="text-muted-foreground">—</span>
									</td>
									<td class="text-foreground">
										{{ stage.artifacts?.length ? stage.artifacts.length : '—' }}
									</td>
									<td>
										<span
											class="inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium"
											:class="
												getStageStatusBadge(stageRunMap[stage.id]?.status ?? 'waiting_to_run')
											"
										>
											{{ getStageStatusText(stageRunMap[stage.id]?.status ?? 'waiting_to_run') }}
										</span>
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
				<div
					v-else-if="snapshot?.stages_snapshot && snapshot.stages_snapshot.length > 0"
					class="p-6"
				>
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
				<div v-if="artifactsLoading" class="flex justify-center py-16">
					<div
						class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary"
					/>
				</div>
				<div
					v-else-if="!isTerminalStatus(run.status)"
					class="text-center py-16 text-muted-foreground"
				>
					<p class="text-sm">运行完成后展示</p>
				</div>
				<div v-else-if="artifacts.length === 0" class="text-center py-16 text-muted-foreground">
					<p class="text-sm">暂无制品</p>
				</div>
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
									<span
										class="inline-block rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground"
									>
										{{ artifact.type }}
									</span>
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
			<div
				ref="logContainer"
				class="h-full overflow-y-auto bg-muted/30 p-4 font-mono text-xs text-foreground"
			>
				<pre v-if="logsText" class="whitespace-pre-wrap">{{ logsText }}</pre>
				<div v-else class="flex h-full items-center justify-center text-muted-foreground">
					<div class="text-center">
						<Loader2 class="mx-auto h-8 w-8 animate-spin" />
						<p class="mt-2">加载日志中...</p>
					</div>
				</div>
			</div>
		</AppDrawer>

		<AppDialog
			v-model:open="isCancelDialogOpen"
			title="确认取消"
			description="确定要取消此流水线运行吗？"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button type="button" class="app-button" @click="isCancelDialogOpen = false">取消</button>
				<button
					type="button"
					:disabled="canceling"
					class="app-button-destructive"
					@click="handleCancel"
				>
					{{ canceling ? '取消中...' : '确认取消' }}
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { Loader2 } from 'lucide-vue-next';
	import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { pipelineRunApi, pipelineTemplateApi } from '@/api/ci';
	import AppDialog from '@/components/AppDialog.vue';
	import AppDrawer from '@/components/AppDrawer.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { PipelineSnapshot, SnapshotStage } from '@/types/ci/snapshot';
	import type { PipelineRun } from '@/types/ci/run';
	import type { Artifact, StageRun } from '@/types/ci/stage_run';
	import { isTerminalStatus } from '@/utils/status';
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
	const logsLoading = ref(false);
	const logContainer = ref<HTMLDivElement>();
	let logPollAbort: AbortController | null = null;

	let pollAbort: AbortController | null = null;
	const isPolling = ref(false);

	const statusBadgeClass = computed(() => {
		const status = run.value?.status;
		if (!status) {
			return 'bg-muted/50 text-muted-foreground';
		}
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
		if (!status) {
			return '';
		}
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
