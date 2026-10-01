<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader :items="[]" :title="t('pipelineRun.detailTitle')" />
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
          v-if="run && !isComplete(run.status)"
          :disabled="canceling"
          class="app-button-destructive h-9 px-3"
          @click="openCancelDialog"
        >
          <X class="size-4" />
          {{ t('common.cancel') }}
        </button>
        <button
          v-if="run && isComplete(run.status)"
          :disabled="deleting"
          class="app-button-danger h-9 px-3"
          @click="openDeleteDialog"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button
          v-if="run && !isComplete(run.status)"
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

    <AppLoadingState v-if="loading" size="section" />

    <div v-else-if="run" class="flex flex-col gap-4">
      <DetailInfoCard :title="t('pipelineRun.basicInfo')">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.runId') }}</dt>
            <dd class="min-w-0 break-all font-mono text-xs text-foreground">{{ run.id }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>流水线</dt>
            <dd>
              <router-link :to="`/pipeline/${run.pipeline_id}`" class="app-link">
                {{ run.pipeline_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.repository') }}</dt>
            <dd>
              <router-link :to="`/repository/${run.repository_id}`" class="app-link">
                {{ run.repository_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.triggerBranch') }}</dt>
            <dd class="text-foreground">{{ run.repository_ref }}</dd>
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
            <dt>配置版本</dt>
            <dd>
              <AppBadge>v{{ run.pipeline_version }}</AppBadge>
            </dd>
          </div>
          <div v-if="run.started_at" class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.startTime') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(run.started_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(run.created_at) }}</dd>
          </div>
          <div v-if="run.started_at && run.finished_at" class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.duration') }}</dt>
            <dd class="text-muted-foreground">
              {{ formatDuration(run.started_at, run.finished_at) }}
            </dd>
          </div>
          <div v-if="run.error_message" class="flex gap-2 sm:col-span-2">
            <dt>{{ t('pipelineRun.fields.errorMessage') }}</dt>
            <dd class="min-w-0 whitespace-pre-wrap break-words text-destructive">
              {{ run.error_message }}
            </dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard title="执行配置">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.triggerType') }}</dt>
            <dd>
              <AppBadge variant="pill">{{ run.trigger }}</AppBadge>
            </dd>
          </div>
          <div v-if="run.environment_id" class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.executionEnvironment') }}</dt>
            <dd class="min-w-0">
              <router-link to="/environment" class="app-link">
                {{ t('nav.environment') }}
              </router-link>
              <span v-if="run.environment_target_type" class="text-muted-foreground">
                · {{ t(`project.environment.targetTypes.${run.environment_target_type}`) }}
              </span>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.snapshot') }}</dt>
            <dd class="min-w-0">
              <router-link :to="`/pipeline/snapshot/${run.snapshot_id}`" class="app-link break-all">
                {{ run.snapshot_id }}
              </router-link>
            </dd>
          </div>
          <div v-if="run.retry_of" class="flex gap-2">
            <dt>{{ t('pipelineRun.fields.retryOf') }}</dt>
            <dd class="min-w-0">
              <router-link :to="`/pipeline-run/${run.retry_of}`" class="app-link break-all">
                {{ run.retry_of }}
              </router-link>
            </dd>
          </div>
        </dl>
      </DetailInfoCard>

      <!-- Stage List -->
      <DetailInfoCard :title="t('pipelineRun.stageOrchestration')" actions-class="flex-nowrap">
        <template #actions>
          <SearchControl
            v-model="stageSearchText"
            :placeholder="t('pipelineRun.searchStagesPlaceholder')"
            :loading="loading"
            class="min-w-0 flex-1"
            @search="handleStageSearch"
          />
        </template>

        <ViewModeTabs v-if="snapshot?.stages_snapshot.length" v-model="stagesView" />
        <AppLoadingState v-if="run.snapshot_id && !snapshot" size="compact" />
        <AppEmptyState v-else-if="!snapshot || filteredStages.length === 0" size="compact" />

        <!-- List View -->
        <div v-else-if="stagesView === 'list'">
          <div class="overflow-x-auto">
            <table class="app-data-table min-w-[800px]">
              <thead>
                <tr>
                  <th>#</th>
                  <th>{{ t('pipelineRun.stage') }}</th>
                  <th>执行镜像</th>
                  <th>{{ t('pipelineRun.dependency') }}</th>
                  <th>{{ t('common.status') }}</th>
                  <th>{{ t('common.operation') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(stage, index) in filteredStages" :key="stage.id">
                  <td class="text-muted-foreground">{{ index + 1 }}</td>
                  <td>
                    <span class="text-foreground">{{ stage.name }}</span>
                  </td>
                  <td class="text-foreground">{{ stage.image }}</td>
                  <td>
                    <div v-if="stage.depends_on.length" class="flex flex-wrap gap-1">
                      <AppBadge v-for="depId in stage.depends_on" :key="depId">
                        {{ snapshotStageMap[depId]?.name }}
                      </AppBadge>
                    </div>
                  </td>
                  <td>
                    <div class="flex items-center gap-1.5">
                      <AppBadge
                        variant="pill"
                        :tone="stageStatusTone(stageRunMap[stage.id]?.status ?? 'waiting_to_run')"
                      >
                        {{ stageRunMap[stage.id]?.status ?? 'waiting_to_run' }}
                      </AppBadge>
                      <PopoverRoot v-if="stageRunMap[stage.id]?.error_message">
                        <PopoverTrigger as-child>
                          <button
                            type="button"
                            class="app-icon-button size-7 text-destructive hover:bg-destructive/10 hover:text-destructive"
                            :title="t('pipelineRun.viewErrorDetails')"
                            :aria-label="t('pipelineRun.viewErrorDetails')"
                          >
                            <CircleAlert class="size-4" aria-hidden="true" />
                          </button>
                        </PopoverTrigger>
                        <PopoverPortal>
                          <PopoverContent
                            side="top"
                            align="start"
                            :side-offset="6"
                            class="z-[60] max-h-64 w-[min(24rem,calc(100vw-32px))] overflow-y-auto rounded-md border border-border bg-popover p-3 text-sm text-popover-foreground shadow-lg outline-none"
                          >
                            <p class="mb-1 font-medium">
                              {{ t('pipelineRun.fields.errorMessage') }}
                            </p>
                            <p class="whitespace-pre-wrap break-words text-muted-foreground">
                              {{ stageRunMap[stage.id]?.error_message }}
                            </p>
                          </PopoverContent>
                        </PopoverPortal>
                      </PopoverRoot>
                    </div>
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
        <div v-else class="p-6">
          <div class="h-[500px]">
            <StageDAGView
              :stages="filteredStages"
              :stage-runs="stageRuns"
              :animated="true"
              @view-stage="openLogDrawer"
            />
          </div>
        </div>
      </DetailInfoCard>

      <DetailInfoCard :title="t('pipelineRun.variableSnapshot')">
        <VariableDeclarationsTable :variables="runVariables" :readonly="true" />
      </DetailInfoCard>

      <DetailInfoCard :title="t('pipelineRun.artifacts')">
        <template #actions>
          <SearchControl
            v-model="artifactSearchText"
            :placeholder="t('pipelineRun.searchArtifactsPlaceholder')"
            :loading="artifactsLoading"
            class="shrink-0"
            @search="handleArtifactSearch"
          />
        </template>
        <AppLoadingState v-if="artifactsLoading" />
        <AppEmptyState v-else-if="filteredArtifacts.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[640px]">
            <thead>
              <tr>
                <th>{{ t('pipelineRun.stage') }}</th>
                <th>收集器</th>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('common.createdAt') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="artifact in filteredArtifacts" :key="artifact.id">
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
      </DetailInfoCard>
    </div>

    <LogDrawer
      :open="showLogsDrawer"
      :title="`${currentStageRun?.stage_name ?? ''} - ${t('pipelineRun.log')}`"
      :state="stageLogState"
      @update:open="handleLogDrawerOpenChange"
    ></LogDrawer>

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

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('pipelineRun.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('pipelineRun.deleteConfirm', { id: runId }) }}
      </p>
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="deleting"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="closeDeleteDialog"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, CircleAlert, Loader2, RotateCcw, Trash2, X } from '@lucide/vue';
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui';
  import { ensureProjectExecutionReady } from '@/router/projectReadiness';
  import { pipelineApi } from '@/api/pipeline/pipeline';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import LogDrawer from '@/components/LogDrawer.vue';
  import ViewModeTabs from '@/components/ViewModeTabs.vue';
  import { provideLogStreamCache, useLogStream } from '@/composables/useLogStream';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ArtifactResp } from '@/gen/proto/orbit/v1/pipeline_run/artifact';
  import type { PipelineRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import type { PipelineStageRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_stage_run';
  import type {
    PipelineSnapshotResp,
    SnapshotStageResp,
  } from '@/gen/proto/orbit/v1/pipeline/snapshot';
  import { useProjectStore } from '@/stores/project';
  import { isComplete, statusTone } from '@/utils/status';
  import { delayAsync, formatDuration, formatTime } from '@/utils/time';
  import StageDAGView from '@/views/pipeline/components/StageDAGView.vue';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  const route = useRoute();
  const router = useRouter();
  const runId = computed(() => route.params.id as string);
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();

  const { loading, execute } = useStatusAsync();
  const { loading: artifactsLoading, execute: executeArtifacts } = useStatusAsync();
  const { loading: retrying, execute: executeRetry } = useStatusAsync();
  const { loading: canceling, execute: executeCancel } = useStatusAsync();
  const { loading: deleting, execute: executeDelete } = useStatusAsync();

  const run = ref<PipelineRunResp>();
  const snapshot = ref<PipelineSnapshotResp>();
  const artifacts = ref<ArtifactResp[]>([]);
  const currentStageRunResp = ref<PipelineStageRunResp>();
  const showLogsDrawer = ref(false);
  const isCancelDialogOpen = ref(false);
  const cancelSubmitError = ref('');
  const isDeleteDialogOpen = ref(false);
  const deleteSubmitError = ref('');
  const stagesView = ref<'list' | 'dag'>('list');
  const stageSearchText = ref('');
  const appliedStageSearch = ref('');
  const artifactSearchText = ref('');
  const appliedArtifactSearch = ref('');
  const logCache = provideLogStreamCache(() => `${projectStore.activeProjectId}:${runId.value}`);
  const stageLogResource = computed(() =>
    projectStore.activeProjectId && currentStageRunResp.value
      ? {
          projectId: projectStore.activeProjectId,
          path: `/api/pipeline-run/${runId.value}/stage/${currentStageRunResp.value.id}/log/stream`,
          kind: 'file' as const,
        }
      : undefined
  );
  const { state: stageLogState } = useLogStream(stageLogResource, showLogsDrawer);

  function selectedProjectId() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      throw new Error(t('pipelineRun.toast.loadDetailFailed'));
    }
    return projectId;
  }

  const runVariables = computed(() => run.value?.variables ?? []);
  const stageRuns = computed(() => run.value?.pipeline_stage_runs ?? []);
  const currentStageRun = computed(
    () =>
      stageRuns.value.find((stageRun) => stageRun.id === currentStageRunResp.value?.id) ??
      currentStageRunResp.value
  );
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

  const filteredStages = computed(() => {
    const stages = snapshot.value?.stages_snapshot ?? [];
    const keyword = appliedStageSearch.value.trim().toLowerCase();
    if (!keyword) {
      return stages;
    }
    return stages.filter(
      (stage) =>
        stage.name.toLowerCase().includes(keyword) || stage.image.toLowerCase().includes(keyword)
    );
  });

  const filteredArtifacts = computed(() => {
    const keyword = appliedArtifactSearch.value.trim().toLowerCase();
    if (!keyword) {
      return artifacts.value;
    }
    return artifacts.value.filter(
      (artifact) =>
        artifact.name.toLowerCase().includes(keyword) ||
        artifact.stage_name.toLowerCase().includes(keyword) ||
        artifact.collector.toLowerCase().includes(keyword)
    );
  });

  let pollAbort: AbortController | null = null;
  const isPolling = ref(false);

  const pipelineStatusTone = computed(() => (run.value ? statusTone(run.value.status) : 'default'));
  function stageStatusTone(status: string) {
    return status === 'skipped' ? 'default' : statusTone(status);
  }

  function openLogDrawer(sr: PipelineStageRunResp) {
    currentStageRunResp.value = sr;
    showLogsDrawer.value = true;
  }

  function openStageLog(stageId: string) {
    const stageRun = stageRunMap.value[stageId];
    if (stageRun) {
      openLogDrawer(stageRun);
    }
  }

  function handleLogDrawerOpenChange(open: boolean) {
    showLogsDrawer.value = open;
  }

  async function fetchRun() {
    try {
      return await execute(async () => {
        const data = await pipelineRunApi.get(selectedProjectId(), runId.value);
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
      const data = await pipelineApi.getSnapshot(selectedProjectId(), snapshotId);
      snapshot.value = data;
    } catch {
      // Snapshot load failure does not block the main flow
    }
  }

  async function fetchArtifacts(silent = false) {
    const fetch = async () => {
      const resp = await pipelineRunApi.listArtifacts(selectedProjectId(), runId.value);
      artifacts.value = resp.items;
    };

    try {
      if (silent) {
        await fetch();
      } else {
        await executeArtifacts(fetch);
      }
    } catch {
      // Artifact load failure does not block the main flow
    }
  }

  function handleStageSearch() {
    appliedStageSearch.value = stageSearchText.value;
    const snapshotId = run.value?.snapshot_id;
    if (snapshotId) {
      void fetchSnapshot(snapshotId);
    }
    void fetchRun();
  }

  function handleArtifactSearch() {
    appliedArtifactSearch.value = artifactSearchText.value;
    void fetchArtifacts();
  }

  async function handleRetry() {
    try {
      await executeRetry(async () => {
        const projectId = selectedProjectId();
        if (!(await ensureProjectExecutionReady(projectId, router, 'ci'))) {
          return;
        }
        const newRun = await pipelineRunApi.retry(projectId, runId.value, {});
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
        await pipelineRunApi.cancel(selectedProjectId(), runId.value, {});
        toast.success(t('pipelineRun.toast.cancelSuccess'));
        isCancelDialogOpen.value = false;
        const currentRun = await fetchRun();
        if (currentRun && isComplete(currentRun.status)) {
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

  function openDeleteDialog() {
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  function closeDeleteDialog() {
    isDeleteDialogOpen.value = false;
    deleteSubmitError.value = '';
  }

  async function handleDelete() {
    deleteSubmitError.value = '';
    try {
      await executeDelete(async () => {
        stopPolling();
        showLogsDrawer.value = false;
        await pipelineRunApi.delete(selectedProjectId(), runId.value);
        toast.success(t('pipelineRun.toast.deleteSuccess'));
        await router.replace('/pipeline-run');
      });
    } catch (error) {
      deleteSubmitError.value =
        error instanceof Error ? error.message : t('pipelineRun.toast.deleteFailed');
    }
  }

  async function startPolling() {
    pollAbort = new AbortController();
    const signal = pollAbort.signal;
    isPolling.value = true;
    while (!signal.aborted) {
      try {
        run.value = await pipelineRunApi.get(selectedProjectId(), runId.value);
        await fetchArtifacts(true);
        if (isComplete(run.value.status)) {
          isPolling.value = false;
          break;
        }
      } catch {
        // Silently retry on transient network failures without interrupting polling
      }
      await delayAsync(2000, signal);
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
    logCache.clear();
    showLogsDrawer.value = false;
    currentStageRunResp.value = undefined;
    run.value = undefined;
    snapshot.value = undefined;
    artifacts.value = [];
  }

  async function handleRunStatus(currentRun: PipelineRunResp | undefined) {
    if (!currentRun) {
      return;
    }
    await fetchArtifacts();
    if (!isComplete(currentRun.status)) {
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
  });
</script>
