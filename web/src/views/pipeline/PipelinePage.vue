<template>
  <div class="space-y-6">
    <div class="app-toolbar-simple">
      <SearchControl
        v-model="search"
        placeholder="搜索流水线"
        :loading="status === 'loading'"
        @search="searchPipelines"
      />
      <div class="flex flex-wrap items-center gap-3">
        <SelectControl
          :model-value="kind"
          :options="pipelineKindOptions"
          width-class="w-36"
          @update:model-value="updatePipelineKind"
        />
        <button class="app-button-primary h-10 px-4" @click="openCreateDialog">
          <Plus class="size-4" />
          新建模板
        </button>
      </div>
    </div>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <p v-else-if="status === 'error'" class="py-16 text-center text-sm text-destructive">
        {{ error || '加载流水线失败' }}
      </p>
      <AppEmptyState v-else-if="pipelines.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1000px]">
          <thead>
            <tr>
              <th>名称</th>
              <th>类型</th>
              <th>代码仓库</th>
              <th>应用</th>
              <th>版本</th>
              <th>更新时间</th>
              <th class="w-52">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="pipeline in pipelines" :key="pipeline.id">
              <td class="max-w-64 truncate">
                <router-link :to="`/pipeline/${pipeline.id}`" class="app-link">
                  {{ pipeline.name }}
                </router-link>
              </td>
              <td>
                <AppBadge
                  :tone="pipeline.kind === 'template' ? 'info' : 'success'"
                  variant="status"
                >
                  {{ pipeline.kind === 'template' ? '模板' : '应用流水线' }}
                </AppBadge>
              </td>
              <td class="text-foreground">
                <template v-if="pipeline.kind === 'application' && pipeline.repository_id">
                  <router-link :to="`/repository/${pipeline.repository_id}`" class="app-link">
                    {{ pipeline.repository_name }}
                  </router-link>
                </template>
                <span v-else class="text-muted-foreground">未绑定</span>
              </td>
              <td class="text-foreground">
                <template
                  v-if="
                    pipeline.kind === 'application' &&
                    pipeline.application_id &&
                    pipeline.application_name
                  "
                >
                  <router-link :to="`/application/${pipeline.application_id}`" class="app-link">
                    {{ pipeline.application_name }}
                  </router-link>
                </template>
                <span v-else class="text-muted-foreground">未绑定</span>
              </td>
              <td>
                <AppBadge>v{{ pipeline.version }}</AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(pipeline.updated_at) }}
              </td>
              <td>
                <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
                  <button class="app-link" @click="openEditDialog(pipeline)">编辑</button>
                  <button
                    v-if="pipeline.kind === 'template'"
                    class="app-link"
                    @click="openInstantiateDialog(pipeline)"
                  >
                    复用
                  </button>
                  <button class="app-link-danger" @click="openDeleteDialog(pipeline)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <ListPagination
        :current="pagination.current"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="changePageSize"
      />
    </div>
  </div>

  <AppDialog v-model:open="createOpen" title="新建流水线模板">
    <form class="space-y-4" @submit.prevent="createTemplate">
      <div class="space-y-1.5">
        <label class="app-field-label">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="createForm.name"
          class="app-input"
          :class="createError ? 'app-input-error' : ''"
        />
        <p v-if="createError" class="app-field-error" role="alert">{{ createError }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label">说明</label>
        <input v-model="createForm.description" class="app-input" />
      </div>
    </form>
    <template #footer>
      <AppDialogActions :busy="operating" @cancel="createOpen = false" @confirm="createTemplate" />
    </template>
  </AppDialog>

  <AppDialog v-model:open="editOpen" title="编辑流水线信息">
    <form class="space-y-4" @submit.prevent="savePipelineInfo">
      <div class="space-y-1.5">
        <label class="app-field-label">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="editForm.name"
          class="app-input"
          :class="editError ? 'app-input-error' : ''"
        />
        <p v-if="editError" class="app-field-error" role="alert">{{ editError }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label">说明</label>
        <input v-model="editForm.description" class="app-input" />
      </div>
      <div v-if="editingPipeline?.kind === 'application'" class="space-y-1.5">
        <label class="app-field-label">应用</label>
        <ComboboxSelect
          v-model="editForm.applicationId"
          :options="editApplicationOptions"
          :disabled="!canEditBindApplication"
          placeholder="可选；绑定后不可更改"
        />
      </div>
    </form>
    <template #footer>
      <AppDialogActions :busy="operating" @cancel="editOpen = false" @confirm="savePipelineInfo" />
    </template>
  </AppDialog>

  <AppDialog
    v-model:open="instantiateOpen"
    title="从模板创建应用流水线"
    width-class="w-[min(760px,calc(100vw-32px))]"
    body-class="max-h-[72vh] space-y-4 overflow-y-auto px-6 py-4"
  >
    <form class="space-y-4" @submit.prevent="instantiate">
      <div class="space-y-1.5">
        <label class="app-field-label">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="instantiateForm.name"
          class="app-input"
          :class="instantiateErrors.name ? 'app-input-error' : ''"
          :aria-invalid="instantiateErrors.name ? 'true' : undefined"
          @input="instantiateErrors.name = ''"
        />
        <p v-if="instantiateErrors.name" class="app-field-error" role="alert">
          {{ instantiateErrors.name }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label">
          代码仓库
          <span class="text-destructive">*</span>
        </label>
        <ComboboxSelect
          v-model="instantiateForm.repositoryId"
          :options="repositoryOptions"
          :invalid="Boolean(instantiateErrors.repositoryId)"
          placeholder="选择仓库"
          @update:model-value="instantiateErrors.repositoryId = ''"
        />
        <p v-if="instantiateErrors.repositoryId" class="app-field-error" role="alert">
          {{ instantiateErrors.repositoryId }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label">应用</label>
        <ComboboxSelect
          v-model="instantiateForm.applicationId"
          :options="applicationOptions"
          :invalid="Boolean(instantiateErrors.applicationId)"
          placeholder="可选；绑定镜像制品到组件时需要"
          @update:model-value="changeInstantiationApplication"
        />
        <p v-if="instantiateErrors.applicationId" class="app-field-error" role="alert">
          {{ instantiateErrors.applicationId }}
        </p>
      </div>
      <template v-if="dockerArtifacts.length && instantiateForm.applicationId">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label">
              来源版本策略
              <span v-if="hasArtifactBindings" class="text-destructive">*</span>
            </label>
            <RawValueSelect
              :model-value="instantiateForm.versionForkStrategy"
              :values="['latest', 'fixed']"
              :invalid="Boolean(instantiateErrors.versionForkStrategy)"
              @update:model-value="changeVersionForkStrategy"
            />
            <p v-if="instantiateErrors.versionForkStrategy" class="app-field-error" role="alert">
              {{ instantiateErrors.versionForkStrategy }}
            </p>
          </div>
          <div v-if="instantiateForm.versionForkStrategy === 'fixed'" class="space-y-1.5">
            <label class="app-field-label">
              来源版本
              <span class="text-destructive">*</span>
            </label>
            <ComboboxSelect
              v-model="instantiateForm.fixedVersionId"
              :options="versionOptions"
              :invalid="Boolean(instantiateErrors.fixedVersionId)"
              description-inline
              placeholder="选择版本"
              @update:model-value="changeFixedVersion"
            />
            <p v-if="instantiateErrors.fixedVersionId" class="app-field-error" role="alert">
              {{ instantiateErrors.fixedVersionId }}
            </p>
          </div>
        </div>
        <div class="space-y-3 border-t border-border pt-4">
          <div>
            <h3 class="app-field-label">Docker 制品绑定</h3>
            <p class="mt-1 text-sm text-muted-foreground">
              可选。绑定后该镜像制品会在成功 Run 时更新目标组件；不绑定则只作追溯。
            </p>
          </div>
          <AppLoadingState v-if="sourceVersionLoading" size="compact" />
          <p v-else-if="sourceVersionError" class="app-field-error" role="alert">
            {{ sourceVersionError }}
          </p>
          <div v-else class="space-y-3">
            <div
              v-for="artifact in dockerArtifacts"
              :key="artifact.key"
              class="grid gap-2 border-b border-border pb-3 last:border-0 sm:grid-cols-[minmax(0,1fr)_minmax(220px,1fr)] sm:items-center"
            >
              <div class="min-w-0">
                <p class="truncate text-sm font-medium text-foreground">{{ artifact.name }}</p>
                <p class="truncate text-xs text-muted-foreground">{{ artifact.stageName }}</p>
              </div>
              <div class="space-y-1.5">
                <ComboboxSelect
                  v-model="artifactBindings[artifact.key]"
                  :options="componentOptions"
                  :invalid="Boolean(artifactBindingErrors[artifact.key])"
                  :disabled="componentOptions.length === 0"
                  placeholder="不绑定组件"
                  @update:model-value="clearArtifactBindingError(artifact.key)"
                />
                <p v-if="artifactBindingErrors[artifact.key]" class="app-field-error" role="alert">
                  {{ artifactBindingErrors[artifact.key] }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </template>
      <p v-if="instantiateError" class="app-field-error" role="alert">{{ instantiateError }}</p>
    </form>
    <template #footer>
      <AppDialogActions
        :busy="operating"
        @cancel="instantiateOpen = false"
        @confirm="instantiate"
      />
    </template>
  </AppDialog>

  <AppDialog
    v-model:open="deleteOpen"
    title="确认删除"
    width-class="w-[min(420px,calc(100vw-32px))]"
  >
    <p class="text-sm text-foreground">确定要删除这个流水线吗？此操作不可恢复。</p>
    <p v-if="deleteError" class="app-field-error mt-3" role="alert">{{ deleteError }}</p>
    <template #footer>
      <AppDialogActions
        :busy="operating"
        variant="destructive"
        @cancel="deleteOpen = false"
        @confirm="deletePipeline"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { Plus } from '@lucide/vue';
  import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { pipelineApi } from '@/api/pipeline/pipeline';
  import { repositoryApi } from '@/api/repository/repository';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import RawValueSelect, { type RawValue } from '@/components/RawValueSelect.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { PipelineResp } from '@/gen/proto/orbit/v1/pipeline/pipeline';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const pipelines = ref<PipelineResp[]>([]);
  const applications = ref<ApplicationResp[]>([]);
  const repositories = ref<RepositoryResp[]>([]);
  const versions = ref<VersionResp[]>([]);
  const sourceVersion = ref<VersionResp>();
  const sourceVersionLoading = ref(false);
  const sourceVersionError = ref('');
  const search = ref('');
  const kind = ref<'all' | 'template' | 'application'>('all');
  const pipelineKindOptions = [
    { value: 'all', label: '全部类型' },
    { value: 'template', label: '模板' },
    { value: 'application', label: '应用流水线' },
  ];
  const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
  const createOpen = ref(false);
  const editOpen = ref(false);
  const instantiateOpen = ref(false);
  const deleteOpen = ref(false);
  const selectedTemplate = ref<PipelineResp>();
  const editingPipeline = ref<PipelineResp>();
  const pendingDeleteId = ref('');
  const createForm = reactive({ name: '', description: '' });
  const editForm = reactive({ name: '', description: '', applicationId: '' });
  const editApplications = ref<ApplicationResp[]>([]);
  const instantiateForm = reactive({
    name: '',
    applicationId: '',
    repositoryId: '',
    versionForkStrategy: 'latest',
    fixedVersionId: '',
  });
  const createError = ref('');
  const editError = ref('');
  const instantiateError = ref('');
  const deleteError = ref('');
  const instantiateErrors = reactive({
    name: '',
    applicationId: '',
    repositoryId: '',
    versionForkStrategy: '',
    fixedVersionId: '',
  });
  const artifactBindings = reactive<Record<string, string>>({});
  const artifactBindingErrors = reactive<Record<string, string>>({});

  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const applicationOptions = computed(() =>
    applications.value.map((application) => ({ value: application.id, label: application.name }))
  );
  const canEditBindApplication = computed(
    () => editingPipeline.value?.kind === 'application' && !editingPipeline.value.application_id
  );
  const editApplicationOptions = computed(() => {
    const options = editApplications.value.map((application) => ({
      value: application.id,
      label: application.name,
    }));
    const currentId = editingPipeline.value?.application_id;
    const currentName = editingPipeline.value?.application_name;
    if (currentId && currentName && !options.some((option) => option.value === currentId)) {
      options.unshift({ value: currentId, label: currentName });
    }
    return options;
  });
  const repositoryOptions = computed(() =>
    repositories.value.map((repository) => ({
      value: repository.id,
      label: repository.name,
    }))
  );
  const versionOptions = computed(() =>
    versions.value.map((version) => ({
      value: version.id,
      label: version.label,
      description: version.status,
    }))
  );
  const componentOptions = computed(() =>
    (sourceVersion.value?.components || []).map((component) => ({
      value: component.name,
      label: component.name,
    }))
  );
  const dockerArtifacts = computed(() =>
    (selectedTemplate.value?.stage_nodes || []).flatMap((stage) =>
      stage.artifacts
        .filter((artifact) => artifact.collector === 'docker_image')
        .map((artifact) => ({
          key: `${stage.id}:${artifact.name}`,
          stageId: stage.id,
          stageName: stage.name,
          name: artifact.name,
        }))
    )
  );
  const selectedArtifactBindings = computed(() =>
    dockerArtifacts.value
      .map((artifact) => ({
        stage_id: artifact.stageId,
        artifact_name: artifact.name,
        component_name: String(artifactBindings[artifact.key] || '').trim(),
      }))
      .filter((binding) => binding.component_name)
  );
  const hasArtifactBindings = computed(() => selectedArtifactBindings.value.length > 0);

  async function fetchPipelines() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await execute(async () => {
        const response = await pipelineApi.list({
          project_id: projectId,
          kind: kind.value === 'all' ? undefined : kind.value,
          search: search.value.trim() || undefined,
          page: pagination.current,
          per_page: pagination.pageSize,
        });
        pipelines.value = response.items;
        pagination.total = response.total;
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载流水线失败');
    }
  }

  async function loadInstantiationOptions() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) return;
    const [applicationResponse, repositoryResponse] = await Promise.all([
      applicationApi.list({ project_id: projectId, per_page: 100 }),
      repositoryApi.list({ project_id: projectId, per_page: 100 }),
    ]);
    applications.value = applicationResponse.items;
    repositories.value = repositoryResponse.items;
  }

  function searchPipelines() {
    pagination.current = 1;
    void fetchPipelines();
  }

  function updatePipelineKind(value: string | number) {
    if (value !== 'all' && value !== 'template' && value !== 'application') {
      return;
    }
    kind.value = value;
    searchPipelines();
  }

  function goPage(page: number) {
    pagination.current = page;
    void fetchPipelines();
  }

  function changePageSize(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    void fetchPipelines();
  }

  function openCreateDialog() {
    Object.assign(createForm, { name: '', description: '' });
    createError.value = '';
    createOpen.value = true;
  }

  async function openEditDialog(pipeline: PipelineResp) {
    editingPipeline.value = pipeline;
    Object.assign(editForm, {
      name: pipeline.name,
      description: pipeline.description,
      applicationId: pipeline.application_id || '',
    });
    editError.value = '';
    editApplications.value = [];
    if (pipeline.kind === 'application') {
      const projectId = projectStore.activeProjectId || pipeline.project_id;
      if (!projectId) {
        toast.error('请先选择项目');
        return;
      }
      try {
        const page = await applicationApi.list({ project_id: projectId, per_page: 100 });
        editApplications.value = page.items ?? [];
      } catch (reason) {
        toast.error(reason instanceof Error ? reason.message : '加载应用列表失败');
        return;
      }
    }
    editOpen.value = true;
  }

  async function savePipelineInfo() {
    const pipeline = editingPipeline.value;
    editError.value = editForm.name.trim() ? '' : '请输入流水线名称';
    if (!pipeline || editError.value) return;
    const payload: {
      name: string;
      description: string;
      application_id?: string;
    } = {
      name: editForm.name.trim(),
      description: editForm.description,
    };
    if (canEditBindApplication.value && editForm.applicationId) {
      payload.application_id = editForm.applicationId;
    }
    try {
      await executeOperation(async () => {
        const updated = await pipelineApi.update(pipeline.id, payload);
        pipelines.value = pipelines.value.map((item) => (item.id === updated.id ? updated : item));
        editingPipeline.value = updated;
        editOpen.value = false;
        toast.success('流水线信息已保存');
      });
    } catch (reason) {
      editError.value = reason instanceof Error ? reason.message : '保存失败';
    }
  }

  async function createTemplate() {
    createError.value = createForm.name.trim() ? '' : '请输入模板名称';
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      createError.value = '请先选择项目';
      return;
    }
    if (createError.value) return;
    try {
      await executeOperation(async () => {
        const pipeline = await pipelineApi.create(
          {
            kind: 'template',
            name: createForm.name.trim(),
            description: createForm.description,
            variable_declarations: [],
          },
          { project_id: projectId }
        );
        createOpen.value = false;
        toast.success('模板已创建');
        await router.push(`/pipeline/${pipeline.id}`);
      });
    } catch (reason) {
      createError.value = reason instanceof Error ? reason.message : '创建模板失败';
    }
  }

  async function openInstantiateDialog(template: PipelineResp) {
    const detail = await pipelineApi.get(template.id);
    selectedTemplate.value = detail;
    Object.assign(instantiateForm, {
      name: `${detail.name}-应用流水线`,
      applicationId: '',
      repositoryId: '',
      versionForkStrategy: 'latest',
      fixedVersionId: '',
    });
    Object.assign(instantiateErrors, {
      name: '',
      applicationId: '',
      repositoryId: '',
      versionForkStrategy: '',
      fixedVersionId: '',
    });
    resetArtifactBindings();
    versions.value = [];
    sourceVersion.value = undefined;
    sourceVersionError.value = '';
    instantiateError.value = '';
    try {
      await loadInstantiationOptions();
      instantiateOpen.value = true;
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载应用或仓库失败');
    }
  }

  function resetArtifactBindings() {
    for (const key of Object.keys(artifactBindings)) delete artifactBindings[key];
    for (const key of Object.keys(artifactBindingErrors)) delete artifactBindingErrors[key];
  }

  async function changeInstantiationApplication(value: ComboboxOptionValue) {
    instantiateForm.applicationId = String(value || '');
    instantiateErrors.applicationId = '';
    instantiateErrors.fixedVersionId = '';
    instantiateForm.fixedVersionId = '';
    resetArtifactBindings();
    versions.value = [];
    sourceVersion.value = undefined;
    sourceVersionError.value = '';
    if (!instantiateForm.applicationId) return;
    try {
      const response = await applicationApi.listVersions(instantiateForm.applicationId, {
        per_page: 100,
      });
      versions.value = response.items;
      await loadSourceVersion();
    } catch (reason) {
      sourceVersionError.value = reason instanceof Error ? reason.message : '加载应用版本失败';
    }
  }

  async function changeVersionForkStrategy(value: RawValue) {
    instantiateForm.versionForkStrategy = String(value);
    instantiateErrors.versionForkStrategy = '';
    instantiateErrors.fixedVersionId = '';
    instantiateForm.fixedVersionId = '';
    resetArtifactBindings();
    await loadSourceVersion();
  }

  async function changeFixedVersion(value: ComboboxOptionValue) {
    instantiateForm.fixedVersionId = String(value || '');
    instantiateErrors.fixedVersionId = '';
    resetArtifactBindings();
    await loadSourceVersion();
  }

  async function loadSourceVersion() {
    sourceVersion.value = undefined;
    sourceVersionError.value = '';
    if (!instantiateForm.applicationId) return;
    const versionId =
      instantiateForm.versionForkStrategy === 'fixed'
        ? instantiateForm.fixedVersionId
        : versions.value[0]?.id;
    if (!versionId) {
      sourceVersionError.value = '应用尚无可用版本，无法绑定 Docker 制品。';
      return;
    }
    sourceVersionLoading.value = true;
    try {
      sourceVersion.value = await applicationApi.getVersion(versionId);
    } catch (reason) {
      sourceVersionError.value = reason instanceof Error ? reason.message : '加载来源版本失败';
    } finally {
      sourceVersionLoading.value = false;
    }
  }

  function clearArtifactBindingError(key: string) {
    delete artifactBindingErrors[key];
  }

  async function openInstantiationFromQuery(templateID: unknown) {
    const id = typeof templateID === 'string' ? templateID.trim() : '';
    if (!id) return;
    try {
      const template = await pipelineApi.get(id);
      if (template.kind !== 'template') {
        throw new Error('只能从模板创建应用流水线');
      }
      await openInstantiateDialog(template);
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载模板失败');
    } finally {
      const query = { ...route.query };
      delete query.instantiate;
      await router.replace({ query });
    }
  }

  async function instantiate() {
    for (const key of Object.keys(artifactBindingErrors)) delete artifactBindingErrors[key];
    const selectedComponents = new Map<string, string>();
    for (const artifact of dockerArtifacts.value) {
      const componentName = String(artifactBindings[artifact.key] || '').trim();
      if (!componentName) continue;
      if (selectedComponents.has(componentName)) {
        artifactBindingErrors[artifact.key] = '同一组件只能绑定一个 Docker 制品';
      } else {
        selectedComponents.set(componentName, artifact.key);
      }
    }
    const bindings = selectedArtifactBindings.value;
    const needsApplication = bindings.length > 0;

    instantiateErrors.name = instantiateForm.name.trim() ? '' : '请输入流水线名称';
    instantiateErrors.repositoryId = instantiateForm.repositoryId ? '' : '请选择代码仓库';
    // Application is optional unless component-bound docker artifacts are selected.
    instantiateErrors.applicationId =
      needsApplication && !instantiateForm.applicationId ? '绑定 Docker 制品时请选择应用' : '';
    instantiateErrors.versionForkStrategy =
      needsApplication && !instantiateForm.versionForkStrategy ? '请选择来源版本策略' : '';
    instantiateErrors.fixedVersionId =
      needsApplication &&
      instantiateForm.versionForkStrategy === 'fixed' &&
      !instantiateForm.fixedVersionId
        ? '请选择来源版本'
        : '';
    if (needsApplication && instantiateForm.applicationId && !sourceVersion.value) {
      for (const artifact of dockerArtifacts.value) {
        if (String(artifactBindings[artifact.key] || '').trim()) {
          artifactBindingErrors[artifact.key] = '请先加载包含目标组件的来源版本';
        }
      }
    }
    instantiateError.value = '';
    const template = selectedTemplate.value;
    if (
      !template ||
      Object.values(instantiateErrors).some(Boolean) ||
      Object.values(artifactBindingErrors).some(Boolean)
    )
      return;
    try {
      await executeOperation(async () => {
        const pipeline = await pipelineApi.instantiate(template.id, {
          name: instantiateForm.name.trim(),
          application_id: instantiateForm.applicationId || undefined,
          repository_id: instantiateForm.repositoryId,
          version_fork_strategy: needsApplication ? instantiateForm.versionForkStrategy : undefined,
          fixed_version_id:
            needsApplication && instantiateForm.versionForkStrategy === 'fixed'
              ? instantiateForm.fixedVersionId
              : undefined,
          artifact_bindings: bindings,
        });
        instantiateOpen.value = false;
        toast.success('应用流水线已创建');
        await router.push(`/pipeline/${pipeline.id}`);
      });
    } catch (reason) {
      instantiateError.value = reason instanceof Error ? reason.message : '创建应用流水线失败';
    }
  }

  async function openDeleteDialog(pipeline: PipelineResp) {
    pendingDeleteId.value = pipeline.id;
    deleteError.value = '';
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
    deleteOpen.value = true;
  }

  async function deletePipeline() {
    deleteError.value = '';
    try {
      await executeOperation(async () => {
        await pipelineApi.delete(pendingDeleteId.value);
        toast.success('流水线已删除');
        deleteOpen.value = false;
        if (pipelines.value.length === 1 && pagination.current > 1) pagination.current -= 1;
        await fetchPipelines();
      });
    } catch (reason) {
      deleteError.value = reason instanceof Error ? reason.message : '删除流水线失败';
    }
  }

  watch(
    () => route.query.instantiate,
    (templateID) => {
      void openInstantiationFromQuery(templateID);
    }
  );

  onMounted(async () => {
    await fetchPipelines();
    await openInstantiationFromQuery(route.query.instantiate);
  });
</script>
