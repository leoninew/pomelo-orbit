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
          <button class="app-button h-9 px-3" @click="openInfoDialog">
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
            添加阶段
          </button>
        </div>
        <AppEmptyState v-if="pipeline.stages.length === 0" size="compact" />
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
      :title="editingStage ? '编辑构建阶段' : '添加构建阶段'"
      width-class="w-[min(760px,calc(100vw-32px))]"
      body-class="max-h-[72vh] space-y-4 overflow-y-auto px-6 py-4"
    >
      <form class="space-y-4" @submit.prevent="saveStage">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label">
              名称
              <span class="text-destructive">*</span>
            </label>
            <input v-model="stageForm.name" class="app-input" />
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label">
              执行镜像
              <span class="text-destructive">*</span>
            </label>
            <input v-model="stageForm.image" class="app-input" />
          </div>
        </div>
        <div class="space-y-1.5">
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
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">制品声明</label>
            <button type="button" class="app-button h-8 px-3" @click="openArtifactDialog()">
              <Plus class="size-4" />
              添加制品
            </button>
          </div>
          <AppEmptyState v-if="stageForm.artifacts.length === 0" size="compact" />
          <div v-else class="overflow-x-auto">
            <table class="app-data-table min-w-[640px]">
              <thead>
                <tr>
                  <th>名称</th>
                  <th>收集器</th>
                  <th v-if="hasApplicationBinding">组件</th>
                  <th class="w-28">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(artifact, index) in stageForm.artifacts"
                  :key="`${artifact.name}-${index}`"
                >
                  <td>{{ artifact.name }}</td>
                  <td>{{ artifact.collector }}</td>
                  <td v-if="hasApplicationBinding">{{ artifact.component_name || '不绑定' }}</td>
                  <td>
                    <button type="button" class="app-link" @click="openArtifactDialog(index)">
                      编辑
                    </button>
                    <button
                      type="button"
                      class="app-link-danger ml-3"
                      @click="stageForm.artifacts.splice(index, 1)"
                    >
                      删除
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <p v-if="stageError" class="app-field-error" role="alert">{{ stageError }}</p>
      </form>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="stageOpen = false" @confirm="saveStage" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="artifactOpen" :title="artifactIndex === -1 ? '添加制品' : '编辑制品'">
      <form class="space-y-4" @submit.prevent="saveArtifact">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label">
              名称
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="artifactForm.name"
              class="app-input"
              :class="artifactErrors.name ? 'app-input-error' : ''"
              :aria-invalid="artifactErrors.name ? 'true' : undefined"
              @input="clearArtifactError('name')"
            />
            <p v-if="artifactErrors.name" class="app-field-error" role="alert">
              {{ artifactErrors.name }}
            </p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label">收集器</label>
            <select
              v-model="artifactForm.collector"
              class="app-input"
              @change="changeArtifactCollector"
            >
              <option value="docker_image">docker_image</option>
              <option value="command">command</option>
              <option value="file">file</option>
              <option value="directory">directory</option>
            </select>
          </div>
        </div>
        <div v-if="artifactForm.collector === 'command'" class="space-y-1.5">
          <label class="app-field-label">
            命令
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="artifactForm.command"
            class="app-input"
            :class="artifactErrors.command ? 'app-input-error' : ''"
            :aria-invalid="artifactErrors.command ? 'true' : undefined"
            @input="clearArtifactError('command')"
          />
          <p v-if="artifactErrors.command" class="app-field-error" role="alert">
            {{ artifactErrors.command }}
          </p>
          <label class="app-field-label">格式</label>
          <select
            v-model="artifactForm.format"
            class="app-input"
            :class="artifactErrors.format ? 'app-input-error' : ''"
            :aria-invalid="artifactErrors.format ? 'true' : undefined"
            @change="clearArtifactError('format')"
          >
            <option value="">请选择格式</option>
            <option value="text">text</option>
            <option value="json">json</option>
            <option value="git_object_id">git_object_id</option>
          </select>
          <p v-if="artifactErrors.format" class="app-field-error" role="alert">
            {{ artifactErrors.format }}
          </p>
        </div>
        <div v-else class="space-y-1.5">
          <label class="app-field-label">
            引用
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="artifactForm.reference"
            class="app-input"
            :class="artifactErrors.reference ? 'app-input-error' : ''"
            :aria-invalid="artifactErrors.reference ? 'true' : undefined"
            @input="clearArtifactError('reference')"
          />
          <p v-if="artifactErrors.reference" class="app-field-error" role="alert">
            {{ artifactErrors.reference }}
          </p>
        </div>
        <div
          v-if="hasApplicationBinding && artifactForm.collector === 'docker_image'"
          class="space-y-1.5"
        >
          <fieldset class="space-y-2">
            <legend class="app-field-label">来源版本策略</legend>
            <RadioGroupRoot
              v-model="artifactBindingForm.version_fork_strategy"
              aria-label="来源版本策略"
              class="flex flex-wrap items-center gap-4"
            >
              <div class="flex items-center gap-2 text-sm">
                <RadioGroupItem
                  id="artifact-version-strategy-latest"
                  value="latest"
                  class="flex size-4 shrink-0 items-center justify-center rounded-full border border-input bg-background text-primary outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 data-[state=checked]:border-primary"
                >
                  <RadioGroupIndicator class="size-2 rounded-full bg-current" />
                </RadioGroupItem>
                <label for="artifact-version-strategy-latest">最新版本</label>
              </div>
              <div class="flex items-center gap-2 text-sm">
                <RadioGroupItem
                  id="artifact-version-strategy-fixed"
                  value="fixed"
                  class="flex size-4 shrink-0 items-center justify-center rounded-full border border-input bg-background text-primary outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 data-[state=checked]:border-primary"
                >
                  <RadioGroupIndicator class="size-2 rounded-full bg-current" />
                </RadioGroupItem>
                <label for="artifact-version-strategy-fixed">固定版本</label>
              </div>
            </RadioGroupRoot>
          </fieldset>
          <p class="app-field-hint">此策略适用于流水线内的全部目标组件映射。</p>
          <label class="app-field-label">目标组件</label>
          <ComboboxSelect
            v-model="artifactForm.component_name"
            :options="componentOptions"
            :disabled="sourceVersionLoading || (!sourceVersionId && !artifactForm.component_name)"
            :empty-text="componentEmptyText"
            :invalid="Boolean(artifactErrors.component_name || componentMappingError)"
            :placeholder="sourceVersionLoading ? '正在加载目标组件' : '不绑定到应用版本'"
            @update:model-value="clearArtifactError('component_name')"
          />
          <p v-if="artifactErrors.component_name" class="app-field-error" role="alert">
            {{ artifactErrors.component_name }}
          </p>
          <p v-else-if="componentMappingError" class="app-field-error" role="alert">
            {{ componentMappingError }}
          </p>
          <p v-else-if="sourceVersionError" class="app-field-error" role="alert">
            {{ sourceVersionError }}
          </p>
          <p v-else-if="!sourceVersionId" class="app-field-hint">
            {{
              artifactBindingForm.version_fork_strategy === 'fixed'
                ? '选择来源版本后加载目标组件。'
                : '应用尚未创建版本，无法选择目标组件。'
            }}
          </p>
          <p
            v-else-if="!sourceVersionLoading && sourceVersionDetail?.components?.length === 0"
            class="app-field-hint"
          >
            来源版本 {{ sourceVersionLabel }} 暂无组件。
          </p>
          <div v-if="artifactBindingForm.version_fork_strategy === 'fixed'" class="space-y-1.5">
            <label class="app-field-label">来源版本</label>
            <ComboboxSelect
              v-model="artifactBindingForm.fixed_version_id"
              :options="versionOptions"
              :invalid="Boolean(artifactErrors.fixed_version_id)"
              placeholder="选择来源版本"
              @update:model-value="clearArtifactError('fixed_version_id')"
            />
            <p v-if="artifactErrors.fixed_version_id" class="app-field-error" role="alert">
              {{ artifactErrors.fixed_version_id }}
            </p>
          </div>
        </div>
      </form>
      <template #footer>
        <AppDialogActions @cancel="closeArtifactDialog" @confirm="saveArtifact" />
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
  import { applicationApi } from '@/api/application/application';
  import { pipelineApi } from '@/api/pipeline/pipeline';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VariableDeclarationResp } from '@/gen/proto/orbit/v1/common/common';
  import type {
    ArtifactConfigReq,
    PipelineStageResp,
  } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import type { PipelineResp } from '@/gen/proto/orbit/v1/pipeline/pipeline';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { PipelineRunVariablePreviewResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import { RadioGroupIndicator, RadioGroupItem, RadioGroupRoot } from 'reka-ui';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const { status, execute } = useStatusAsync();
  const { loading: saving, execute: executeSave } = useStatusAsync();
  const pipelineId = computed(() => String(route.params.id));
  const pipeline = ref<PipelineResp>();
  const versions = ref<VersionResp[]>([]);
  const infoOpen = ref(false);
  const stageOpen = ref(false);
  const artifactOpen = ref(false);
  const runOpen = ref(false);
  const variableOpen = ref(false);
  const deleteOpen = ref(false);
  const editingStage = ref<PipelineStageResp>();
  const artifactIndex = ref(-1);
  const infoError = ref('');
  const stageError = ref('');
  const sourceVersionDetail = ref<VersionResp>();
  const sourceVersionLoading = ref(false);
  const sourceVersionError = ref('');
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
    name: '',
    image: '',
    script: '',
    artifacts: [] as ArtifactConfigReq[],
    depends_on: [] as string[],
    sort_order: 0,
    description: '',
    version_fork_strategy: 'latest',
    fixed_version_id: '',
  });
  const artifactForm = reactive<ArtifactConfigReq>({
    name: '',
    collector: 'docker_image',
    reference: '',
    command: '',
    format: '',
    component_name: undefined,
  });
  const artifactBindingForm = reactive({ version_fork_strategy: 'latest', fixed_version_id: '' });
  type ArtifactFieldError =
    'name' | 'command' | 'format' | 'reference' | 'fixed_version_id' | 'component_name';
  const artifactErrors = reactive<Record<ArtifactFieldError, string>>({
    name: '',
    command: '',
    format: '',
    reference: '',
    fixed_version_id: '',
    component_name: '',
  });
  const runForm = reactive({ trigger_ref: '', variables: {} as Record<string, string> });
  const variableForm = reactive({ name: '', value: '', description: '', secret: false });
  const editingVariableName = ref('');

  const isTemplate = computed(() => pipeline.value?.kind === 'template');
  const hasApplicationBinding = computed(() =>
    Boolean(pipeline.value?.application_id && pipeline.value.application_name)
  );
  const orderedStages = computed(() =>
    [...(pipeline.value?.stages || [])].sort((left, right) => left.sort_order - right.sort_order)
  );
  const otherStages = computed(() =>
    orderedStages.value.filter((stage) => stage.id !== editingStage.value?.id)
  );
  const hasComponentMapping = computed(() =>
    stageForm.artifacts.some(
      (artifact) => artifact.collector === 'docker_image' && Boolean(artifact.component_name)
    )
  );
  const versionOptions = computed(() =>
    versions.value.map((version) => ({
      value: version.id,
      label: version.label,
      description: version.component_summary,
    }))
  );
  const sourceVersionId = computed(() =>
    artifactBindingForm.version_fork_strategy === 'fixed'
      ? artifactBindingForm.fixed_version_id
      : (versions.value[0]?.id ?? '')
  );
  const sourceVersionLabel = computed(
    () => versions.value.find((version) => version.id === sourceVersionId.value)?.label || ''
  );
  const componentOptions = computed(() => {
    const options = (sourceVersionDetail.value?.components || []).map((component) => ({
      value: component.name,
      label: component.name,
      description: component.image,
    }));
    const mappedComponent = artifactForm.component_name;
    if (mappedComponent && !options.some((option) => option.value === mappedComponent))
      options.unshift({ value: mappedComponent, label: mappedComponent, description: '当前映射' });
    return options;
  });
  const componentEmptyText = computed(() => {
    if (sourceVersionLoading.value) return '正在加载目标组件';
    if (sourceVersionError.value) return '目标组件加载失败';
    if (!sourceVersionId.value)
      return artifactBindingForm.version_fork_strategy === 'fixed'
        ? '请先选择来源版本'
        : '应用暂无版本';
    return '该来源版本暂无组件';
  });
  const componentMappingError = computed(() => {
    const mappedComponent = artifactForm.component_name;
    if (!mappedComponent || !sourceVersionDetail.value || sourceVersionLoading.value) return '';
    if (sourceVersionDetail.value.components.some((item) => item.name === mappedComponent))
      return '';
    return `来源版本 ${sourceVersionLabel.value || sourceVersionId.value} 不包含目标组件 ${mappedComponent}`;
  });
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
    return pipeline.value?.stages.find((stage) => stage.id === id)?.name || id;
  }
  function mappedArtifacts(stage: PipelineStageResp) {
    return stage.artifacts.filter((artifact) => artifact.component_name);
  }
  function cloneArtifacts(artifacts: ArtifactConfigReq[]) {
    return artifacts.map((artifact) => ({ ...artifact }));
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
  function clearArtifactError(field: ArtifactFieldError) {
    artifactErrors[field] = '';
  }
  function clearArtifactErrors() {
    for (const field of Object.keys(artifactErrors) as ArtifactFieldError[])
      artifactErrors[field] = '';
  }
  function clearSourceVersion() {
    sourceVersionRequest += 1;
    sourceVersionDetail.value = undefined;
    sourceVersionLoading.value = false;
    sourceVersionError.value = '';
  }

  let sourceVersionRequest = 0;
  async function loadSourceVersion() {
    const request = ++sourceVersionRequest;
    const versionId = sourceVersionId.value;
    sourceVersionDetail.value = undefined;
    sourceVersionError.value = '';
    if (
      !artifactOpen.value ||
      !hasApplicationBinding.value ||
      artifactForm.collector !== 'docker_image' ||
      !versionId
    ) {
      sourceVersionLoading.value = false;
      return;
    }

    sourceVersionLoading.value = true;
    try {
      const version = await applicationApi.getVersion(versionId);
      if (request === sourceVersionRequest) sourceVersionDetail.value = version;
    } catch (reason) {
      if (request === sourceVersionRequest)
        sourceVersionError.value =
          reason instanceof Error ? reason.message : '加载来源版本的目标组件失败';
    } finally {
      if (request === sourceVersionRequest) sourceVersionLoading.value = false;
    }
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
        if (pipeline.value.application_id) await loadVersions();
      }
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载流水线失败');
      await router.push('/pipeline');
    }
  }

  async function loadVersions() {
    if (!pipeline.value?.application_id) return;
    const response = await applicationApi.listVersions(pipeline.value.application_id, {
      per_page: 100,
    });
    versions.value = response.items;
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

  function openStageDialog(stage?: PipelineStageResp) {
    editingStage.value = stage;
    Object.assign(
      stageForm,
      stage
        ? {
            name: stage.name,
            image: stage.image,
            script: stage.script,
            artifacts: cloneArtifacts(stage.artifacts),
            depends_on: [...stage.depends_on],
            sort_order: stage.sort_order,
            description: stage.description,
            version_fork_strategy: pipeline.value?.version_fork_strategy || 'latest',
            fixed_version_id: pipeline.value?.fixed_version_id || '',
          }
        : {
            name: '',
            image: '',
            script: '',
            artifacts: [],
            depends_on: [],
            sort_order: (orderedStages.value.at(-1)?.sort_order || 0) + 1,
            description: '',
            version_fork_strategy: 'latest',
            fixed_version_id: '',
          }
    );
    stageError.value = '';
    stageOpen.value = true;
  }

  function openArtifactDialog(index = -1) {
    artifactIndex.value = index;
    Object.assign(
      artifactForm,
      index === -1
        ? {
            name: '',
            collector: 'docker_image',
            reference: '',
            command: '',
            format: '',
            component_name: undefined,
          }
        : { ...stageForm.artifacts[index] }
    );
    Object.assign(artifactBindingForm, {
      version_fork_strategy: stageForm.version_fork_strategy,
      fixed_version_id: stageForm.fixed_version_id,
    });
    clearArtifactErrors();
    artifactOpen.value = true;
  }

  function closeArtifactDialog() {
    artifactOpen.value = false;
  }

  function changeArtifactCollector() {
    clearArtifactErrors();
    if (artifactForm.collector !== 'docker_image') artifactForm.component_name = undefined;
  }

  function saveArtifact() {
    clearArtifactErrors();
    if (!artifactForm.name.trim()) artifactErrors.name = '请输入制品名称';
    if (artifactForm.collector === 'command' && !artifactForm.command.trim())
      artifactErrors.command = '请输入制品命令';
    if (artifactForm.collector === 'command' && !artifactForm.format)
      artifactErrors.format = '请选择命令输出格式';
    if (artifactForm.collector !== 'command' && !artifactForm.reference.trim())
      artifactErrors.reference = '请输入制品引用';
    if (
      hasApplicationBinding.value &&
      artifactForm.collector === 'docker_image' &&
      artifactForm.component_name &&
      artifactBindingForm.version_fork_strategy === 'fixed' &&
      !artifactBindingForm.fixed_version_id
    )
      artifactErrors.fixed_version_id = '请选择来源版本';
    if (
      hasApplicationBinding.value &&
      artifactForm.collector === 'docker_image' &&
      artifactForm.component_name
    ) {
      if (!sourceVersionId.value) artifactErrors.component_name = '请先选择来源版本';
      else if (sourceVersionLoading.value) artifactErrors.component_name = '正在加载来源版本组件';
      else if (sourceVersionError.value) artifactErrors.component_name = '无法验证目标组件';
      else if (!sourceVersionDetail.value) artifactErrors.component_name = '正在加载来源版本组件';
      else if (componentMappingError.value)
        artifactErrors.component_name = componentMappingError.value;
    }
    if (Object.values(artifactErrors).some(Boolean)) return;
    const artifact: ArtifactConfigReq = {
      ...artifactForm,
      name: artifactForm.name.trim(),
      reference: artifactForm.collector === 'command' ? '' : artifactForm.reference.trim(),
      command: artifactForm.collector === 'command' ? artifactForm.command.trim() : '',
      format: artifactForm.collector === 'command' ? artifactForm.format : '',
      component_name:
        hasApplicationBinding.value &&
        artifactForm.collector === 'docker_image' &&
        artifactForm.component_name
          ? artifactForm.component_name
          : undefined,
    };
    const nextArtifacts =
      artifactIndex.value === -1
        ? [...stageForm.artifacts, artifact]
        : stageForm.artifacts.map((item, index) =>
            index === artifactIndex.value ? artifact : item
          );
    const hasNextComponentMapping = nextArtifacts.some(
      (item) => item.collector === 'docker_image' && Boolean(item.component_name)
    );
    if (
      hasApplicationBinding.value &&
      artifact.collector === 'docker_image' &&
      hasNextComponentMapping
    )
      Object.assign(stageForm, artifactBindingForm);
    if (artifactIndex.value === -1) stageForm.artifacts.push(artifact);
    else stageForm.artifacts[artifactIndex.value] = artifact;
    closeArtifactDialog();
  }

  function remainingMappings() {
    return (
      (pipeline.value?.stages || [])
        .filter((stage) => stage.id !== editingStage.value?.id)
        .flatMap((stage) => stage.artifacts)
        .some((artifact) => artifact.collector === 'docker_image' && artifact.component_name) ||
      hasComponentMapping.value
    );
  }

  async function saveStage() {
    stageError.value = !stageForm.name.trim()
      ? '请输入阶段名称'
      : !stageForm.image.trim()
        ? '请输入执行镜像'
        : hasComponentMapping.value &&
            stageForm.version_fork_strategy === 'fixed' &&
            !stageForm.fixed_version_id
          ? '请选择固定来源版本'
          : '';
    if (stageError.value) return;
    const payload = {
      name: stageForm.name.trim(),
      image: stageForm.image.trim(),
      script: stageForm.script,
      artifacts: cloneArtifacts(stageForm.artifacts),
      depends_on: [...stageForm.depends_on],
      sort_order: stageForm.sort_order,
      description: stageForm.description,
    };
    try {
      await executeSave(async () => {
        if (editingStage.value) {
          const mapped = remainingMappings();
          pipeline.value = await pipelineApi.updateStage(pipelineId.value, editingStage.value.id, {
            name: payload.name,
            image: payload.image,
            script: payload.script,
            artifacts: { items: payload.artifacts },
            depends_on: { items: payload.depends_on },
            sort_order: payload.sort_order,
            description: payload.description,
            version_fork_strategy: mapped ? stageForm.version_fork_strategy : undefined,
            fixed_version_id:
              mapped && stageForm.version_fork_strategy === 'fixed'
                ? stageForm.fixed_version_id
                : mapped
                  ? ''
                  : undefined,
            clear_version_fork_strategy:
              !mapped && Boolean(pipeline.value?.version_fork_strategy) ? true : undefined,
          });
        } else
          pipeline.value = await pipelineApi.createStage(pipelineId.value, {
            ...payload,
            version_fork_strategy: hasComponentMapping.value
              ? stageForm.version_fork_strategy
              : undefined,
            fixed_version_id:
              hasComponentMapping.value && stageForm.version_fork_strategy === 'fixed'
                ? stageForm.fixed_version_id
                : undefined,
          });
        await loadPipelineVariablePreview();
        stageOpen.value = false;
        toast.success('阶段已保存');
      });
    } catch (reason) {
      stageError.value = reason instanceof Error ? reason.message : '保存阶段失败';
    }
  }

  async function removeStage(stage: PipelineStageResp) {
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
    () => artifactBindingForm.version_fork_strategy,
    () => {
      clearArtifactError('fixed_version_id');
      if (artifactBindingForm.version_fork_strategy === 'latest')
        artifactBindingForm.fixed_version_id = '';
    }
  );
  watch(artifactOpen, (open) => {
    if (!open) {
      clearArtifactErrors();
      clearSourceVersion();
    }
  });
  watch(
    [() => artifactOpen.value, () => artifactForm.collector, () => sourceVersionId.value],
    () => {
      void loadSourceVersion();
    }
  );
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
