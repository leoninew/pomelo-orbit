<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader :items="[]" :title="pipeline?.name || '流水线详情'" />
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="pipeline && !isTemplate"
          class="app-button-primary h-9 px-3"
          @click="openRunDialog"
        >
          <Play class="size-4" />
          运行
        </button>
        <button
          v-if="pipeline && isTemplate"
          class="app-button-primary h-9 px-3"
          @click="goToInstantiation"
        >
          <CopyPlus class="size-4" />
          复用
        </button>
        <button v-if="pipeline" class="app-button-danger h-9 px-3" @click="deleteOpen = true">
          <Trash2 class="size-4" />
          删除
        </button>
        <button class="app-button h-9 px-3" @click="router.push('/pipeline')">
          <ArrowLeft class="size-4" />
          返回
        </button>
      </div>
    </div>

    <AppLoadingState v-if="status === 'loading'" size="section" />
    <template v-else-if="pipeline">
      <DetailInfoCard title="基本信息" editable @edit="openInfoDialog">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>名称</dt>
            <dd class="text-foreground">{{ pipeline.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>配置版本</dt>
            <dd class="text-foreground">v{{ pipeline.version }}</dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt>说明</dt>
            <dd class="text-foreground">{{ pipeline.description || '未填写' }}</dd>
          </div>
          <template v-if="!isTemplate">
            <div class="flex gap-2">
              <dt>来源模板</dt>
              <dd class="text-foreground">
                {{ pipeline.source_template_name }} v{{ pipeline.source_template_version }}
              </dd>
            </div>
            <div class="flex gap-2">
              <dt>应用</dt>
              <dd>
                <router-link
                  v-if="hasApplicationBinding"
                  :to="`/application/${pipeline.application_id}`"
                  class="app-link"
                >
                  {{ pipeline.application_name }}
                </router-link>
                <span v-else class="text-muted-foreground">未绑定</span>
              </dd>
            </div>
            <div class="flex gap-2">
              <dt>代码仓库</dt>
              <dd>
                <router-link :to="`/repository/${pipeline.repository_id}`" class="app-link">
                  {{ pipeline.repository_name }}
                </router-link>
              </dd>
            </div>
            <div v-if="hasApplicationBinding" class="flex gap-2">
              <dt>来源版本</dt>
              <dd class="text-foreground">{{ versionStrategyLabel }}</dd>
            </div>
          </template>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard title="构建阶段">
        <template #actions>
          <button
            v-if="outdatedStages.length > 0"
            class="app-button-warning h-9 px-3"
            :disabled="saving"
            @click="updateOutdatedStages"
          >
            <RefreshCw class="size-4" :class="updatingStages ? 'animate-spin' : ''" />
            {{
              updatingStages
                ? `正在更新 ${updatedStageCount}/${stagesToUpdateCount} 个构建阶段`
                : `更新 ${outdatedStages.length} 个构建阶段`
            }}
          </button>
          <ViewModeToggle v-if="pipeline.stage_nodes.length > 0" v-model="stagesView" />
          <button class="app-button-primary h-9 px-3" @click="openStageDialog()">
            <Plus class="size-4" />
            引入阶段
          </button>
        </template>
        <AppEmptyState v-if="pipeline.stage_nodes.length === 0" size="compact" />
        <div v-else-if="stagesView === 'list'" class="overflow-x-auto">
          <table class="app-data-table min-w-[920px]">
            <thead>
              <tr>
                <th>#</th>
                <th>名称</th>
                <th>镜像</th>
                <th>来源阶段版本</th>
                <th>依赖</th>
                <th v-if="hasApplicationBinding">组件映射</th>
                <th class="w-32">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="stage in orderedStages" :key="stage.id">
                <td>{{ stage.sort_order }}</td>
                <td class="text-foreground">{{ stage.name }}</td>
                <td class="max-w-xs truncate text-foreground" :title="stage.image">
                  {{ stage.image }}
                </td>
                <td>v{{ stage.source_template_stage_version }}</td>
                <td>
                  <div class="flex flex-wrap gap-1">
                    <AppBadge v-for="dependency in stage.depends_on" :key="dependency">
                      {{ stageName(dependency) }}
                    </AppBadge>
                  </div>
                </td>
                <td v-if="hasApplicationBinding">
                  <div class="flex flex-wrap gap-1">
                    <AppBadge
                      v-for="artifact in mappedArtifacts(stage)"
                      :key="artifact.name"
                      variant="pill"
                    >
                      {{ artifact.component_name }}
                    </AppBadge>
                  </div>
                </td>
                <td>
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openStageDialog(stage)">编辑</button>
                    <button class="app-link-danger" @click="removeStage(stage)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="p-6">
          <div class="h-[500px]">
            <StageDAGView :stages="orderedStages" :animated="true" />
          </div>
        </div>
      </DetailInfoCard>

      <DetailInfoCard title="变量配置">
        <template #actions>
          <button class="app-button-primary h-9 px-3" @click="openAddVariableDialog">
            <Plus class="size-4" />
            添加自定义变量
          </button>
        </template>
        <AppLoadingState v-if="pipelineVariablePreviewLoading" size="compact" />
        <p v-else-if="pipelineVariablePreviewError" class="py-4 text-sm text-destructive">
          {{ pipelineVariablePreviewError }}
        </p>
        <VariableDeclarationsTable
          v-else
          :declarations="pipelineVariableRows"
          :readonly="false"
          @edit="openEditVariableDialog"
          @delete="deleteVariable"
        />
      </DetailInfoCard>
    </template>

    <AppDialog v-model:open="infoOpen" title="编辑流水线信息">
      <form class="space-y-4" @submit.prevent="saveInfo">
        <div class="space-y-1.5">
          <label class="app-field-label">
            名称
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="infoForm.name"
            class="app-input"
            :class="infoErrors.name ? 'app-input-error' : ''"
          />
          <p v-if="infoErrors.name" class="app-field-error" role="alert">{{ infoErrors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">说明</label>
          <input v-model="infoForm.description" class="app-input" />
        </div>
        <div v-if="!isTemplate" class="space-y-1.5">
          <label class="app-field-label">应用</label>
          <ComboboxSelect
            v-model="infoForm.applicationId"
            :options="applicationOptions"
            :disabled="!canBindApplication"
            placeholder="可选；绑定后不可更改"
            :invalid="Boolean(infoErrors.applicationId)"
            @update:model-value="infoErrors.applicationId = ''"
          />
          <p v-if="infoErrors.applicationId" class="app-field-error" role="alert">
            {{ infoErrors.applicationId }}
          </p>
        </div>
        <p v-if="infoError" class="app-field-error" role="alert">{{ infoError }}</p>
      </form>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="infoOpen = false" @confirm="saveInfo" />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="stageOpen"
      :title="editingStage ? '编辑构建阶段' : '引入构建阶段'"
      width-class="w-[min(760px,calc(100vw-32px))]"
      body-class="max-h-[72vh] space-y-4 overflow-y-auto px-6 py-4"
    >
      <form class="space-y-4" @submit.prevent="saveStage">
        <div v-if="!editingStage" class="space-y-1.5">
          <label class="app-field-label">
            阶段
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            v-model="stageForm.source_template_stage_id"
            :options="stageTemplateOptions"
            placeholder="选择阶段"
            :invalid="Boolean(stageError) && !stageForm.source_template_stage_id"
            @update:model-value="selectStageTemplate"
          />
        </div>
        <div class="space-y-1.5">
          <div class="space-y-1.5">
            <label class="app-field-label">
              名称
              <span class="text-destructive">*</span>
            </label>
            <input v-model="stageForm.name" class="app-input" />
          </div>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label">排序</label>
            <input v-model.number="stageForm.sort_order" type="number" min="0" class="app-input" />
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label">说明</label>
            <input v-model="stageForm.description" class="app-input" />
          </div>
        </div>
        <fieldset class="space-y-2">
          <legend class="app-field-label">依赖阶段</legend>
          <label
            v-for="stage in otherStages"
            :key="stage.id"
            class="flex items-center gap-2 text-sm text-foreground"
          >
            <input
              v-model="stageForm.depends_on"
              type="checkbox"
              :value="stage.id"
              class="size-4"
            />
            {{ stage.name }}
          </label>
        </fieldset>
        <p v-if="stageError" class="app-field-error" role="alert">{{ stageError }}</p>
      </form>
      <template #title-actions>
        <button
          v-if="editingStage?.latest_template_stage_version"
          type="button"
          class="app-button-warning h-8 shrink-0 px-3"
          @click="openTemplateUpdate"
        >
          更新至模板 v{{ editingStage.latest_template_stage_version }}
        </button>
      </template>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="stageOpen = false" @confirm="saveStage" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="templateUpdateOpen" title="更新至模板">
      <AppLoadingState v-if="templateUpdateLoading" size="compact" />
      <div v-else-if="templateUpdatePreview" class="space-y-3">
        <div
          v-for="difference in templateUpdatePreview.differences"
          :key="difference.field"
          class="grid gap-2 border-b border-border pb-3 text-sm last:border-0"
        >
          <p class="font-medium text-foreground">
            {{ templateUpdateFieldLabel(difference.field) }}
          </p>
          <div class="grid gap-2 sm:grid-cols-2">
            <pre
              class="whitespace-pre-wrap break-words bg-muted/40 p-2 text-xs text-muted-foreground"
              >{{ difference.current }}</pre>
            <pre class="whitespace-pre-wrap break-words bg-muted/40 p-2 text-xs text-foreground">{{
              difference.target
            }}</pre>
          </div>
        </div>
        <p class="text-sm text-muted-foreground">私有名称、说明、依赖、排序和制品配置将保留。</p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          confirm-label="更新并保存"
          :confirm-disabled="!templateUpdatePreview?.available"
          @cancel="templateUpdateOpen = false"
          @confirm="applyTemplateUpdate"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="runOpen" title="运行流水线">
      <AppLoadingState v-if="runPreviewLoading && !runPreview" size="compact" />
      <form v-else class="space-y-4" @submit.prevent="runPipeline">
        <div v-for="variable in runVariableDeclarations" :key="variable.name" class="space-y-1.5">
          <label class="app-field-label">{{ variable.name }}</label>
          <input
            v-if="variable.editable"
            v-model="runForm.variables[variable.name]"
            :type="variable.secret ? 'password' : 'text'"
            class="app-input"
            :class="runFieldErrors[variable.name] ? 'app-input-error' : ''"
            :aria-invalid="Boolean(runFieldErrors[variable.name])"
            @input="clearRunFieldError(variable.name)"
          />
          <input
            v-else
            :value="displayVariableValue(variable.value ?? variable.default)"
            type="text"
            class="app-input"
            disabled
          />
          <p v-if="runFieldErrors[variable.name]" class="app-field-error">
            {{ runFieldErrors[variable.name] }}
          </p>
        </div>
        <p v-if="runPreviewError" class="app-field-error">{{ runPreviewError }}</p>
        <p v-if="runError" class="app-field-error">{{ runError }}</p>
      </form>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          :confirm-disabled="!runPreview || runPreviewLoading || Boolean(runPreviewError)"
          @cancel="runOpen = false"
          @confirm="runPipeline"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="variableOpen"
      :title="editingVariableName ? '编辑自定义变量' : '添加自定义变量'"
    >
      <form class="space-y-4" @submit.prevent="saveVariable">
        <div class="space-y-1.5">
          <label class="app-field-label">
            变量名
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="variableForm.name"
            class="app-input"
            :disabled="Boolean(editingVariableName)"
            :class="variableError ? 'app-input-error' : ''"
          />
          <p v-if="variableError" class="app-field-error">{{ variableError }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">变量值</label>
          <input
            v-model="variableForm.value"
            :type="variableForm.secret ? 'password' : 'text'"
            class="app-input"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">说明</label>
          <input v-model="variableForm.description" class="app-input" />
        </div>
        <label class="flex items-center gap-2 text-sm text-foreground">
          <input v-model="variableForm.secret" type="checkbox" class="app-checkbox" />
          敏感变量
        </label>
      </form>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="variableOpen = false" @confirm="saveVariable" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="deleteOpen" title="删除流水线">
      <p class="text-sm text-muted-foreground">删除后不能恢复。运行和制品历史会保留其执行快照。</p>
      <p v-if="deleteError" class="app-field-error mt-3">{{ deleteError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          confirm-label="删除"
          @cancel="deleteOpen = false"
          @confirm="deletePipeline"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, CopyPlus, Play, Plus, RefreshCw, Trash2 } from '@lucide/vue';
  import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { pipelineApi } from '@/api/pipeline/pipeline';
  import { pipelineStageApi } from '@/api/pipeline/pipeline_stage';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import { useProjectStore } from '@/stores/project';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ViewModeToggle from '@/components/ViewModeToggle.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VariableDeclarationResp } from '@/gen/proto/orbit/v1/common/common';
  import type {
    PipelineStageNodeResp,
    PipelineStageTemplateUpdatePreviewResp,
  } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { PipelineResp } from '@/gen/proto/orbit/v1/pipeline/pipeline';
  import type { PipelineRunVariablePreviewResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import StageDAGView from '@/views/pipeline/components/StageDAGView.vue';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: saving, execute: executeSave } = useStatusAsync();
  const pipelineId = computed(() => String(route.params.id));
  const pipeline = ref<PipelineResp>();
  const stagesView = ref<'list' | 'dag'>('list');
  const updatingStages = ref(false);
  const updatedStageCount = ref(0);
  const stagesToUpdateCount = ref(0);
  const infoOpen = ref(false);
  const stageOpen = ref(false);
  const templateUpdateOpen = ref(false);
  const templateUpdateLoading = ref(false);
  const templateUpdatePreview = ref<PipelineStageTemplateUpdatePreviewResp>();
  const runOpen = ref(false);
  const variableOpen = ref(false);
  const deleteOpen = ref(false);
  const editingStage = ref<PipelineStageNodeResp>();
  const stageTemplates = ref<
    import('@/gen/proto/orbit/v1/pipeline/pipeline_stage').PipelineStageResp[]
  >([]);
  const infoError = ref('');
  const infoErrors = reactive({ name: '', applicationId: '' });
  const applications = ref<ApplicationResp[]>([]);
  const stageError = ref('');
  const runError = ref('');
  const runPreview = ref<PipelineRunVariablePreviewResp>();
  const runPreviewLoading = ref(false);
  const runPreviewError = ref('');
  const runFieldErrors = reactive<Record<string, string>>({});
  const pipelineVariablePreview = ref<PipelineRunVariablePreviewResp>();
  const pipelineVariablePreviewLoading = ref(false);
  const pipelineVariablePreviewError = ref('');
  const initialRunVariableValues = ref<Record<string, string>>({});
  const variableError = ref('');
  const deleteError = ref('');
  const infoForm = reactive({ name: '', description: '', applicationId: '' });
  const stageForm = reactive({
    source_template_stage_id: '',
    name: '',
    depends_on: [] as string[],
    sort_order: 0,
    description: '',
  });
  const runForm = reactive({ variables: {} as Record<string, string> });
  const variableForm = reactive({ name: '', value: '', description: '', secret: false });
  const editingVariableName = ref('');

  const isTemplate = computed(() => pipeline.value?.kind === 'template');
  const hasApplicationBinding = computed(() =>
    Boolean(pipeline.value?.application_id && pipeline.value.application_name)
  );
  const canBindApplication = computed(() => !isTemplate.value && !hasApplicationBinding.value);
  const applicationOptions = computed(() => {
    const options = applications.value.map((application) => ({
      value: application.id,
      label: application.name,
    }));
    // Ensure the currently bound application remains visible when the control is read-only.
    const currentId = pipeline.value?.application_id;
    const currentName = pipeline.value?.application_name;
    if (currentId && currentName && !options.some((option) => option.value === currentId)) {
      options.unshift({ value: currentId, label: currentName });
    }
    return options;
  });
  const orderedStages = computed(() =>
    [...(pipeline.value?.stage_nodes || [])].sort(
      (left, right) => left.sort_order - right.sort_order
    )
  );
  const outdatedStages = computed(() =>
    orderedStages.value.filter(
      (stage) =>
        stage.latest_template_stage_version !== undefined && stage.latest_template_stage_version > 0
    )
  );
  const otherStages = computed(() =>
    orderedStages.value.filter((stage) => stage.id !== editingStage.value?.id)
  );
  const stageTemplateOptions = computed(() =>
    stageTemplates.value.map((stage) => ({
      value: stage.id,
      label: `${stage.name} v${stage.version}`,
    }))
  );
  const versionStrategyLabel = computed(() =>
    pipeline.value?.version_fork_strategy === 'fixed'
      ? `固定版本 ${pipeline.value.fixed_version_label || ''}`
      : pipeline.value?.version_fork_strategy === 'latest'
        ? '最新版本'
        : '未绑定组件制品'
  );
  const runVariableDeclarations = computed(() => runPreview.value?.variable_declarations || []);
  const pipelineVariables = computed(() => pipeline.value?.variable_declarations || []);
  const pipelineCustomVariables = computed(() =>
    pipelineVariables.value.filter((variable) => variable.source === 'pipeline_custom')
  );
  const pipelineVariableRows = computed(() =>
    (pipelineVariablePreview.value?.variable_declarations || pipelineVariables.value).map(
      (variable) => ({
        ...variable,
        editable: variable.source === 'pipeline_custom',
      })
    )
  );

  function stageName(id: string) {
    return pipeline.value?.stage_nodes.find((stage) => stage.id === id)?.name || id;
  }
  function mappedArtifacts(stage: PipelineStageNodeResp) {
    return stage.artifacts.filter((artifact) => artifact.component_name);
  }
  function displayVariableValue(value: unknown) {
    return value == null ? '' : String(value);
  }

  function buildRunVariableForm() {
    const variables: Record<string, string> = {};
    for (const variable of runVariableDeclarations.value) {
      if (!variable.editable) continue;
      variables[variable.name] = runForm.variables[variable.name] || '';
    }
    return variables;
  }

  function clearRunFieldError(name: string) {
    delete runFieldErrors[name];
  }

  function setRunFieldErrorFromMessage(message: string) {
    const missing = message.match(/Missing variable value:\s*(\S+)/i);
    const requiredRef = /repository_ref is required/i.test(message);
    if (missing) {
      runFieldErrors[missing[1]] = `请输入变量值: ${missing[1]}`;
      return true;
    }
    if (requiredRef) {
      runFieldErrors.repository_ref = '请输入分支或标签';
      return true;
    }
    return false;
  }

  let runPreviewRequest = 0;
  let runPreviewTimer: ReturnType<typeof setTimeout> | undefined;
  let pipelineVariablePreviewRequest = 0;

  function previewPipelineVariables(variables: Record<string, string>) {
    return pipelineRunApi.previewVariables(pipelineId.value, {
      variables,
    });
  }

  async function loadPipelineVariablePreview() {
    if (pipeline.value?.kind !== 'application') {
      pipelineVariablePreview.value = undefined;
      pipelineVariablePreviewLoading.value = false;
      pipelineVariablePreviewError.value = '';
      return;
    }
    const request = ++pipelineVariablePreviewRequest;
    pipelineVariablePreviewLoading.value = true;
    pipelineVariablePreviewError.value = '';
    try {
      const preview = await previewPipelineVariables({});
      if (request !== pipelineVariablePreviewRequest) return;
      pipelineVariablePreview.value = preview;
    } catch (reason) {
      if (request !== pipelineVariablePreviewRequest) return;
      pipelineVariablePreview.value = undefined;
      pipelineVariablePreviewError.value =
        reason instanceof Error ? reason.message : '解析运行时变量失败';
    } finally {
      if (request === pipelineVariablePreviewRequest) pipelineVariablePreviewLoading.value = false;
    }
  }

  async function loadRunVariablePreview(initializeValues = false) {
    const request = ++runPreviewRequest;
    runPreviewLoading.value = true;
    runPreviewError.value = '';
    try {
      const preview = await previewPipelineVariables(buildRunVariableForm());
      if (request !== runPreviewRequest) return;
      runPreview.value = preview;
      if (initializeValues) {
        const values: Record<string, string> = {};
        for (const variable of preview.variable_declarations) {
          if (variable.editable)
            values[variable.name] = displayVariableValue(variable.value ?? variable.default);
        }
        runForm.variables = values;
        initialRunVariableValues.value = { ...values };
      }
    } catch (reason) {
      if (request === runPreviewRequest) {
        const message = reason instanceof Error ? reason.message : '解析运行时变量失败';
        runPreviewError.value = setRunFieldErrorFromMessage(message) ? '' : message;
      }
    } finally {
      if (request === runPreviewRequest) runPreviewLoading.value = false;
    }
  }

  function scheduleRunVariablePreview() {
    if (!runOpen.value) return;
    if (runPreviewTimer) clearTimeout(runPreviewTimer);
    runPreviewTimer = setTimeout(() => void loadRunVariablePreview(), 250);
  }
  async function fetchPipeline() {
    pipelineVariablePreviewRequest += 1;
    pipelineVariablePreview.value = undefined;
    pipelineVariablePreviewLoading.value = false;
    pipelineVariablePreviewError.value = '';
    try {
      await execute(async () => {
        pipeline.value = await pipelineApi.get(pipelineId.value);
      });
      if (pipeline.value?.kind === 'application') {
        await loadPipelineVariablePreview();
      }
      await loadStageTemplates();
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载流水线失败');
      await router.push('/pipeline');
    }
  }

  async function loadStageTemplates() {
    if (!pipeline.value?.project_id) {
      stageTemplates.value = [];
      return;
    }
    const response = await pipelineStageApi.list({
      project_id: pipeline.value.project_id,
      per_page: 100,
    });
    stageTemplates.value = response.items;
  }

  function selectStageTemplate(value: ComboboxOptionValue) {
    const stage = stageTemplates.value.find((item) => item.id === String(value));
    if (!stage) return;
    Object.assign(stageForm, {
      name: stage.name,
      description: stage.description,
    });
  }

  async function openInfoDialog() {
    if (!pipeline.value) return;
    Object.assign(infoForm, {
      name: pipeline.value.name,
      description: pipeline.value.description,
      applicationId: pipeline.value.application_id || '',
    });
    Object.assign(infoErrors, { name: '', applicationId: '' });
    infoError.value = '';
    applications.value = [];
    if (!isTemplate.value) {
      const projectId = pipeline.value.project_id || projectStore.activeProjectId || '';
      if (!projectId) {
        toast.error('请先选择项目');
        return;
      }
      try {
        const page = await applicationApi.list({
          project_id: projectId,
          per_page: 100,
        });
        applications.value = page.items ?? [];
      } catch (reason) {
        toast.error(reason instanceof Error ? reason.message : '加载应用列表失败');
        return;
      }
    }
    infoOpen.value = true;
  }

  async function saveInfo() {
    infoErrors.name = infoForm.name.trim() ? '' : '请输入流水线名称';
    infoErrors.applicationId = '';
    infoError.value = '';
    if (infoErrors.name) return;
    const payload: {
      name: string;
      description: string;
      application_id?: string;
    } = {
      name: infoForm.name.trim(),
      description: infoForm.description,
    };
    if (canBindApplication.value && infoForm.applicationId) {
      payload.application_id = infoForm.applicationId;
    }
    try {
      await executeSave(async () => {
        pipeline.value = await pipelineApi.update(pipelineId.value, payload);
        await loadPipelineVariablePreview();
        infoOpen.value = false;
        toast.success('流水线信息已保存');
      });
    } catch (reason) {
      infoError.value = reason instanceof Error ? reason.message : '保存失败';
    }
  }

  function pipelineVariablesWith(nextVariable?: VariableDeclarationResp) {
    const next = pipelineCustomVariables.value.filter(
      (variable) => variable.name !== nextVariable?.name
    );
    return nextVariable ? [...next, nextVariable] : next;
  }

  function openAddVariableDialog() {
    Object.assign(variableForm, { name: '', value: '', description: '', secret: false });
    editingVariableName.value = '';
    variableError.value = '';
    variableOpen.value = true;
  }

  function openEditVariableDialog(name: string) {
    const variable = pipelineCustomVariables.value.find((item) => item.name === name);
    if (!variable) return;
    Object.assign(variableForm, {
      name: variable.name,
      value: displayVariableValue(variable.value ?? variable.default),
      description: variable.description,
      secret: variable.secret,
    });
    editingVariableName.value = name;
    variableError.value = '';
    variableOpen.value = true;
  }

  async function saveVariable() {
    const name = variableForm.name.trim();
    variableError.value = !name
      ? '请输入变量名'
      : !editingVariableName.value &&
          pipelineVariables.value.some((variable) => variable.name === name)
        ? '变量名已存在'
        : '';
    if (variableError.value) return;
    const variable: VariableDeclarationResp = {
      name,
      description: variableForm.description.trim(),
      default: undefined,
      value: variableForm.value,
      secret: variableForm.secret,
      source: 'pipeline_custom',
      editable: true,
    };
    try {
      await executeSave(async () => {
        pipeline.value = await pipelineApi.update(pipelineId.value, {
          variable_declarations: { items: pipelineVariablesWith(variable) },
        });
        await loadPipelineVariablePreview();
        variableOpen.value = false;
        toast.success(editingVariableName.value ? '变量已更新' : '变量已添加');
      });
    } catch (reason) {
      variableError.value = reason instanceof Error ? reason.message : '保存变量失败';
    }
  }

  async function deleteVariable(name: string) {
    try {
      await executeSave(async () => {
        pipeline.value = await pipelineApi.update(pipelineId.value, {
          variable_declarations: {
            items: pipelineCustomVariables.value.filter((variable) => variable.name !== name),
          },
        });
        await loadPipelineVariablePreview();
        toast.success('变量已重置');
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '重置变量失败');
    }
  }

  function openStageDialog(stage?: PipelineStageNodeResp) {
    editingStage.value = stage;
    Object.assign(
      stageForm,
      stage
        ? {
            source_template_stage_id: stage.source_template_stage_id,
            name: stage.name,
            depends_on: [...stage.depends_on],
            sort_order: stage.sort_order,
            description: stage.description,
          }
        : {
            source_template_stage_id: '',
            name: '',
            depends_on: [],
            sort_order: (orderedStages.value.at(-1)?.sort_order || 0) + 1,
            description: '',
          }
    );
    stageError.value = '';
    stageOpen.value = true;
  }

  async function openTemplateUpdate() {
    if (!editingStage.value) return;
    templateUpdateOpen.value = true;
    templateUpdateLoading.value = true;
    templateUpdatePreview.value = undefined;
    try {
      const preview = await pipelineApi.previewStageTemplateUpdate(
        pipelineId.value,
        editingStage.value.id
      );
      if (!preview.available) {
        templateUpdateOpen.value = false;
        toast.error('阶段模板没有可用更新');
        await fetchPipeline();
        return;
      }
      templateUpdatePreview.value = preview;
    } catch (reason) {
      templateUpdateOpen.value = false;
      toast.error(reason instanceof Error ? reason.message : '加载模板更新失败');
    } finally {
      templateUpdateLoading.value = false;
    }
  }

  function templateUpdateFieldLabel(field: string) {
    return (
      {
        name: '来源名称',
        image: '执行镜像',
        script: '脚本',
        description: '模板说明',
        artifacts: '制品声明',
      }[field] || field
    );
  }

  async function applyTemplateUpdate() {
    const preview = templateUpdatePreview.value;
    const stage = editingStage.value;
    if (!preview || !stage) return;
    try {
      await executeSave(async () => {
        pipeline.value = await pipelineApi.updateStageTemplate(pipelineId.value, stage.id, {
          expected_source_template_stage_version: preview.expected_source_template_stage_version,
          target_template_stage_version: preview.target_template_stage_version,
        });
        await loadPipelineVariablePreview();
        templateUpdateOpen.value = false;
        stageOpen.value = false;
        toast.success('阶段已更新至模板版本');
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '更新阶段模板失败');
      await fetchPipeline();
    }
  }

  async function updateOutdatedStages() {
    const stages = [...outdatedStages.value];
    if (stages.length === 0) return;

    let skippedCount = 0;
    updatingStages.value = true;
    updatedStageCount.value = 0;
    stagesToUpdateCount.value = stages.length;
    try {
      await executeSave(async () => {
        for (const stage of stages) {
          const preview = await pipelineApi.previewStageTemplateUpdate(pipelineId.value, stage.id);
          if (!preview.available) {
            skippedCount += 1;
            updatedStageCount.value += 1;
            continue;
          }
          pipeline.value = await pipelineApi.updateStageTemplate(pipelineId.value, stage.id, {
            expected_source_template_stage_version: preview.expected_source_template_stage_version,
            target_template_stage_version: preview.target_template_stage_version,
          });
          updatedStageCount.value += 1;
        }
        await loadPipelineVariablePreview();
      });
      const updatedCount = stages.length - skippedCount;
      toast.success(
        skippedCount > 0
          ? `已更新 ${updatedCount} 个构建阶段，${skippedCount} 个当前不可更新`
          : `已更新 ${updatedCount} 个构建阶段`
      );
    } catch (reason) {
      const completedCount = updatedStageCount.value - skippedCount;
      toast.error(
        `${reason instanceof Error ? reason.message : '更新构建阶段失败'}${
          completedCount > 0 ? `，已更新 ${completedCount} 个构建阶段` : ''
        }`
      );
      await fetchPipeline();
    } finally {
      updatingStages.value = false;
      updatedStageCount.value = 0;
      stagesToUpdateCount.value = 0;
    }
  }

  async function saveStage() {
    stageError.value =
      !editingStage.value && !stageForm.source_template_stage_id
        ? '请选择阶段'
        : !stageForm.name.trim()
          ? '请输入阶段名称'
          : '';
    if (stageError.value) return;
    const payload = {
      name: stageForm.name.trim(),
      depends_on: [...stageForm.depends_on],
      sort_order: stageForm.sort_order,
      description: stageForm.description,
    };
    try {
      await executeSave(async () => {
        if (editingStage.value) {
          pipeline.value = await pipelineApi.updateStage(pipelineId.value, editingStage.value.id, {
            name: payload.name,
            depends_on: { items: payload.depends_on },
            sort_order: payload.sort_order,
            description: payload.description,
          });
        } else
          pipeline.value = await pipelineApi.importStage(pipelineId.value, {
            source_template_stage_id: stageForm.source_template_stage_id,
            name: payload.name,
            description: payload.description,
            depends_on: payload.depends_on,
            sort_order: payload.sort_order,
          });
        await loadPipelineVariablePreview();
        stageOpen.value = false;
        toast.success('阶段已保存');
      });
    } catch (reason) {
      stageError.value = reason instanceof Error ? reason.message : '保存阶段失败';
    }
  }

  async function removeStage(stage: PipelineStageNodeResp) {
    try {
      await executeSave(async () => {
        pipeline.value = await pipelineApi.deleteStage(pipelineId.value, stage.id);
        await loadPipelineVariablePreview();
        toast.success('阶段已删除');
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '删除阶段失败');
    }
  }

  function openRunDialog() {
    runForm.variables = {};
    for (const name of Object.keys(runFieldErrors)) delete runFieldErrors[name];
    runPreview.value = undefined;
    initialRunVariableValues.value = {};
    runError.value = '';
    runPreviewError.value = '';
    runOpen.value = true;
    void loadRunVariablePreview(true);
  }

  async function runPipeline() {
    for (const name of Object.keys(runFieldErrors)) delete runFieldErrors[name];
    const missingVariable = runVariableDeclarations.value.find((variable) => {
      if (!variable.editable || runForm.variables[variable.name]?.trim()) return false;
      runFieldErrors[variable.name] =
        variable.name === 'repository_ref' ? '请输入分支或标签' : `请输入变量值: ${variable.name}`;
      return true;
    });
    runError.value =
      !runPreview.value || runPreviewLoading.value
        ? '正在解析运行时变量'
        : runPreviewError.value
          ? '运行时变量解析失败'
          : missingVariable
            ? '请完善必填变量'
            : '';
    if (runError.value) return;
    try {
      await executeSave(async () => {
        const run = await pipelineRunApi.trigger(pipelineId.value, {
          variables: buildRunVariableForm(),
        });
        runOpen.value = false;
        toast.success('流水线已触发');
        await router.push(`/pipeline-run/${run.id}`);
      });
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : '触发流水线失败';
      runError.value = setRunFieldErrorFromMessage(message) ? '请完善必填变量' : message;
    }
  }

  function goToInstantiation() {
    router.push({ path: '/pipeline', query: { instantiate: pipelineId.value } });
  }

  function resetRunDialog() {
    runPreviewRequest += 1;
    if (runPreviewTimer) clearTimeout(runPreviewTimer);
    runPreviewTimer = undefined;
    runPreview.value = undefined;
    runPreviewLoading.value = false;
    runPreviewError.value = '';
    runError.value = '';
    initialRunVariableValues.value = {};
    runForm.variables = {};
    for (const name of Object.keys(runFieldErrors)) delete runFieldErrors[name];
  }

  async function deletePipeline() {
    deleteError.value = '';
    try {
      await executeSave(async () => {
        await pipelineApi.delete(pipelineId.value);
        toast.success('流水线已删除');
        await router.push('/pipeline');
      });
    } catch (reason) {
      deleteError.value = reason instanceof Error ? reason.message : '删除流水线失败';
    }
  }

  watch(
    () => runForm.variables,
    () => {
      if (Object.keys(initialRunVariableValues.value).length > 0) scheduleRunVariablePreview();
    },
    { deep: true }
  );
  watch(runOpen, (open) => {
    if (open) return;
    resetRunDialog();
  });
  watch(pipelineId, () => {
    runOpen.value = false;
    resetRunDialog();
    void fetchPipeline();
  });
  onMounted(fetchPipeline);
  onBeforeUnmount(() => {
    resetRunDialog();
  });
</script>
