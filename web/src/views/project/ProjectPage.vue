<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('project.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('project.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button px-4" @click="openHandoverDialog">
          <Upload class="size-4" />
          {{ t('project.importHandover') }}
        </button>
        <button class="app-button-primary px-5" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('project.createProject') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
        <p class="text-sm">{{ error || t('project.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="filteredProjects.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[900px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[15%]" />
            <col class="w-[10%]" />
            <col class="w-[20%]" />
            <col class="w-[20%]" />
            <col class="w-[15%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('project.name') }}</th>
              <th>{{ t('project.code') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.updatedAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="project in pagedProjects" :key="project.id">
              <td class="max-w-0 truncate text-foreground" :title="project.name">
                <router-link :to="`/project/${project.id}`" class="app-link">
                  {{ project.name }}
                </router-link>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ project.code }}</td>
              <td>
                <AppBadge v-if="project.is_active" variant="status" tone="success">
                  {{ t('project.active') }}
                </AppBadge>
                <AppBadge v-else variant="status" tone="default">
                  {{ t('project.deprecated') }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(project.created_at) }}
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(project.updated_at) }}
              </td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditDialog(project)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    v-if="project.is_active"
                    class="app-link-danger"
                    @click="openDeprecateDialog(project)"
                  >
                    {{ t('project.deprecate') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <ListPagination
        :current="pagination.current"
        :page-size="pagination.pageSize"
        :total="filteredProjects.length"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <AppDialog
      v-model:open="isDialogOpen"
      :title="editingProject ? t('project.editProject') : t('project.createProject')"
      width-class="w-[min(760px,calc(100vw-32px))]"
      body-class="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4"
      content-class="max-h-[calc(100vh-32px)] flex flex-col"
    >
      <form class="space-y-4" novalidate @submit.prevent="handleSave">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="project-name">
            {{ t('project.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="project-name"
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            :aria-invalid="errors.name ? 'true' : undefined"
            @input="errors.name = ''"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <template v-if="!editingProject">
          <div class="space-y-1.5">
            <label class="app-field-label block" for="project-code">
              {{ t('project.code') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              id="project-code"
              v-model="form.code"
              type="text"
              class="app-input"
              :class="errors.code ? 'app-input-error' : ''"
              :placeholder="t('project.codeHint')"
              :aria-invalid="errors.code ? 'true' : undefined"
              @input="errors.code = ''"
            />
            <p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
          </div>
        </template>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="submitError" class="app-field-error mt-3" role="alert">{{ submitError }}</p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="isDialogOpen = false" @confirm="handleSave" />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isHandoverDialogOpen"
      :title="t('project.importHandover')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form class="space-y-4" novalidate @submit.prevent="handleHandoverImport">
        <div class="space-y-1.5">
          <label id="handover-file-label" class="app-field-label block">
            {{ t('project.handoverFile') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="handover-file"
            ref="handoverFileInput"
            :key="handoverFileInputKey"
            type="file"
            accept="application/json,.json"
            class="sr-only"
            :disabled="operating"
            :aria-invalid="handoverErrors.file ? 'true' : undefined"
            :aria-labelledby="'handover-file-label'"
            :aria-describedby="handoverErrors.file ? 'handover-file-error' : undefined"
            @change="handleHandoverFile"
          />
          <div
            class="flex min-h-28 flex-col gap-3 rounded-md border border-dashed bg-muted/20 p-4 sm:flex-row sm:items-center"
            :class="[
              handoverErrors.file ? 'border-destructive' : 'border-input',
              operating ? 'cursor-not-allowed opacity-60' : 'hover:border-ring',
            ]"
            :aria-labelledby="'handover-file-label'"
            @dragover.prevent
            @drop.prevent="handleHandoverFileDrop"
          >
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary"
              aria-hidden="true"
            >
              <FileUp class="size-5" />
            </span>
            <template v-if="handoverFile">
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-medium text-foreground" :title="handoverFile.name">
                  {{ handoverFile.name }}
                </p>
                <p class="mt-0.5 text-xs text-muted-foreground">
                  {{ formatHandoverFileSize(handoverFile.size) }}
                </p>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button
                  type="button"
                  class="app-button h-9 px-3"
                  :disabled="operating"
                  @click="selectHandoverFile"
                >
                  <FileUp class="size-4" />
                  {{ t('project.handoverChooseFile') }}
                </button>
                <button
                  type="button"
                  class="app-button h-9 w-9 p-0"
                  :disabled="operating"
                  :aria-label="t('common.remove')"
                  :title="t('common.remove')"
                  @click="clearHandoverFile"
                >
                  <X class="size-4" aria-hidden="true" />
                </button>
              </div>
            </template>
            <template v-else>
              <p class="min-w-0 flex-1 text-sm text-muted-foreground">
                {{ t('project.handoverFileEmpty') }}
              </p>
              <button
                type="button"
                class="app-button-primary h-9 shrink-0 px-3"
                :disabled="operating"
                @click="selectHandoverFile"
              >
                <FileUp class="size-4" />
                {{ t('project.handoverChooseFile') }}
              </button>
            </template>
          </div>
          <p
            v-if="handoverErrors.file"
            id="handover-file-error"
            class="app-field-error text-xs"
            role="alert"
          >
            {{ handoverErrors.file }}
          </p>
        </div>

        <div class="space-y-1.5">
          <label class="app-field-label block" for="handover-decryption-key">
            {{ t('project.handoverDecryptionKey') }}
          </label>
          <input
            id="handover-decryption-key"
            v-model="handoverDecryptionKey"
            type="password"
            autocomplete="off"
            class="app-input"
            :placeholder="t('project.handoverDecryptionKeyHint')"
            :disabled="operating || !handoverOverrideEnvironment"
          />
        </div>

        <label
          class="flex items-start gap-2 text-sm text-foreground"
          for="handover-override-environment"
        >
          <input
            id="handover-override-environment"
            v-model="handoverOverrideEnvironment"
            type="checkbox"
            class="app-checkbox mt-0.5"
            :disabled="operating"
          />
          <span>{{ t('project.handoverOverrideEnvironment') }}</span>
        </label>

        <div class="space-y-1.5">
          <span id="handover-mode-label" class="app-field-label block">
            {{ t('project.handoverMode') }}
          </span>
          <div
            class="flex h-10 items-center gap-3"
            role="group"
            aria-labelledby="handover-mode-label"
          >
            <button
              type="button"
              class="text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50"
              :class="handoverMode === 'new' ? 'text-foreground' : 'text-muted-foreground'"
              :disabled="operating"
              @click="setHandoverMode(false)"
            >
              {{ t('project.handoverModeNew') }}
            </button>
            <SwitchRoot
              :model-value="handoverMode === 'replace'"
              class="app-switch-root"
              :disabled="operating"
              :aria-label="t('project.handoverMode')"
              @update:model-value="setHandoverMode"
            >
              <SwitchThumb class="app-switch-thumb" />
            </SwitchRoot>
            <button
              type="button"
              class="text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50"
              :class="handoverMode === 'replace' ? 'text-foreground' : 'text-muted-foreground'"
              :disabled="operating"
              @click="setHandoverMode(true)"
            >
              {{ t('project.handoverModeReplace') }}
            </button>
          </div>
        </div>

        <template v-if="handoverMode === 'new'">
          <div class="space-y-1.5">
            <label class="app-field-label block" for="handover-project-name">
              {{ t('project.name') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              id="handover-project-name"
              v-model="handoverForm.name"
              type="text"
              class="app-input"
              :class="handoverErrors.name ? 'app-input-error' : ''"
              :disabled="operating"
              :aria-invalid="handoverErrors.name ? 'true' : undefined"
              @input="handoverErrors.name = ''"
            />
            <p v-if="handoverErrors.name" class="app-field-error text-xs">
              {{ handoverErrors.name }}
            </p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block" for="handover-project-code">
              {{ t('project.code') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              id="handover-project-code"
              v-model="handoverForm.code"
              type="text"
              class="app-input"
              :class="handoverErrors.code ? 'app-input-error' : ''"
              :placeholder="t('project.codeHint')"
              :disabled="operating"
              :aria-invalid="handoverErrors.code ? 'true' : undefined"
              @input="handoverErrors.code = ''"
            />
            <p v-if="handoverErrors.code" class="app-field-error text-xs">
              {{ handoverErrors.code }}
            </p>
          </div>
        </template>
        <div v-else class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('project.handoverTargetProject') }}
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            v-model="handoverTargetProjectId"
            :options="handoverTargetOptions"
            :placeholder="t('project.handoverTargetProject')"
            :disabled="operating"
            :invalid="Boolean(handoverErrors.targetProject)"
            description-inline
            @update:model-value="handoverErrors.targetProject = ''"
          />
          <p v-if="handoverErrors.targetProject" class="app-field-error text-xs" role="alert">
            {{ handoverErrors.targetProject }}
          </p>
        </div>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="handoverSubmitError" class="app-field-error mt-3" role="alert">
        {{ handoverSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('project.importHandover')"
          @cancel="closeHandoverDialog"
          @confirm="handleHandoverImport"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isDeprecateDialogOpen" :title="t('project.deprecateProject')">
      <p class="text-sm text-foreground">
        {{ t('project.deprecateConfirmPrefix') }}
        <span>{{ deprecatingProject?.name }}</span>
        {{ t('project.deprecateConfirmSuffix') }}
      </p>
      <p v-if="deprecateSubmitError" class="app-field-error mt-3" role="alert">
        {{ deprecateSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeprecateDialogOpen = false"
          @confirm="handleDeprecate"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { FileUp, Plus, Upload, X } from '@lucide/vue';
  import { computed, nextTick, onMounted, reactive, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { SwitchRoot, SwitchThumb, ToolbarRoot } from 'reka-ui';
  import { projectApi } from '@/api/project/project';
  import type { ProjectResp } from '@/gen/proto/orbit/v1/project/project';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const router = useRouter();
  const route = useRoute();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const searchText = ref('');
  const appliedSearch = ref('');
  const isDialogOpen = ref(false);
  const isDeprecateDialogOpen = ref(false);
  const isHandoverDialogOpen = ref(false);
  const editingProject = ref<ProjectResp | null>(null);
  const deprecatingProject = ref<ProjectResp | null>(null);
  const pagination = reactive({ current: 1, pageSize: 10 });
  const form = reactive({
    name: '',
    code: '',
  });
  const errors = reactive({
    name: '',
    code: '',
  });
  const submitError = ref('');
  const deprecateSubmitError = ref('');
  const handoverMode = ref<'new' | 'replace'>('new');
  const handoverFile = ref<File>();
  const handoverFileInput = ref<HTMLInputElement>();
  const handoverFileInputKey = ref(0);
  const handoverForm = reactive({
    name: '',
    code: '',
  });
  const handoverTargetProjectId = ref('');
  const handoverDecryptionKey = ref('');
  const handoverOverrideEnvironment = ref(true);
  const handoverErrors = reactive({ file: '', name: '', code: '', targetProject: '' });
  const handoverSubmitError = ref('');

  const filteredProjects = computed(() => {
    const keyword = appliedSearch.value.trim().toLowerCase();
    if (!keyword) {
      return projectStore.projects;
    }
    return projectStore.projects.filter(
      (project) =>
        project.name.toLowerCase().includes(keyword) || project.code.toLowerCase().includes(keyword)
    );
  });
  const totalPages = computed(() =>
    Math.max(1, Math.ceil(filteredProjects.value.length / pagination.pageSize))
  );
  const pagedProjects = computed(() => {
    const start = (pagination.current - 1) * pagination.pageSize;
    return filteredProjects.value.slice(start, start + pagination.pageSize);
  });
  const handoverTargetOptions = computed(() =>
    projectStore.activeProjects.map((project) => ({
      value: project.id,
      label: project.name,
      description: project.code,
    }))
  );

  function resetForm(project?: ProjectResp) {
    form.name = project?.name ?? '';
    form.code = project?.code ?? '';
    Object.keys(errors).forEach((key) => {
      errors[key as keyof typeof errors] = '';
    });
    submitError.value = '';
  }

  function validate() {
    errors.name = form.name.trim() ? '' : t('project.nameRequired');
    if (editingProject.value) {
      return !errors.name;
    }
    errors.code = /^[a-z0-9_-]+$/.test(form.code) ? '' : t('project.codeInvalid');
    return !errors.name && !errors.code;
  }

  function openCreateDialog() {
    editingProject.value = null;
    resetForm();
    isDialogOpen.value = true;
  }

  function openHandoverDialog() {
    handoverMode.value = 'new';
    handoverFile.value = undefined;
    handoverFileInputKey.value += 1;
    handoverForm.name = '';
    handoverForm.code = '';
    handoverTargetProjectId.value = '';
    handoverDecryptionKey.value = '';
    handoverOverrideEnvironment.value = true;
    handoverErrors.file = '';
    handoverErrors.name = '';
    handoverErrors.code = '';
    handoverErrors.targetProject = '';
    handoverSubmitError.value = '';
    isHandoverDialogOpen.value = true;
  }

  function setHandoverMode(replace: boolean) {
    handoverMode.value = replace ? 'replace' : 'new';
    handoverErrors.name = '';
    handoverErrors.code = '';
    handoverErrors.targetProject = '';
  }

  function closeHandoverDialog() {
    isHandoverDialogOpen.value = false;
    handoverSubmitError.value = '';
  }

  function handleHandoverFile(event: Event) {
    setHandoverFile((event.target as HTMLInputElement).files?.[0]);
  }

  function handleHandoverFileDrop(event: DragEvent) {
    if (operating.value) {
      return;
    }
    setHandoverFile(event.dataTransfer?.files?.[0]);
  }

  function selectHandoverFile() {
    if (operating.value || !handoverFileInput.value) {
      return;
    }
    handoverFileInput.value.value = '';
    handoverFileInput.value.click();
  }

  function clearHandoverFile() {
    handoverFile.value = undefined;
    handoverFileInputKey.value += 1;
    handoverErrors.file = '';
  }

  function setHandoverFile(file?: File) {
    if (!file) {
      return;
    }
    handoverFile.value = file;
    handoverErrors.file = '';
  }

  function formatHandoverFileSize(bytes: number) {
    const units = ['B', 'KB', 'MB', 'GB'];
    let value = bytes;
    let unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
      value /= 1024;
      unitIndex += 1;
    }
    return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
  }

  function openEditDialog(project: ProjectResp) {
    editingProject.value = project;
    resetForm(project);
    isDialogOpen.value = true;
  }

  async function openDeprecateDialog(project: ProjectResp) {
    deprecatingProject.value = project;
    deprecateSubmitError.value = '';
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
    isDeprecateDialogOpen.value = true;
  }

  function handleSearch() {
    appliedSearch.value = searchText.value;
    pagination.current = 1;
    void fetchProjects();
  }

  function goPage(page: number) {
    pagination.current = page;
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
  }

  async function fetchProjects() {
    await execute(async () => {
      await projectStore.fetchProjects();
    });
  }

  async function handleSave() {
    submitError.value = '';
    if (!validate()) {
      return;
    }
    try {
      await executeOp(async () => {
        if (editingProject.value) {
          await projectStore.updateProject(editingProject.value.id, { name: form.name.trim() });
          toast.success(t('project.updated'));
          isDialogOpen.value = false;
        } else {
          const project = await projectStore.createProject({
            name: form.name.trim(),
            code: form.code.trim(),
          });
          toast.success(t('project.created'));
          isDialogOpen.value = false;
          projectStore.setActiveProject(project.id);
          await router.push({
            name: 'ProjectInitialization',
            params: { id: project.id },
            query: { redirect: '/gateway' },
          });
        }
      });
    } catch (error: unknown) {
      submitError.value = error instanceof Error ? error.message : t('project.saveFailed');
    }
  }

  async function handleDeprecate() {
    if (!deprecatingProject.value) {
      return;
    }
    deprecateSubmitError.value = '';
    const projectId = deprecatingProject.value.id;
    try {
      await executeOp(async () => {
        await projectStore.deprecateProject(projectId);
        toast.success(t('project.deprecatedToast'));
        isDeprecateDialogOpen.value = false;
      });
    } catch (error: unknown) {
      deprecateSubmitError.value = error instanceof Error ? error.message : t('project.saveFailed');
    }
  }

  function validateHandoverImport() {
    handoverErrors.file = handoverFile.value ? '' : t('project.handoverFileRequired');
    if (handoverMode.value === 'new') {
      handoverErrors.name = handoverForm.name.trim() ? '' : t('project.nameRequired');
      handoverErrors.code = /^[a-z0-9_-]+$/.test(handoverForm.code) ? '' : t('project.codeInvalid');
      handoverErrors.targetProject = '';
    } else {
      handoverErrors.name = '';
      handoverErrors.code = '';
      handoverErrors.targetProject = handoverTargetProjectId.value
        ? ''
        : t('project.handoverTargetRequired');
    }
    return (
      !handoverErrors.file &&
      !handoverErrors.name &&
      !handoverErrors.code &&
      !handoverErrors.targetProject &&
      Boolean(handoverFile.value)
    );
  }

  async function handleHandoverImport() {
    handoverSubmitError.value = '';
    if (!validateHandoverImport() || !handoverFile.value) {
      return;
    }
    try {
      await executeOp(async () => {
        const project = await projectApi.importHandover(handoverFile.value as File, {
          mode: handoverMode.value,
          name: handoverMode.value === 'new' ? handoverForm.name.trim() : undefined,
          code: handoverMode.value === 'new' ? handoverForm.code.trim() : undefined,
          targetProjectId:
            handoverMode.value === 'replace' ? handoverTargetProjectId.value : undefined,
          decryptionKey: handoverDecryptionKey.value.trim(),
          overrideEnvironment: handoverOverrideEnvironment.value,
        });
        await projectStore.fetchProjects();
        projectStore.setActiveProject(project.id);
        closeHandoverDialog();
        toast.success(t('project.handoverImported'));
        await router.push({ name: 'ProjectDetail', params: { id: project.id } });
      });
    } catch (error: unknown) {
      handoverSubmitError.value =
        error instanceof Error ? error.message : t('project.handoverImportFailed');
    }
  }

  onMounted(async () => {
    await fetchProjects();
    if (route.query.handover === 'import') {
      openHandoverDialog();
    }
  });
</script>
