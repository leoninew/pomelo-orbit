<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <h1 class="app-detail-page-title break-words">{{ pipeline?.name || '流水线详情' }}</h1>
        <AppBadge v-if="pipeline && isTemplate" tone="info" variant="status">模板</AppBadge>
      </div>
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
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">基本信息</h2>
          <button class="app-button-primary h-9 px-3" @click="openInfoDialog">
            <Pencil class="size-4" />
            编辑
          </button>
        </div>
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
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">构建阶段</h2>
          <button class="app-button-primary h-9 px-3" @click="openStageDialog()">
            <Plus class="size-4" />
            引入阶段
          </button>
        </div>
        <AppEmptyState v-if="pipeline.stage_nodes.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[820px]">
            <thead>
              <tr>
                <th>#</th>
                <th>名称</th>
                <th>镜像</th>
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
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">变量声明</h2>
          <button class="app-button-primary h-9 px-3" @click="openAddVariableDialog">
            <Plus class="size-4" />
            添加自定义变量
          </button>
        </div>
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
      </div>
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
            :class="infoError ? 'app-input-error' : ''"
          />
          <p v-if="infoError" class="app-field-error">{{ infoError }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">说明</label>
          <textarea v-model="infoForm.description" rows="3" class="app-textarea" />
        </div>
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
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label">
              名称
              <span class="text-destructive">*</span>
            </label>
            <input v-model="stageForm.name" class="app-input" />
          </div>
          <div v-if="!isTemplate" class="space-y-1.5">
            <label class="app-field-label">
              执行镜像
              <span class="text-destructive">*</span>
            </label>
            <input v-model="stageForm.image" class="app-input" />
          </div>
        </div>
        <div v-if="!isTemplate" class="space-y-1.5">
          <label class="app-field-label">脚本</label>
          <textarea v-model="stageForm.script" rows="7" class="app-textarea font-mono" />
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
        <div class="space-y-1.5">
          <label class="app-field-label">
            分支或标签
            <span class="text-destructive">*</span>
          </label>
          <input v-model="runForm.trigger_ref" class="app-input" />
        </div>
        <div v-for="variable in runVariableDeclarations" :key="variable.name" class="space-y-1.5">
          <label class="app-field-label">{{ variable.name }}</label>
          <input
            v-if="variable.editable"
            v-model="runForm.variables[variable.name]"
            :type="variable.secret ? 'password' : 'text'"
            class="app-input"
          />
          <input
            v-else
            :value="displayVariableValue(variable.value ?? variable.default)"
            type="text"
            class="app-input"
            disabled
          />
        </div>
        <p v-if="runPreviewError" class="app-field-error">{{ runPreviewError }}</p>
        <p v-if="runError" class="app-field-error">{{ runError }}</p>
      </form>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          :confirm-disabled="runPreviewLoading || Boolean(runPreviewError)"
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
  import { ArrowLeft, CopyPlus, Pencil, Play, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { pipelineApi } from '@/api/pipeline/pipeline';
  import { pipelineStageApi } from '@/api/pipeline/pipeline_stage';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VariableDeclarationResp } from '@/gen/proto/orbit/v1/common/common';
  import type {
    PipelineStageNodeResp,
    PipelineStageTemplateUpdatePreviewResp,
  } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import type { PipelineResp } from '@/gen/proto/orbit/v1/pipeline/pipeline';
  import type { PipelineRunVariablePreviewResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const { status, execute } = useStatusAsync();
  const { loading: saving, execute: executeSave } = useStatusAsync();
  const pipelineId = computed(() => String(route.params.id));
  const pipeline = ref<PipelineResp>();
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
  const stageError = ref('');
  const runError = ref('');
  const runPreview = ref<PipelineRunVariablePreviewResp>();
  const runPreviewLoading = ref(false);
  const runPreviewError = ref('');
  const pipelineVariablePreview = ref<PipelineRunVariablePreviewResp>();
  const pipelineVariablePreviewLoading = ref(false);
  const pipelineVariablePreviewError = ref('');
  const initialRunVariableValues = ref<Record<string, string>>({});
  const variableError = ref('');
  const deleteError = ref('');
  const infoForm = reactive({ name: '', description: '' });
  const stageForm = reactive({
    source_template_stage_id: '',
    name: '',
    image: '',
    script: '',
    depends_on: [] as string[],
    sort_order: 0,
    description: '',
  });
  const runForm = reactive({ trigger_ref: '', variables: {} as Record<string, string> });
  const variableForm = reactive({ name: '', value: '', description: '', secret: false });
  const editingVariableName = ref('');

  const isTemplate = computed(() => pipeline.value?.kind === 'template');
  const hasApplicationBinding = computed(() =>
    Boolean(pipeline.value?.application_id && pipeline.value.application_name)
  );
  const orderedStages = computed(() =>
    [...(pipeline.value?.stage_nodes || [])].sort(
      (left, right) => left.sort_order - right.sort_order
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
  const runVariableDeclarations = computed(
    () => runPreview.value?.variable_declarations || pipeline.value?.variable_declarations || []
  );
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

  function buildRunVariableOverrides() {
    const variables: Record<string, string> = {};
    for (const variable of runVariableDeclarations.value) {
      if (!variable.editable) continue;
      const current = runForm.variables[variable.name] || '';
      const initial = initialRunVariableValues.value[variable.name] || '';
      if (current !== initial) variables[variable.name] = current;
    }
    return variables;
  }

  let runPreviewRequest = 0;
  let runPreviewTimer: ReturnType<typeof setTimeout> | undefined;
  let pipelineVariablePreviewRequest = 0;

  function previewPipelineVariables(triggerRef: string, variables: Record<string, string>) {
    return pipelineRunApi.previewVariables(pipelineId.value, {
      trigger_ref: triggerRef,
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
      const preview = await previewPipelineVariables('', {});
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
      const preview = await previewPipelineVariables(
        runForm.trigger_ref.trim(),
        buildRunVariableOverrides()
      );
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
      if (!runForm.trigger_ref.trim()) runForm.trigger_ref = preview.trigger_ref;
    } catch (reason) {
      if (request === runPreviewRequest)
        runPreviewError.value = reason instanceof Error ? reason.message : '解析运行时变量失败';
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
      image: stage.image,
      script: stage.script,
      description: stage.description,
    });
  }

  function openInfoDialog() {
    if (!pipeline.value) return;
    Object.assign(infoForm, { name: pipeline.value.name, description: pipeline.value.description });
    infoError.value = '';
    infoOpen.value = true;
  }

  async function saveInfo() {
    infoError.value = infoForm.name.trim() ? '' : '请输入流水线名称';
    if (infoError.value) return;
    try {
      await executeSave(async () => {
        pipeline.value = await pipelineApi.update(pipelineId.value, {
          name: infoForm.name.trim(),
          description: infoForm.description,
        });
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
            image: stage.image,
            script: stage.script,
            depends_on: [...stage.depends_on],
            sort_order: stage.sort_order,
            description: stage.description,
          }
        : {
            source_template_stage_id: '',
            name: '',
            image: '',
            script: '',
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
      }[field] ||
      field
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

  async function saveStage() {
    stageError.value =
      !editingStage.value && !stageForm.source_template_stage_id
        ? '请选择阶段'
        : !stageForm.name.trim()
          ? '请输入阶段名称'
          : !isTemplate.value && !stageForm.image.trim()
            ? '请输入执行镜像'
            : '';
    if (stageError.value) return;
    const payload = {
      name: stageForm.name.trim(),
      image: stageForm.image.trim(),
      script: stageForm.script,
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
            image: isTemplate.value ? undefined : payload.image,
            script: isTemplate.value ? undefined : payload.script,
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
    runForm.trigger_ref = '';
    runForm.variables = {};
    runPreview.value = undefined;
    initialRunVariableValues.value = {};
    runError.value = '';
    runPreviewError.value = '';
    runOpen.value = true;
    void loadRunVariablePreview(true);
  }

  async function runPipeline() {
    const missingVariable = runVariableDeclarations.value.find(
      (variable) => variable.editable && !runForm.variables[variable.name]?.trim()
    );
    runError.value = runPreviewLoading.value
      ? '正在解析运行时变量'
      : runPreviewError.value
        ? '运行时变量解析失败'
        : !runForm.trigger_ref.trim()
          ? '请输入分支或标签'
          : missingVariable
            ? `请输入变量值: ${missingVariable.name}`
            : '';
    if (runError.value) return;
    try {
      await executeSave(async () => {
        const run = await pipelineRunApi.trigger(pipelineId.value, {
          trigger_ref: runForm.trigger_ref.trim(),
          variables: buildRunVariableOverrides(),
        });
        runOpen.value = false;
        toast.success('流水线已触发');
        await router.push(`/pipeline-run/${run.id}`);
      });
    } catch (reason) {
      runError.value = reason instanceof Error ? reason.message : '触发流水线失败';
    }
  }

  function goToInstantiation() {
    router.push({ path: '/pipeline', query: { instantiate: pipelineId.value } });
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
    () => runForm.trigger_ref,
    () => scheduleRunVariablePreview()
  );
  watch(
    () => runForm.variables,
    () => {
      if (Object.keys(initialRunVariableValues.value).length > 0) scheduleRunVariablePreview();
    },
    { deep: true }
  );
  watch(runOpen, (open) => {
    if (open) return;
    runPreviewRequest += 1;
    if (runPreviewTimer) clearTimeout(runPreviewTimer);
    runPreviewTimer = undefined;
  });
  watch(pipelineId, fetchPipeline);
  onMounted(fetchPipeline);
  onBeforeUnmount(() => {
    if (runPreviewTimer) clearTimeout(runPreviewTimer);
  });
</script>
