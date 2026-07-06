<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-xl font-semibold text-foreground">
        {{ template?.name || t('pipelineTemplate.detailTitle') }}
      </h1>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="template" class="app-button-primary h-9 px-3" @click="openEditInfoModal">
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
        <button
          v-if="template && (isDirty || hasStageUpdates)"
          :disabled="saving"
          class="app-button-primary h-9 px-3"
          @click="handleSave"
        >
          <Save class="size-4" />
          {{ hasStageUpdates && !isDirty ? t('pipelineTemplate.updated') : t('common.save') }}
        </button>
        <button v-if="template" class="app-button h-9 px-3" @click="openRunModal">
          <Play class="size-4" />
          {{ t('pipelineTemplate.runPipeline') }}
        </button>
        <button
          v-if="template"
          :disabled="duplicating"
          class="app-button h-9 px-3"
          @click="handleDuplicate"
        >
          <Copy class="size-4" />
          {{ t('common.copy') }}
        </button>
        <button v-if="template" class="app-button-danger h-9 px-3" @click="openDeleteModal">
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/ci/template')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <AppSpinner v-if="status === 'loading'" class="py-12" />

    <!-- Content -->
    <div v-else-if="template" class="flex flex-col gap-4">
      <!-- Basic Info Card -->
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('pipelineTemplate.basicInfo') }}</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('pipelineTemplate.templateName') }}
            </dt>
            <dd class="text-foreground">{{ template.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('pipelineTemplate.version') }}</dt>
            <dd class="text-foreground">v{{ template.version }}</dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.description') }}</dt>
            <dd class="text-foreground">
              {{ template.description || t('pipelineTemplate.noDescription') }}
            </dd>
          </div>
        </dl>
      </div>

      <!-- Stage Orchestration -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('pipelineTemplate.stageOrchestration') }}
          </h2>
          <div class="flex items-center gap-2">
            <ViewModeToggle v-model="viewMode" />
            <button class="app-button-primary h-8 px-3" @click="openAddOrchModal">
              <Plus class="size-4" />
              {{ t('pipelineTemplate.addStage') }}
            </button>
          </div>
        </div>
        <!-- List View -->
        <div v-if="viewMode === 'list'" class="overflow-x-auto">
          <table class="app-table-detail min-w-[760px]">
            <thead>
              <tr>
                <th>{{ t('pipelineTemplate.stage') }}</th>
                <th>{{ t('pipelineTemplate.version') }}</th>
                <th>{{ t('pipelineTemplate.dependency') }}</th>
                <th>{{ t('pipelineTemplate.artifact') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="sortableOrch.length === 0">
                <td colspan="5" class="text-center text-muted-foreground">
                  {{ t('pipelineTemplate.noStageHint') }}
                </td>
              </tr>
              <tr v-for="(orch, idx) in sortableOrch" :key="orch.stage_id">
                <td>
                  <div class="flex items-center gap-2">
                    <router-link :to="`/ci/build-stage/${orch.stage_id}`" class="app-link">
                      {{ stageCache[orch.stage_id]?.name ?? orch.stage_name }}
                    </router-link>
                    <AppBadge
                      v-if="
                        stageCache[orch.stage_id] &&
                        stageCache[orch.stage_id].version > orch.stage_version
                      "
                      tone="warning"
                    >
                      {{ t('pipelineTemplate.updated') }}
                    </AppBadge>
                  </div>
                </td>
                <td class="text-foreground">v{{ orch.stage_version }}</td>
                <td>
                  <div v-if="orch.depends_on.length > 0" class="flex flex-wrap gap-1">
                    <AppBadge v-for="depId in orch.depends_on" :key="depId">
                      {{ stageCache[depId]?.name ?? depId }}
                    </AppBadge>
                  </div>
                  <span v-else class="text-muted-foreground">—</span>
                </td>
                <td class="text-foreground">
                  {{ stageCache[orch.stage_id]?.artifacts?.length ?? '—' }}
                </td>
                <td>
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openEditOrchModal(idx)">
                      {{ t('common.edit') }}
                    </button>
                    <button class="app-link-danger" @click="confirmRemoveOrch(idx)">
                      {{ t('common.remove') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- DAG View -->
        <div v-else-if="viewMode === 'dag'" class="p-6">
          <div v-if="dagStages.length === 0" class="text-center text-muted-foreground">
            {{ t('pipelineTemplate.noStageHint') }}
          </div>
          <div v-else class="h-[500px]">
            <StageDAGView :stages="dagStages" :animated="true" />
          </div>
        </div>
        <!-- Invalid State -->
        <div v-else class="p-6 text-center text-destructive">
          {{ t('pipelineTemplate.invalidViewMode') }}
        </div>
      </div>

      <!-- {{ t('pipelineTemplate.variableDeclarations') }} -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('pipelineTemplate.variableDeclarations') }}
          </h2>
          <button class="app-button-primary h-8 px-3" @click="openAddVarModal">
            <Plus class="size-4" />
            {{ t('pipelineTemplate.addVariable') }}
          </button>
        </div>
        <VariableDeclarationsTable
          :declarations="declarations"
          :readonly="false"
          @edit="openEditVarModal"
          @delete="confirmDeleteVariable"
        />
      </div>

      <!-- {{ t('pipelineTemplate.artifactDeclarations') }} -->
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">
            {{ t('pipelineTemplate.artifactDeclarations') }}
          </h2>
        </div>
        <AppEmptyState v-if="artifactDeclarations.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-table-detail min-w-[720px]">
            <thead>
              <tr>
                <th>Stage</th>
                <th>{{ t('common.type') }}</th>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('pipelineTemplate.pathOrImage') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(artifact, idx) in artifactDeclarations" :key="idx">
                <td class="text-foreground">{{ artifact.stageName }}</td>
                <td>
                  <AppBadge>
                    {{ artifact.type }}
                  </AppBadge>
                </td>
                <td class="text-foreground">{{ artifact.name }}</td>
                <td class="text-muted-foreground">{{ artifact.path || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <AppDialog v-model:open="isEditInfoDialogOpen" :title="t('pipelineTemplate.editBasicInfo')">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.templateName') }}</label>
        <input v-model="editForm.name" type="text" class="app-input" />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('common.description') }}</label>
        <textarea v-model="editForm.description" rows="3" class="app-textarea"></textarea>
      </div>
      <template #footer>
        <button class="app-button" @click="cancelEditInfo">{{ t('common.cancel') }}</button>
        <button :disabled="saving" class="app-button-primary" @click="handleEditInfoOk">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isAddOrchDialogOpen" :title="t('pipelineTemplate.addStage')">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.selectStage') }}</label>
        <ComboboxSelect
          :model-value="addOrchForm.stageId"
          :options="stageSelectOptions"
          :portal="false"
          :placeholder="t('pipelineTemplate.searchStage')"
          :empty-text="t('pipelineTemplate.noStageFound')"
          @update:model-value="handleStageSelection"
        />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('pipelineTemplate.dependsOnStageOptional') }}
        </label>
        <div class="space-y-2">
          <label v-for="orch in sortableOrch" :key="orch.stage_id" class="flex items-center gap-2">
            <input
              v-model="addOrchForm.dependsOn"
              :value="orch.stage_id"
              type="checkbox"
              class="app-checkbox"
            />
            <span class="text-sm text-foreground">{{ orch.stage_name }}</span>
          </label>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isAddOrchDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="!addOrchForm.stageId" class="app-button-primary" @click="confirmAddOrch">
          {{ t('common.add') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isEditOrchDialogOpen" :title="t('pipelineTemplate.editDependencies')">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.dependsOnStage') }}</label>
        <div class="space-y-2">
          <label
            v-for="orch in editableOrchOptions"
            :key="orch.stage_id"
            class="flex items-center gap-2"
          >
            <input
              v-model="editOrchForm.dependsOn"
              :value="orch.stage_id"
              type="checkbox"
              class="app-checkbox"
            />
            <span class="text-sm text-foreground">{{ orch.stage_name }}</span>
          </label>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isEditOrchDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" @click="confirmEditOrch">{{ t('common.save') }}</button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isRunDialogOpen" :title="t('pipelineTemplate.runPipeline')">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.selectRepository') }}</label>
        <ComboboxSelect
          :model-value="runForm.repositoryId"
          :options="repoSelectOptions"
          :portal="false"
          :placeholder="t('pipelineTemplate.searchRepository')"
          :empty-text="t('pipelineTemplate.noRepositoryFound')"
          @update:model-value="handleRepoSelection"
        />
        <p
          v-if="selectedRepository && !selectedRepository.git_credential_id"
          class="mt-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive"
        >
          {{ t('pipelineTemplate.repositoryMissingCredential') }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.triggerBranch') }}</label>
        <input
          v-model="runForm.triggerRef"
          type="text"
          :placeholder="selectedRepository?.default_branch || 'main'"
          class="app-input"
        />
      </div>
      <template #footer>
        <button class="app-button" @click="isRunDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button
          :disabled="!runForm.repositoryId || !selectedRepository?.git_credential_id || running"
          class="app-button-primary"
          @click="handleRunOk"
        >
          {{ t('pipelineTemplate.runPipeline') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isAddVarDialogOpen" :title="t('pipelineTemplate.addVariableTitle')">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.variableName') }}</label>
        <input v-model="varForm.name" type="text" class="app-input" />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.variableValue') }}</label>
        <input v-model="varForm.value" type="text" class="app-input" />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.variableDescription') }}</label>
        <input v-model="varForm.description" type="text" class="app-input" />
      </div>
      <template #footer>
        <button class="app-button" @click="isAddVarDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" @click="handleAddVarOk">{{ t('common.add') }}</button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isEditVarDialogOpen" :title="t('pipelineTemplate.editVariableTitle')">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.variableName') }}</label>
        <input v-model="varForm.name" type="text" disabled class="app-input" />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.variableValue') }}</label>
        <input v-model="varForm.value" type="text" class="app-input" />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.variableDescription') }}</label>
        <input v-model="varForm.description" type="text" class="app-input" />
      </div>
      <template #footer>
        <button class="app-button" @click="isEditVarDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" @click="handleEditVarOk">{{ t('common.save') }}</button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('pipelineTemplate.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">{{ t('pipelineTemplate.deleteTemplateConfirm') }}</p>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="deleting" class="app-button-destructive" @click="handleDeleteOk">
          {{ t('pipelineTemplate.confirmDeleteAction') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteOrchDialogOpen"
      :title="t('pipelineTemplate.confirmRemove')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">{{ t('pipelineTemplate.removeStageConfirm') }}</p>
      <template #footer>
        <button class="app-button" @click="isDeleteOrchDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" @click="removeOrch">
          {{ t('pipelineTemplate.confirmRemoveAction') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteVarDialogOpen"
      :title="t('pipelineTemplate.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{ t('pipelineTemplate.deleteVariableConfirm', { name: varToDelete }) }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteVarDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" @click="deleteVariable">
          {{ t('pipelineTemplate.confirmDeleteAction') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Copy, Pencil, Play, Plus, Save, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
  // import { VueDraggable } from 'vue-draggable-plus';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { buildStageApi, pipelineTemplateApi, repositoryApi } from '@/api/ci';
  import AppDialog from '@/components/AppDialog.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ViewModeToggle from '@/components/ViewModeToggle.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { BuildStageResp } from '@/gen/proto/orbit/build_stage';
  import type { RepositoryResp } from '@/gen/proto/orbit/repository';
  import type { PipelineTemplateResp, StageOrchestrationResp } from '@/gen/proto/orbit/template';
  import type { VariableDeclarationResp } from '@/gen/proto/orbit/common';
  import { detectCircularDependencies } from '@/utils/dag';
  import StageDAGView from './components/StageDAGView.vue';
  import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

  interface ArtifactDeclaration {
    stageName: string;
    type: string;
    name: string;
    path: string;
  }

  const route = useRoute();
  const router = useRouter();
  const templateId = computed(() => route.params.id as string);
  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();

  const { status, execute } = useStatusAsync();
  const { loading: saving, execute: executeSave } = useStatusAsync();
  const { loading: deleting, execute: executeDelete } = useStatusAsync();
  const { loading: running, execute: executeRun } = useStatusAsync();
  const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

  const template = ref<PipelineTemplateResp>();
  const sortableOrch = ref<StageOrchestrationResp[]>([]);
  const declarations = ref<VariableDeclarationResp[]>([]);
  const stageCache = reactive<Record<string, BuildStageResp>>({});
  const viewMode = ref<'list' | 'dag'>('list');

  const stageOptions = ref<BuildStageResp[]>([]);

  const stageSelectOptions = computed(() => {
    const inOrch = new Set(sortableOrch.value.map((o) => o.stage_id));
    return stageOptions.value
      .filter((stage) => !inOrch.has(stage.id))
      .map((stage) => ({
        value: stage.id,
        label: stage.name,
        description: `v${stage.version} · ${stage.image}`,
      }));
  });

  const repoOptions = ref<RepositoryResp[]>([]);
  const repoSelectOptions = computed(() =>
    repoOptions.value.map((repo) => ({
      value: repo.id,
      label: repo.name,
      description: repo.git_credential_id
        ? repo.repository_url
        : t('pipelineTemplate.missingGitCredential'),
    }))
  );

  // Saved snapshots for dirty detection
  const savedOrch = ref<string>('[]');
  const savedDeclarations = ref<string>('[]');

  const isDirty = computed(() => {
    const orchStr = JSON.stringify(sortableOrch.value.map((o, i) => ({ ...o, sort_order: i })));
    const declStr = JSON.stringify(declarations.value);
    return orchStr !== savedOrch.value || declStr !== savedDeclarations.value;
  });

  const hasStageUpdates = computed(() =>
    sortableOrch.value.some((orch) => {
      const stage = stageCache[orch.stage_id];
      return stage && stage.version > orch.stage_version;
    })
  );

  const artifactDeclarations = computed(() => {
    const result: ArtifactDeclaration[] = [];
    for (const orch of sortableOrch.value) {
      const stage = stageCache[orch.stage_id];
      for (const artifact of stage?.artifacts ?? []) {
        result.push({
          stageName: stage?.name ?? orch.stage_name,
          type: artifact.type,
          name: artifact.name,
          path: artifact.path,
        });
      }
    }
    return result;
  });

  const isEditInfoDialogOpen = ref(false);
  const isAddOrchDialogOpen = ref(false);
  const isEditOrchDialogOpen = ref(false);
  const isRunDialogOpen = ref(false);
  const isAddVarDialogOpen = ref(false);
  const isEditVarDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeleteOrchDialogOpen = ref(false);
  const isDeleteVarDialogOpen = ref(false);

  const editForm = reactive({ name: '', description: '' });
  const addOrchForm = reactive({
    stageId: '',
    dependsOn: [] as string[],
  });
  const runForm = reactive({
    repositoryId: '',
    triggerRef: '',
  });
  const editOrchForm = reactive({
    editingStageId: '',
    dependsOn: [] as string[],
  });
  const varForm = reactive({
    name: '',
    value: '',
    description: '',
  });
  const orchToDelete = ref(-1);
  const varToDelete = ref('');

  const editableOrchOptions = computed(() =>
    sortableOrch.value.filter((o) => o.stage_id !== editOrchForm.editingStageId)
  );

  const selectedRepository = computed(() =>
    repoOptions.value.find((r) => r.id === runForm.repositoryId)
  );

  const dagStages = computed(() =>
    sortableOrch.value.map((orch) => {
      const stage = stageCache[orch.stage_id];
      return {
        id: orch.stage_id,
        name: stage?.name ?? orch.stage_id,
        image: stage?.image ?? '',
        script: stage?.script ?? '',
        version: stage?.version ?? 1,
        depends_on: orch.depends_on,
        artifacts: stage?.artifacts ?? [],
      };
    })
  );

  // Watch repository selection and fill default branch
  watch(
    () => runForm.repositoryId,
    (newRepoId) => {
      if (newRepoId) {
        const repo = repoOptions.value.find((r) => r.id === newRepoId);
        if (repo?.default_branch) {
          runForm.triggerRef = repo.default_branch;
        }
      }
    }
  );

  function applyTemplateState(tmpl: PipelineTemplateResp) {
    template.value = tmpl;
    sortableOrch.value = [...tmpl.orchestration].sort((a, b) => a.sort_order - b.sort_order);
    declarations.value = [...tmpl.variable_declarations];
    // Update saved snapshots
    savedOrch.value = JSON.stringify(sortableOrch.value.map((o, i) => ({ ...o, sort_order: i })));
    savedDeclarations.value = JSON.stringify(declarations.value);
    Object.keys(stageCache).forEach((key) => delete stageCache[key]);
    tmpl.stages.forEach((stage) => (stageCache[stage.id] = stage));
    Object.assign(editForm, {
      name: tmpl.name,
      description: tmpl.description ?? '',
    });
  }

  async function fetchTemplate() {
    try {
      await execute(async () => {
        const tmpl = await pipelineTemplateApi.get(templateId.value);
        applyTemplateState(tmpl);
      });
    } catch {
      toast.error(t('pipelineTemplate.toast.loadDetailFailed'));
      router.push('/ci/template');
    }
  }

  async function searchStages() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('pipelineTemplate.toast.selectProjectRequired'));
      return;
    }
    try {
      const resp = await buildStageApi.list({
        per_page: 100,
        project_id: projectId,
      });
      stageOptions.value = resp.items;
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : t('pipelineTemplate.toast.loadStagesFailed')
      );
    }
  }

  function handleStageSelection(value: ComboboxOptionValue) {
    addOrchForm.stageId = String(value || '');
    const stage = stageOptions.value.find((item) => item.id === addOrchForm.stageId);
    if (stage) {
      stageCache[stage.id] = stage;
    }
  }

  async function syncDeclarations() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('pipelineTemplate.toast.selectProjectRequired'));
      return;
    }
    try {
      const resp = await pipelineTemplateApi.resolveVariables(
        {
          orchestration: sortableOrch.value.map((item, index) => ({
            ...item,
            sort_order: index,
          })),
          variable_declarations: declarations.value,
        },
        { project_id: projectId }
      );
      declarations.value = resp.items;
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('pipelineTemplate.toast.syncVariablesFailed')
      );
    }
  }

  function openEditInfoModal() {
    Object.assign(editForm, {
      name: template.value?.name ?? '',
      description: template.value?.description ?? '',
    });
    isEditInfoDialogOpen.value = true;
  }

  function cancelEditInfo() {
    Object.assign(editForm, {
      name: template.value?.name ?? '',
      description: template.value?.description ?? '',
    });
    isEditInfoDialogOpen.value = false;
  }

  async function handleEditInfoOk() {
    if (!editForm.name.trim()) {
      toast.error(t('pipelineTemplate.validation.nameNotEmpty'));
      return;
    }

    try {
      await executeSave(async () => {
        const data = await pipelineTemplateApi.update(templateId.value, {
          name: editForm.name,
          description: editForm.description,
        });
        applyTemplateState(data);
        toast.success(t('pipelineTemplate.toast.saveSuccess'));
      });
      isEditInfoDialogOpen.value = false;
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('pipelineTemplate.toast.saveFailed'));
    }
  }

  async function handleSave() {
    const orchForCheck = sortableOrch.value.map((o) => ({
      name: o.stage_id,
      depends_on: o.depends_on,
    }));
    const cycle = detectCircularDependencies(orchForCheck);
    if (cycle) {
      toast.error(t('pipelineTemplate.cycleDetected', { cycle: cycle.join(' → ') }));
      return;
    }

    try {
      await executeSave(async () => {
        // Auto-update stage_version in orchestration
        for (const orch of sortableOrch.value) {
          const stage = stageCache[orch.stage_id];
          if (stage && stage.version > orch.stage_version) {
            orch.stage_version = stage.version;
          }
        }

        const data = await pipelineTemplateApi.update(templateId.value, {
          name: editForm.name,
          description: editForm.description,
          orchestration: {
            items: sortableOrch.value.map((o, i) => ({
              ...o,
              sort_order: i,
            })),
          },
          variable_declarations: { items: declarations.value },
        });
        applyTemplateState(data);
        toast.success(t('pipelineTemplate.toast.saveSnapshotSuccess', { version: data.version }));
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('pipelineTemplate.toast.saveFailed'));
    }
  }

  async function handleDuplicate() {
    try {
      await executeDuplicate(async () => {
        const newTemplate = await pipelineTemplateApi.duplicate(templateId.value, {});
        toast.success(t('pipelineTemplate.toast.duplicateSuccess'));
        router.push(`/ci/template/${newTemplate.id}`);
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('pipelineTemplate.toast.duplicateFailed'));
    }
  }

  function openDeleteModal() {
    isDeleteDialogOpen.value = true;
  }

  async function handleDeleteOk() {
    try {
      await executeDelete(async () => {
        await pipelineTemplateApi.delete(templateId.value);
        toast.success(t('pipelineTemplate.toast.deleteSuccess'));
        router.push('/ci/template');
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('pipelineTemplate.toast.deleteFailed'));
    }
  }

  // ── Orchestration operations ────────────────────────────────────────────────────

  async function openAddOrchModal() {
    addOrchForm.stageId = '';
    addOrchForm.dependsOn = [];
    isAddOrchDialogOpen.value = true;
    await searchStages();
  }

  async function confirmAddOrch() {
    if (!addOrchForm.stageId) {
      return;
    }
    const stage =
      stageOptions.value.find((s) => s.id === addOrchForm.stageId) ??
      stageCache[addOrchForm.stageId];
    if (!stage) {
      return;
    }
    const newOrch: StageOrchestrationResp = {
      stage_id: addOrchForm.stageId,
      stage_name: stage.name,
      stage_version: stage.version,
      depends_on: addOrchForm.dependsOn,
      sort_order: sortableOrch.value.length,
    };
    const orchForCheck = [...sortableOrch.value, newOrch].map((o) => ({
      name: o.stage_id,
      depends_on: o.depends_on,
    }));
    const cycle = detectCircularDependencies(orchForCheck);
    if (cycle) {
      toast.error(t('pipelineTemplate.cycleDetected', { cycle: cycle.join(' → ') }));
      return;
    }
    sortableOrch.value.push(newOrch);
    stageCache[stage.id] = stage;
    await syncDeclarations();
    isAddOrchDialogOpen.value = false;
  }

  function confirmRemoveOrch(idx: number) {
    orchToDelete.value = idx;
    isDeleteOrchDialogOpen.value = true;
  }

  async function removeOrch() {
    const idx = orchToDelete.value;
    if (idx === -1) {
      return;
    }
    const removed = sortableOrch.value[idx];
    sortableOrch.value.splice(idx, 1);
    for (const o of sortableOrch.value) {
      o.depends_on = o.depends_on.filter((depId) => depId !== removed.stage_id);
    }
    await syncDeclarations();
    isDeleteOrchDialogOpen.value = false;
    orchToDelete.value = -1;
  }

  function openEditOrchModal(idx: number) {
    const orch = sortableOrch.value[idx];
    if (!orch) {
      return;
    }
    editOrchForm.editingStageId = orch.stage_id;
    editOrchForm.dependsOn = [...orch.depends_on];
    isEditOrchDialogOpen.value = true;
  }

  function confirmEditOrch() {
    const orch = sortableOrch.value.find((o) => o.stage_id === editOrchForm.editingStageId);
    if (!orch) {
      return;
    }
    const orchForCheck = sortableOrch.value.map((o) => ({
      name: o.stage_id,
      depends_on:
        o.stage_id === editOrchForm.editingStageId ? editOrchForm.dependsOn : o.depends_on,
    }));
    const cycle = detectCircularDependencies(orchForCheck);
    if (cycle) {
      toast.error(t('pipelineTemplate.cycleDetected', { cycle: cycle.join(' → ') }));
      return;
    }
    orch.depends_on = editOrchForm.dependsOn;
    editOrchForm.editingStageId = '';
    editOrchForm.dependsOn = [];
    isEditOrchDialogOpen.value = false;
  }

  // ── Run pipeline ────────────────────────────────────────────────────────────────

  async function searchRepos() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('pipelineTemplate.toast.selectProjectRequired'));
      return;
    }
    try {
      const resp = await repositoryApi.list({
        per_page: 100,
        project_id: projectId,
      });
      repoOptions.value = resp.items;
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : t('pipelineTemplate.toast.loadRepositoriesFailed')
      );
    }
  }

  function handleRepoSelection(value: ComboboxOptionValue) {
    runForm.repositoryId = String(value || '');
  }

  async function openRunModal() {
    if (isDirty.value) {
      toast.error(t('pipelineTemplate.toast.unsavedChanges'));
      return;
    }
    runForm.repositoryId = '';
    runForm.triggerRef = '';
    await searchRepos();
    isRunDialogOpen.value = true;
  }

  async function handleRunOk() {
    if (!runForm.repositoryId) {
      toast.error(t('pipelineTemplate.toast.selectRepositoryRequired'));
      return;
    }

    const repo = selectedRepository.value;
    if (!repo?.git_credential_id) {
      toast.error(t('pipelineTemplate.toast.repositoryMissingCredential'));
      return;
    }

    try {
      await executeRun(async () => {
        const triggerRef = runForm.triggerRef || repo.default_branch || 'main';
        const run = await repositoryApi.trigger(runForm.repositoryId, {
          template_id: templateId.value,
          trigger_ref: triggerRef,
          variables: {},
        });
        toast.success(t('pipelineTemplate.toast.runTriggered'));
        isRunDialogOpen.value = false;
        router.push(`/ci/run/${run.id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('pipelineTemplate.toast.runFailed'));
    }
  }

  // ── Variable management ─────────────────────────────────────────────────────────

  function openAddVarModal() {
    varForm.name = '';
    varForm.value = '';
    varForm.description = '';
    isAddVarDialogOpen.value = true;
  }

  function handleAddVarOk() {
    if (!varForm.name.trim()) {
      toast.error(t('pipelineTemplate.validation.variableNameRequired'));
      return;
    }

    // Check duplicates
    if (declarations.value.some((d) => d.name === varForm.name)) {
      toast.error(t('pipelineTemplate.validation.variableNameExists'));
      return;
    }

    declarations.value.push({
      name: varForm.name,
      description: varForm.description,
      default: undefined,
      value: varForm.value || undefined,
      secret: false,
      source: 'template_custom',
      editable: true,
    });

    isAddVarDialogOpen.value = false;
  }

  function openEditVarModal(name: string) {
    const decl = declarations.value.find((d) => d.name === name);
    if (!decl) {
      return;
    }
    varForm.name = decl.name;
    varForm.value = String(decl.value ?? '');
    varForm.description = decl.description ?? '';
    isEditVarDialogOpen.value = true;
  }

  function handleEditVarOk() {
    const decl = declarations.value.find((d) => d.name === varForm.name);
    if (decl) {
      decl.value = varForm.value || undefined;
      decl.description = varForm.description;
    }
    isEditVarDialogOpen.value = false;
  }

  function confirmDeleteVariable(name: string) {
    varToDelete.value = name;
    isDeleteVarDialogOpen.value = true;
  }

  function deleteVariable() {
    const name = varToDelete.value;
    if (!name) {
      return;
    }
    const idx = declarations.value.findIndex((d) => d.name === name);
    if (idx !== -1) {
      declarations.value.splice(idx, 1);
    }
    isDeleteVarDialogOpen.value = false;
    varToDelete.value = '';
  }

  watch(templateId, fetchTemplate);
  onMounted(fetchTemplate);
  onUnmounted(() => {
    // Cleanup if needed
  });
</script>
