<template>
  <div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div>
          <h1 class="app-detail-page-title break-words">
            {{ stage?.name ?? t('buildStageDetail.title') }}
          </h1>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="stage"
          class="app-button h-9 px-3"
          :disabled="duplicating"
          @click="handleDuplicate"
        >
          <Copy class="size-4" />
          {{ t('common.copy') }}
        </button>
        <button
          v-if="stage"
          class="app-button-danger h-9 px-3"
          :disabled="deleting"
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/pipeline/stage')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <AppLoadingState v-if="status === 'loading'" size="section" />

    <!-- Content -->
    <template v-else-if="stage">
      <!-- Basic Info Card -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('buildStageDetail.basicInfo') }}
          </h2>
          <button class="app-button-primary h-9 px-3" @click="openEditModal">
            <Pencil class="size-4" />
            {{ t('common.edit') }}
          </button>
        </div>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('common.name') }}</dt>
            <dd class="text-foreground">{{ stage.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('buildStageDetail.version') }}</dt>
            <dd class="text-foreground">v{{ stage.version }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('buildStageDetail.image') }}</dt>
            <dd class="text-foreground">{{ stage.image }}</dd>
          </div>
          <div v-if="stage.description" class="flex gap-2 sm:col-span-2">
            <dt>{{ t('common.description') }}</dt>
            <dd class="text-foreground">{{ stage.description }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(stage.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(stage.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- Script Card -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('buildStageDetail.script') }}
          </h2>
          <div class="flex items-center gap-2">
            <button class="app-button-primary h-9 px-3" @click="openScriptDrawer">
              <Pencil class="size-4" />
              {{ t('common.edit') }}
            </button>
            <button v-if="stage.script" class="app-button h-9 px-3" @click="handleCopyScript">
              <Copy class="size-4" />
              {{ t('common.copy') }}
            </button>
          </div>
        </div>
        <div v-if="stage.script" class="p-5">
          <MonacoEditor
            :model-value="stage.script"
            language="shell"
            height="200px"
            :readonly="true"
            squared
          />
        </div>
        <div v-else class="px-5 py-10 text-center text-muted-foreground">
          <p class="text-sm">{{ t('buildStageDetail.noScript') }}</p>
        </div>
      </div>

      <!-- Artifacts Card -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('buildStageDetail.artifactConfig') }}
          </h2>
          <button class="app-button-primary h-9 px-3" @click="openAddArtifactModal">
            <Plus class="size-4" />
            {{ t('buildStageDetail.addArtifact') }}
          </button>
        </div>
        <AppEmptyState v-if="sortableArtifacts.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[720px]">
            <thead>
              <tr>
                <th>#</th>
                <th>Collector</th>
                <th>{{ t('common.name') }}</th>
                <th>引用/命令</th>
                <th class="w-32">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(artifact, idx) in sortableArtifacts" :key="idx">
                <td class="text-muted-foreground">{{ idx + 1 }}</td>
                <td>
                  <AppBadge variant="pill">
                    {{ artifact.collector }}
                  </AppBadge>
                </td>
                <td class="text-foreground">{{ artifact.name }}</td>
                <td class="text-muted-foreground">
                  {{ artifact.collector === 'command' ? artifact.command : artifact.reference }}
                </td>
                <td class="w-32">
                  <div class="flex items-center gap-3">
                    <button class="app-link" :disabled="saving" @click="openEditArtifactModal(idx)">
                      {{ t('common.edit') }}
                    </button>
                    <button
                      class="app-link-danger"
                      :disabled="saving"
                      @click="confirmRemoveArtifact(idx)"
                    >
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">应用版本关联</h2>
          <div class="flex items-center gap-2">
            <button
              class="app-button-primary h-9 px-3"
              :disabled="saving"
              @click="openBuildVersionBindingDialog"
            >
              <Link2 class="size-4" />
              {{ stage.build_version_binding ? '编辑关联' : '配置关联' }}
            </button>
            <button
              v-if="stage.build_version_binding"
              class="app-button h-9 px-3"
              :disabled="saving"
              @click="clearBuildVersionBinding"
            >
              <Unlink class="size-4" />
              解除关联
            </button>
          </div>
        </div>
        <dl v-if="stage.build_version_binding" class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>应用</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ stage.build_version_binding.application_name }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>Fork 策略</dt>
            <dd class="text-foreground">{{ stage.build_version_binding.fork_strategy }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>目标组件</dt>
            <dd class="text-foreground">{{ stage.build_version_binding.component_name }}</dd>
          </div>
          <div v-if="stage.build_version_binding.fixed_version_id" class="flex gap-2">
            <dt>固定版本</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ stage.build_version_binding.fixed_version_id }}
            </dd>
          </div>
        </dl>
        <div v-else class="px-5 py-10 text-center text-sm text-muted-foreground">
          未关联应用版本
        </div>
      </div>
    </template>

    <AppDialog v-model:open="isEditDialogOpen" :title="t('buildStageDetail.editBuild')">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('common.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="editErrors.name ? 'app-input-error' : ''"
            :placeholder="t('buildStageDetail.namePlaceholder')"
            :aria-invalid="editErrors.name ? 'true' : undefined"
            @input="editErrors.name = ''"
          />
          <p v-if="editErrors.name" class="app-field-error" role="alert">
            {{ editErrors.name }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('buildStageDetail.image') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.image"
            type="text"
            class="app-input"
            :class="editErrors.image ? 'app-input-error' : ''"
            :placeholder="t('buildStageDetail.imagePlaceholder')"
            :aria-invalid="editErrors.image ? 'true' : undefined"
            @input="editErrors.image = ''"
          />
          <p v-if="editErrors.image" class="app-field-error" role="alert">
            {{ editErrors.image }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('buildStageDetail.descriptionOptional') }}
          </label>
          <input
            v-model="form.description"
            type="text"
            class="app-input"
            :placeholder="t('buildStageDetail.descriptionPlaceholder')"
          />
        </div>
      </div>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="isEditDialogOpen = false" @confirm="handleSave" />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isBuildVersionBindingDialogOpen"
      title="应用版本关联"
      width-class="w-[min(620px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            应用
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            :model-value="bindingForm.applicationId"
            :options="applicationOptions"
            placeholder="选择应用"
            @update:model-value="handleBindingApplicationChange"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            Fork 策略
            <span class="text-destructive">*</span>
          </label>
          <RawValueSelect
            :model-value="bindingForm.forkStrategy"
            :values="['latest', 'fixed']"
            @update:model-value="handleForkStrategyChange"
          />
        </div>
        <div v-if="bindingForm.forkStrategy === 'fixed'" class="space-y-1.5">
          <label class="app-field-label block">
            固定版本
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            :model-value="bindingForm.fixedVersionId"
            :options="versionOptions"
            placeholder="选择版本"
            @update:model-value="handleFixedVersionChange"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            目标组件
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            :model-value="bindingForm.componentName"
            :options="componentOptions"
            placeholder="选择组件"
            @update:model-value="handleBindingComponentChange"
          />
        </div>
        <p v-if="bindingError" class="app-field-error" role="alert">{{ bindingError }}</p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          @cancel="isBuildVersionBindingDialogOpen = false"
          @confirm="saveBuildVersionBinding"
        />
      </template>
    </AppDialog>

    <AppDrawer
      v-model:open="showScriptDrawer"
      :title="t('buildStageDetail.editScript')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
    >
      <div class="flex h-full min-h-0 flex-col p-4">
        <div class="min-h-0 flex-1">
          <MonacoEditor
            v-model="scriptTemp"
            language="shell"
            height="100%"
            :placeholder="t('buildStageDetail.scriptPlaceholder')"
          />
        </div>
      </div>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="closeScriptDrawer" @confirm="confirmScript" />
      </template>
    </AppDrawer>

    <AppDialog
      v-model:open="isArtifactDialogOpen"
      :title="
        artifactForm.isEdit ? t('buildStageDetail.editArtifact') : t('buildStageDetail.addArtifact')
      "
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            Collector
            <span class="text-destructive">*</span>
          </label>
          <RawValueSelect
            v-model="artifactForm.collector"
            :values="artifactCollectorValues"
            placeholder="选择 collector"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('common.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="artifactForm.name"
            type="text"
            class="app-input"
            :class="artifactErrors.name ? 'app-input-error' : ''"
            :placeholder="t('buildStageDetail.artifactNamePlaceholder')"
            :aria-invalid="artifactErrors.name ? 'true' : undefined"
            @input="artifactErrors.name = ''"
          />
          <p v-if="artifactErrors.name" class="app-field-error" role="alert">
            {{ artifactErrors.name }}
          </p>
        </div>
        <div v-if="artifactForm.collector !== 'command'" class="space-y-1.5">
          <label class="app-field-label block">
            引用
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="artifactForm.reference"
            type="text"
            class="app-input"
            :class="artifactErrors.reference ? 'app-input-error' : ''"
            :placeholder="
              artifactForm.collector === 'docker_image' ? '本地镜像引用' : '制品文件路径'
            "
            :aria-invalid="artifactErrors.reference ? 'true' : undefined"
            @input="artifactErrors.reference = ''"
          />
          <p v-if="artifactErrors.reference" class="app-field-error" role="alert">
            {{ artifactErrors.reference }}
          </p>
        </div>
        <template v-else>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              命令
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="artifactForm.command"
              type="text"
              class="app-input"
              :class="artifactErrors.command ? 'app-input-error' : ''"
              placeholder="在阶段运行环境中执行"
              :aria-invalid="artifactErrors.command ? 'true' : undefined"
              @input="artifactErrors.command = ''"
            />
            <p v-if="artifactErrors.command" class="app-field-error" role="alert">
              {{ artifactErrors.command }}
            </p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              输出格式
              <span class="text-destructive">*</span>
            </label>
            <RawValueSelect v-model="artifactForm.format" :values="artifactValueFormats" />
            <p v-if="artifactErrors.format" class="app-field-error" role="alert">
              {{ artifactErrors.format }}
            </p>
          </div>
        </template>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          @cancel="isArtifactDialogOpen = false"
          @confirm="handleSaveArtifact"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteArtifactDialogOpen"
      :title="t('buildStageDetail.deleteArtifact')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{
          t('buildStageDetail.deleteArtifactConfirm', {
            name: sortableArtifacts[artifactToDelete]?.name ?? '',
          })
        }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          variant="destructive"
          @cancel="isDeleteArtifactDialogOpen = false"
          @confirm="removeArtifact"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('buildStageDetail.deleteStage')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('buildStageDetail.deleteStageConfirm', { name: stage?.name ?? '' }) }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="deleting"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Copy, Link2, Pencil, Plus, Trash2, Unlink } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { pipelineStageApi } from '@/api/pipeline/pipeline_stage';
  import { applicationApi } from '@/api/application/application';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type {
    ArtifactConfigResp,
    PipelineStageResp,
  } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { formatTime } from '@/utils/time';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n({ useScope: 'global' });
  const stageId = computed(() => route.params.id as string);
  const toast = useToast();
  const projectStore = useProjectStore();

  const { status, execute } = useStatusAsync();
  const { loading: saving, execute: executeSave } = useStatusAsync();
  const { loading: deleting, execute: executeDelete } = useStatusAsync();
  const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

  const stage = ref<PipelineStageResp>();
  const isDeleteDialogOpen = ref(false);
  const isEditDialogOpen = ref(false);
  const isArtifactDialogOpen = ref(false);
  const isDeleteArtifactDialogOpen = ref(false);
  const isBuildVersionBindingDialogOpen = ref(false);
  const showScriptDrawer = ref(false);
  const scriptTemp = ref('');
  const artifactCollectorValues = ['file', 'command', 'docker_image'];
  const artifactValueFormats = ['text', 'git_object_id'];
  const form = reactive({ name: '', image: '', description: '' });
  const editErrors = reactive({ name: '', image: '' });
  const artifactForm = reactive({
    isEdit: false,
    order: -1,
    collector: 'docker_image',
    name: '',
    reference: '',
    command: '',
    format: '',
  });
  const artifactErrors = reactive({ name: '', reference: '', command: '', format: '' });
  const sortableArtifacts = ref<ArtifactConfigResp[]>([]);
  const artifactToDelete = ref(-1);
  const applications = ref<ApplicationResp[]>([]);
  const versions = ref<VersionResp[]>([]);
  const sourceVersion = ref<VersionResp>();
  const bindingError = ref('');
  const bindingForm = reactive({
    applicationId: '',
    componentName: '',
    forkStrategy: 'latest',
    fixedVersionId: '',
  });

  const applicationOptions = computed(() =>
    applications.value.map((application) => ({
      value: application.id,
      label: application.name,
      description: application.code,
    }))
  );
  const versionOptions = computed(() =>
    versions.value.map((version) => ({
      value: version.id,
      label: version.label,
      description: version.status,
    }))
  );
  const latestVersionId = computed(() =>
    versions.value.reduce(
      (latestId, version) => (version.id > latestId ? version.id : latestId),
      ''
    )
  );
  const componentOptions = computed(() =>
    (sourceVersion.value?.components ?? []).map((component) => ({
      value: component.name,
      label: component.name,
      description: component.image,
    }))
  );

  async function fetchStage() {
    try {
      await execute(async () => {
        stage.value = await pipelineStageApi.get(stageId.value);
        sortableArtifacts.value = stage.value.artifacts ? [...stage.value.artifacts] : [];
      });
    } catch {
      toast.error(t('buildStageDetail.fetchFailed'));
      router.push('/pipeline/stage');
    }
  }

  async function loadApplications() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    const response = await applicationApi.list({ project_id: projectId, per_page: 100 });
    applications.value = response.items;
  }

  async function loadVersions(applicationId: string) {
    if (!applicationId) {
      versions.value = [];
      sourceVersion.value = undefined;
      return;
    }
    const response = await applicationApi.listVersions(applicationId, { per_page: 100 });
    versions.value = response.items;
  }

  async function loadSourceVersion(versionId: string) {
    sourceVersion.value = versionId ? await applicationApi.getVersion(versionId) : undefined;
  }

  async function loadSourceVersionForStrategy() {
    const versionId =
      bindingForm.forkStrategy === 'fixed' ? bindingForm.fixedVersionId : latestVersionId.value;
    await loadSourceVersion(versionId);
  }

  async function openBuildVersionBindingDialog() {
    const binding = stage.value?.build_version_binding;
    Object.assign(bindingForm, {
      applicationId: binding?.application_id ?? '',
      componentName: binding?.component_name ?? '',
      forkStrategy: binding?.fork_strategy ?? 'latest',
      fixedVersionId: binding?.fixed_version_id ?? '',
    });
    bindingError.value = '';
    try {
      await loadApplications();
      await loadVersions(bindingForm.applicationId);
      await loadSourceVersionForStrategy();
      isBuildVersionBindingDialogOpen.value = true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '加载应用版本失败');
    }
  }

  async function handleBindingApplicationChange(value: string | number | boolean) {
    bindingForm.applicationId = String(value || '');
    bindingForm.fixedVersionId = '';
    bindingForm.componentName = '';
    sourceVersion.value = undefined;
    bindingError.value = '';
    try {
      await loadVersions(bindingForm.applicationId);
      await loadSourceVersionForStrategy();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '加载应用版本失败');
    }
  }

  async function handleFixedVersionChange(value: string | number | boolean) {
    bindingForm.fixedVersionId = String(value || '');
    bindingForm.componentName = '';
    bindingError.value = '';
    try {
      await loadSourceVersionForStrategy();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '加载版本组件失败');
    }
  }

  async function handleForkStrategyChange(value: string | number | boolean) {
    bindingForm.forkStrategy = String(value || 'latest');
    bindingForm.componentName = '';
    bindingError.value = '';
    try {
      await loadSourceVersionForStrategy();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '加载版本组件失败');
    }
  }

  function handleBindingComponentChange(value: string | number | boolean) {
    bindingForm.componentName = String(value || '');
    bindingError.value = '';
  }

  async function saveBuildVersionBinding() {
    if (!stage.value) {
      return;
    }
    bindingError.value =
      !bindingForm.applicationId || !bindingForm.componentName.trim()
        ? '请选择应用和目标组件'
        : bindingForm.forkStrategy === 'fixed' && !bindingForm.fixedVersionId
          ? '请选择固定版本'
          : '';
    if (bindingError.value) {
      return;
    }
    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId.value, {
          build_version_binding: {
            application_id: bindingForm.applicationId,
            component_name: bindingForm.componentName.trim(),
            fork_strategy: bindingForm.forkStrategy,
            fixed_version_id:
              bindingForm.forkStrategy === 'fixed' ? bindingForm.fixedVersionId : undefined,
          },
        });
        isBuildVersionBindingDialogOpen.value = false;
        toast.success('应用版本关联已保存');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存应用版本关联失败');
    }
  }

  async function clearBuildVersionBinding() {
    if (!stage.value) {
      return;
    }
    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId.value, {
          clear_build_version_binding: true,
        });
        toast.success('应用版本关联已解除');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '解除应用版本关联失败');
    }
  }

  function openEditModal() {
    if (!stage.value) {
      return;
    }
    Object.assign(form, {
      name: stage.value.name,
      image: stage.value.image,
      description: stage.value.description,
    });
    Object.assign(editErrors, { name: '', image: '' });
    isEditDialogOpen.value = true;
  }

  function openScriptDrawer() {
    scriptTemp.value = stage.value?.script ?? '';
    showScriptDrawer.value = true;
  }

  function closeScriptDrawer() {
    showScriptDrawer.value = false;
  }

  async function handleCopyScript() {
    if (!stage.value?.script) {
      return;
    }
    if (!navigator.clipboard) {
      toast.error(t('buildStageDetail.copyFailed'));
      return;
    }
    try {
      await navigator.clipboard.writeText(stage.value.script);
      toast.success(t('buildStageDetail.scriptCopied'));
    } catch {
      toast.error(t('buildStageDetail.copyFailed'));
    }
  }

  async function confirmScript() {
    try {
      await executeSave(async () => {
        const updated = await pipelineStageApi.update(stageId.value, {
          script: scriptTemp.value,
        });
        stage.value = updated;
        showScriptDrawer.value = false;
        toast.success(t('buildStageDetail.scriptSaved'));
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.saveFailed'));
    }
  }

  async function handleSave() {
    editErrors.name = form.name.trim() ? '' : t('buildStageDetail.nameRequired');
    editErrors.image = form.image.trim() ? '' : t('buildStageDetail.imageRequired');
    if (editErrors.name || editErrors.image) {
      return;
    }
    try {
      await executeSave(async () => {
        const updated = await pipelineStageApi.update(stageId.value, {
          name: form.name,
          image: form.image,
          description: form.description,
        });
        stage.value = updated;
        toast.success(t('buildStageDetail.updateSuccess'));
        isEditDialogOpen.value = false;
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.saveFailed'));
    }
  }

  function openAddArtifactModal() {
    Object.assign(artifactForm, {
      isEdit: false,
      order: -1,
      collector: 'docker_image',
      name: '',
      reference: '',
      command: '',
      format: '',
    });
    Object.assign(artifactErrors, { name: '', reference: '', command: '', format: '' });
    isArtifactDialogOpen.value = true;
  }

  function openEditArtifactModal(idx: number) {
    const artifact = sortableArtifacts.value[idx];
    if (!artifact) {
      return;
    }
    Object.assign(artifactForm, {
      isEdit: true,
      order: idx,
      collector: artifact.collector,
      name: artifact.name,
      reference: artifact.reference,
      command: artifact.command,
      format: artifact.format,
    });
    Object.assign(artifactErrors, { name: '', reference: '', command: '', format: '' });
    isArtifactDialogOpen.value = true;
  }

  function confirmRemoveArtifact(idx: number) {
    artifactToDelete.value = idx;
    isDeleteArtifactDialogOpen.value = true;
  }

  async function removeArtifact() {
    const idx = artifactToDelete.value;
    if (idx === -1) {
      return;
    }

    sortableArtifacts.value.splice(idx, 1);

    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId.value, {
          artifacts: { items: sortableArtifacts.value.length > 0 ? sortableArtifacts.value : [] },
        });
        toast.success(t('buildStageDetail.deleteSuccess'));
        isDeleteArtifactDialogOpen.value = false;
        artifactToDelete.value = -1;
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.deleteFailed'));
    }
  }

  async function handleSaveArtifact() {
    artifactErrors.name = artifactForm.name.trim()
      ? ''
      : t('buildStageDetail.artifactNameRequired');
    artifactErrors.reference =
      artifactForm.collector !== 'command' && !artifactForm.reference.trim() ? '请输入引用' : '';
    artifactErrors.command =
      artifactForm.collector === 'command' && !artifactForm.command.trim() ? '请输入命令' : '';
    artifactErrors.format =
      artifactForm.collector === 'command' && !artifactForm.format ? '请选择输出格式' : '';
    if (
      artifactErrors.name ||
      artifactErrors.reference ||
      artifactErrors.command ||
      artifactErrors.format
    ) {
      return;
    }

    const nextArtifact = {
      collector: artifactForm.collector,
      name: artifactForm.name.trim(),
      reference: artifactForm.collector === 'command' ? '' : artifactForm.reference.trim(),
      command: artifactForm.collector === 'command' ? artifactForm.command.trim() : '',
      format: artifactForm.collector === 'command' ? artifactForm.format : '',
    };

    if (artifactForm.isEdit) {
      sortableArtifacts.value[artifactForm.order] = nextArtifact;
    } else {
      if (sortableArtifacts.value.some((a) => a.name === nextArtifact.name)) {
        artifactErrors.name = t('buildStageDetail.artifactNameExists');
        return;
      }
      sortableArtifacts.value.push(nextArtifact);
    }

    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId.value, {
          artifacts: { items: sortableArtifacts.value },
        });
        toast.success(
          artifactForm.isEdit
            ? t('buildStageDetail.updateSuccess')
            : t('buildStageDetail.addSuccess')
        );
        isArtifactDialogOpen.value = false;
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.saveFailed'));
    }
  }

  function openDeleteModal() {
    isDeleteDialogOpen.value = true;
  }

  async function handleDuplicate() {
    try {
      await executeDuplicate(async () => {
        const newStage = await pipelineStageApi.duplicate(stageId.value, {});
        toast.success(t('buildStageDetail.duplicateSuccess'));
        router.push(`/pipeline/stage/${newStage.id}`);
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.duplicateFailed'));
    }
  }

  async function handleDelete() {
    try {
      await executeDelete(async () => {
        await pipelineStageApi.delete(stageId.value);
        toast.success(t('buildStageDetail.deleteSuccess'));
        router.push('/pipeline/stage');
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.deleteFailed'));
    }
  }

  watch(stageId, fetchStage);
  watch(
    () => bindingForm.forkStrategy,
    (strategy) => {
      if (strategy === 'latest') {
        bindingForm.fixedVersionId = '';
        sourceVersion.value = undefined;
      }
    }
  );
  onMounted(fetchStage);
</script>
