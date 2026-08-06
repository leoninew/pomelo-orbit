<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ t('pipelineRun.detailTitle') }}
        </h1>
        <DetailHeaderMeta v-if="run">
          <AppBadge variant="status" :tone="pipelineStatusTone">{{ run.status }}</AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="run?.status === 'faulted'"
          :disabled="retrying"
          class="app-button-primary h-9 px-3"
          @click="handleRetry"
        >
          <RotateCcw class="size-4" />
          {{ t('pipelineRun.retry') }}
        </button>
        <button
          v-if="run?.status === 'waiting_to_run' || run?.status === 'running'"
          :disabled="canceling"
          class="app-button-destructive h-9 px-3"
          @click="openCancelDialog"
        >
          <X class="size-4" />
          {{ t('common.cancel') }}
        </button>
        <button
          v-if="run && !isTerminalStatus(run.status)"
          class="app-button inline-flex h-9 items-center gap-2 px-3"
          :class="isPolling ? 'text-primary' : ''"
          @click="togglePolling"
        >
          <Loader2 class="h-4 w-4" :class="isPolling ? 'animate-spin' : ''" />
          {{ isPolling ? t('pipelineRun.autoRefreshing') : t('pipelineRun.refreshPaused') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/pipeline-run')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <AppLoadingState v-if="loading" size="section" />

    <!-- Content -->
    <div v-else-if="run" class="flex flex-col gap-4">
      <!-- Basic Info Card -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('pipelineRun.basicInfo') }}</h2>
        </div>
        <dl class="app-detail-info-grid">
          <div v-if="run.snapshot_id" class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.runId') }}</dt>
            <dd class="min-w-0 text-foreground">{{ runId }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="pill" :tone="pipelineStatusTone">
                {{ run.status }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.repository') }}
            </dt>
            <dd>
              <router-link :to="`/repository/${run.repository_id}`" class="app-link">
                {{ run.repository_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.triggerType') }}
            </dt>
            <dd>
              <AppBadge variant="pill">
                {{ run.trigger }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.triggerBranch') }}
            </dt>
            <dd class="text-foreground">{{ run.trigger_ref }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.template') }}
            </dt>
            <dd>
              <router-link :to="`/pipeline/template/${run.template_id}`" class="app-link">
                {{ run.template_name }} v{{ run.template_version }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.snapshot') }}
            </dt>
            <dd>
              <router-link :to="`/pipeline/snapshot/${run.snapshot_id}`" class="app-link">
                {{ t('application.view') }}
              </router-link>
            </dd>
          </div>
          <div v-if="run.retry_of" class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.retryOf') }}
            </dt>
            <dd>
              <router-link :to="`/pipeline-run/${run.retry_of}`" class="app-link">
                {{ t('application.view') }}
              </router-link>
            </dd>
          </div>
          <div v-if="run.started_at" class="flex gap-2">
            <dt>{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(run.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.startTime') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(run.started_at) }}</dd>
          </div>
          <div v-if="run.finished_at" class="flex gap-2">
            <dt>
              {{ t('pipelineRun.fields.endTime') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(run.finished_at) }}</dd>
          </div>
          <div v-if="run.error_message" class="flex gap-2 sm:col-span-2">
            <dt>
              {{ t('pipelineRun.fields.errorMessage') }}
            </dt>
            <dd class="min-w-0 text-destructive">
              <span class="block truncate" :title="run.error_message">
                {{ run.error_message }}
              </span>
            </dd>
          </div>
        </dl>
      </div>

      <!-- Stage List -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('pipelineRun.stageOrchestration') }}
          </h2>
          <ViewModeToggle v-model="stagesView" />
        </div>

        <AppLoadingState v-if="run.snapshot_id && !snapshot" size="compact" />
        <AppEmptyState
          v-else-if="!snapshot || snapshot.stages_snapshot.length === 0"
          size="compact"
        />

        <!-- List View -->
        <div v-else-if="stagesView === 'list'">
          <div class="overflow-x-auto">
            <table class="app-data-table min-w-[960px]">
              <thead>
                <tr>
                  <th>#</th>
                  <th>{{ t('pipelineRun.stage') }}</th>
                  <th>{{ t('pipelineRun.fields.version') }}</th>
                  <th>{{ t('pipelineRun.dependency') }}</th>
                  <th>{{ t('pipelineRun.artifact') }}</th>
                  <th>{{ t('common.status') }}</th>
                  <th>{{ t('pipelineRun.fields.errorMessage') }}</th>
                  <th>{{ t('common.operation') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(stage, index) in snapshot?.stages_snapshot ?? []" :key="stage.id">
                  <td class="text-muted-foreground">{{ index + 1 }}</td>
                  <td>
                    <router-link :to="`/pipeline/stage/${stage.id}`" class="app-link">
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
                  </td>
                  <td class="text-foreground">
                    {{ stage.artifacts?.length }}
                  </td>
                  <td>
                    <AppBadge
                      variant="pill"
                      :tone="stageStatusTone(stageRunMap[stage.id]?.status ?? 'waiting_to_run')"
                    >
                      {{ stageRunMap[stage.id]?.status ?? 'waiting_to_run' }}
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
                  </td>
                  <td>
                    <button
                      v-if="stageRunMap[stage.id]"
                      class="app-link"
                      @click="openStageLog(stage.id)"
                    >
                      {{ t('pipelineRun.viewLog') }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- DAG View -->
        <div v-else-if="stagesView === 'dag'" class="p-6">
          <div class="h-[500px]">
            <StageDAGView
              :stages="snapshot.stages_snapshot"
              :stage-runs="stageRuns"
              :animated="true"
              @view-stage="openLogDrawer"
            />
          </div>
        </div>
        <!-- Invalid State -->
        <div v-else class="p-6 text-center text-destructive">
          {{ t('pipelineRun.invalidViewMode') }}
        </div>
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('pipelineRun.variableSnapshot') }}
          </h2>
        </div>
        <VariableDeclarationsTable :declarations="runVariableDeclarations" :readonly="true" />
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('pipelineRun.artifacts') }}</h2>
        </div>
        <AppLoadingState v-if="artifactsLoading" />
        <AppEmptyState
          v-else-if="!isTerminalStatus(run.status)"
          :message="t('pipelineRun.artifactsAfterCompletion')"
          size="compact"
        />
        <AppEmptyState v-else-if="artifacts.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[640px]">
            <thead>
              <tr>
                <th>{{ t('pipelineRun.stage') }}</th>
                <th>Collector</th>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('common.createdAt') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="artifact in artifacts" :key="artifact.id">
                <td class="text-foreground">{{ artifact.stage_name }}</td>
                <td>
                  <AppBadge variant="pill">
                    {{ artifact.collector }}
                  </AppBadge>
                </td>
                <td>
                  <router-link :to="`/pipeline-run/artifact/${artifact.id}`" class="app-link">
                    {{ artifact.name }}
                  </router-link>
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
      :title="`${currentStageRunResp?.stage_name ?? ''} - ${t('pipelineRun.log')}`"
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
            squared
          />
        </div>
        <div v-else class="flex flex-1 items-center justify-center text-muted-foreground">
          <div class="text-center">
            <AppSpinner v-if="stageLogStatus === 'loading' || stageLogStatus === 'streaming'" />
            <p v-if="stageLogStatus === 'loading'" class="mt-2">
              {{ t('pipelineRun.logLoading') }}
            </p>
            <p v-else-if="stageLogStatus === 'streaming'" class="mt-2">
              {{ t('pipelineRun.logWaiting') }}
            </p>
            <p v-else-if="stageLogStatus === 'empty'">{{ t('pipelineRun.noLogOutput') }}</p>
            <div v-else-if="stageLogStatus === 'error'">
              <p class="text-destructive">{{ stageLogError || t('pipelineRun.logLoadFailed') }}</p>
              <button type="button" class="app-link mt-2 text-sm" @click="retryStageLog">
                {{ t('pipelineRun.retry') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </AppDrawer>

    <AppDialog
      v-model:open="isCancelDialogOpen"
      :title="t('pipelineRun.confirmCancel')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">{{ t('pipelineRun.cancelConfirm') }}</p>
      <p v-if="cancelSubmitError" class="app-field-error mt-3" role="alert">
        {{ cancelSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="canceling"
          variant="destructive"
          @cancel="isCancelDialogOpen = false"
          @confirm="handleCancel"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Loader2, RotateCcw, X } from 'lucide-vue-next';
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { pipelineTemplateApi } from '@/api/pipeline/template';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import ViewModeToggle from '@/components/ViewModeToggle.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ArtifactResp } from '@/gen/proto/orbit/v1/pipeline_run/artifact';
  import type { PipelineRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import type { PipelineStageRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_stage_run';
  import type {
    PipelineSnapshotResp,
    SnapshotStageResp,
  } from '@/gen/proto/orbit/v1/pipeline/snapshot';
  import { isTerminalStatus, statusTone } from '@/utils/status';
  import { delayAsync, formatTime } from '@/utils/time';
  import StageDAGView from '@/views/pipeline/components/StageDAGView.vue';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  const route = useRoute();
  const router = useRouter();
  const runId = computed(() => route.params.id as string);
  const { t } = useI18n();
  const toast = useToast();

  const { loading, execute } = useStatusAsync();
  const { loading: artifactsLoading, execute: executeArtifacts } = useStatusAsync();
  const { loading: retrying, execute: executeRetry } = useStatusAsync();
  const { loading: canceling, execute: executeCancel } = useStatusAsync();

  const run = ref<PipelineRunResp>();
  const snapshot = ref<PipelineSnapshotResp>();
  const artifacts = ref<ArtifactResp[]>([]);
  const currentStageRunResp = ref<PipelineStageRunResp>();
  const showLogsDrawer = ref(false);
  const isCancelDialogOpen = ref(false);
  const cancelSubmitError = ref('');
  const stagesView = ref<'list' | 'dag'>('list');
  const stageLogStatus = ref<'loading' | 'streaming' | 'done' | 'empty' | 'error'>('loading');
  const stageLogError = ref('');

  const runVariableDeclarations = computed(() => run.value?.variables_snapshot ?? []);
  const stageRuns = computed(() => run.value?.pipeline_stage_runs ?? []);
  const stageRunMap = computed<Record<string, PipelineStageRunResp>>(() => {
    const map: Record<string, PipelineStageRunResp> = {};
    for (const sr of stageRuns.value) {
      map[sr.stage_id] = sr;
    }
    return map;
  });
  const snapshotStageMap = computed<Record<string, SnapshotStageResp>>(() => {
    const map: Record<string, SnapshotStageResp> = {};
    for (const stage of snapshot.value?.stages_snapshot ?? []) {
      map[stage.id] = stage;
    }
    return map;
  });

  // Log drawer state
  const logsText = ref('');
  let logPollAbort: AbortController | null = null;

  let pollAbort: AbortController | null = null;
  const isPolling = ref(false);

  const pipelineStatusTone = computed(() => (run.value ? statusTone(run.value.status) : 'default'));
  function stageStatusTone(status: string) {
    return status === 'skipped' ? 'default' : statusTone(status);
  }

  function openLogDrawer(sr: PipelineStageRunResp) {
    logPollAbort?.abort();
    currentStageRunResp.value = sr;
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
    if (!currentStageRunResp.value) {
      return;
    }
    logPollAbort?.abort();
    logsText.value = '';
    stageLogStatus.value = 'loading';
    stageLogError.value = '';
    startLogPolling(currentStageRunResp.value.id);
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
        if (currentStageRunResp.value?.id !== stageRunId) {
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
          stageLogError.value =
            error instanceof Error ? error.message : t('pipelineRun.logLoadFailed');
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
      toast.error(t('pipelineRun.toast.loadDetailFailed'));
      router.push('/pipeline-run');
    }
  }

  async function fetchSnapshot(snapshotId: string) {
    try {
      const data = await pipelineTemplateApi.getSnapshot(snapshotId);
      snapshot.value = data;
    } catch {
      // Snapshot load failure does not block the main flow
    }
  }

  async function fetchArtifacts() {
    try {
      await executeArtifacts(async () => {
        const resp = await pipelineRunApi.listArtifacts(runId.value);
        artifacts.value = resp.items;
      });
    } catch {
      // Artifact load failure does not block the main flow
    }
  }

  async function handleRetry() {
    try {
      await executeRetry(async () => {
        const newRun = await pipelineRunApi.retry(runId.value, {});
        toast.success(t('pipelineRun.toast.retrySuccess'));
        router.push(`/pipeline-run/${newRun.id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('pipelineRun.toast.retryFailed'));
    }
  }

  async function handleCancel() {
    cancelSubmitError.value = '';
    try {
      await executeCancel(async () => {
        await pipelineRunApi.cancel(runId.value, {});
        toast.success(t('pipelineRun.toast.cancelSuccess'));
        isCancelDialogOpen.value = false;
        const currentRun = await fetchRun();
        if (currentRun && isTerminalStatus(currentRun.status)) {
          await fetchArtifacts();
        }
      });
    } catch (error) {
      cancelSubmitError.value =
        error instanceof Error ? error.message : t('pipelineRun.toast.cancelFailed');
    }
  }

  function openCancelDialog() {
    cancelSubmitError.value = '';
    isCancelDialogOpen.value = true;
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
        // Silently retry on transient network failures without interrupting polling
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

  async function handleRunStatus(currentRun: PipelineRunResp | undefined) {
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
