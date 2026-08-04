<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('application.toolbar')">
      <SearchControl
        v-model="searchText"
        :placeholder="t('application.searchPlaceholder')"
        :loading="status === 'loading'"
        class="shrink-0"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button
          class="app-button-primary px-5"
          :disabled="status === 'loading'"
          @click="openCreateDialog"
        >
          <Plus class="size-4" />
          {{ t('application.createApplication') }}
        </button>
        <button class="app-button px-5" @click="triggerImport">
          <Upload class="size-4" />
          {{ t('application.import') }}
        </button>
        <input
          ref="fileInput"
          type="file"
          accept=".json"
          class="hidden"
          @change="handleFileImport"
        />
      </div>
    </ToolbarRoot>

    <!-- 加载 -->
    <div v-if="status === 'loading'" class="app-surface">
      <AppLoadingState />
    </div>
    <div v-else-if="status === 'error'" class="app-surface">
      <div class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('application.toast.loadFailed') }}</p>
      </div>
    </div>

    <div v-else class="app-surface">
      <AppEmptyState v-if="applications.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[960px]">
          <thead>
            <tr>
              <th>{{ t('common.name') }}</th>
              <th>{{ t('application.code') }}</th>
              <th>{{ t('application.kind') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th class="w-24">{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="app in applications" :key="app.id">
              <td>
                <router-link :to="`/application/${app.id}`" class="app-link">
                  {{ app.name }}
                </router-link>
              </td>
              <td class="text-foreground">{{ app.code }}</td>
              <td>
                <AppBadge variant="pill" :tone="applicationKindTone(app.kind)">
                  {{ app.kind }}
                </AppBadge>
              </td>
              <td class="text-foreground">{{ formatTime(app.created_at) }}</td>
              <td class="w-24">
                <button class="app-link" :disabled="operating" @click="openEditDialog(app)">
                  {{ t('common.edit') }}
                </button>
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
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <AppDialog v-model:open="isCreateDialogOpen" :title="t('application.createApplication')">
      <ApplicationFormFields
        :form="createForm"
        :errors="createErrors"
        @clear-error="clearCreateError"
        @update:form="Object.assign(createForm, $event)"
      />
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.create')"
          @cancel="isCreateDialogOpen = false"
          @confirm="handleCreateOk"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isEditDialogOpen"
      :title="t('application.detail.dialog.editApplication')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('common.name') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="editForm.name"
          type="text"
          class="app-input"
          :class="editErrors.name ? 'app-input-error' : ''"
          :aria-invalid="editErrors.name ? 'true' : undefined"
          @input="editErrors.name = ''"
        />
        <p v-if="editErrors.name" class="app-field-error mt-1 text-xs">{{ editErrors.name }}</p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('application.code') }}</label>
        <input v-model="editForm.code" type="text" disabled class="app-input" />
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="isEditDialogOpen = false"
          @confirm="handleEditOk"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isImportDialogOpen" :title="t('application.importApplication')">
      <ApplicationFormFields
        :form="importForm"
        :errors="importErrors"
        @clear-error="clearImportError"
        @update:form="Object.assign(importForm, $event)"
      />
      <div class="app-tip">
        {{ importSummary }}
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.import')"
          @cancel="isImportDialogOpen = false"
          @confirm="handleImportOk"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus, Upload } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import ApplicationFormFields from '@/components/ApplicationFormFields.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type {
    ApplicationCreateReq,
    ApplicationResp,
  } from '@/gen/proto/orbit/v1/application/application';
  import type { ApplicationImportReq } from '@/gen/proto/orbit/v1/application/application_bundle';
  import { applicationKindTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const applications = ref<ApplicationResp[]>([]);
  const searchText = ref('');
  const isCreateDialogOpen = ref(false);
  const isEditDialogOpen = ref(false);
  const isImportDialogOpen = ref(false);
  const fileInput = ref<HTMLInputElement>();
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const createForm = reactive<ApplicationCreateReq>({
    name: '',
    code: '',
    kind: 'standard',
  });
  const createErrors = reactive({ name: '', code: '' });
  const editingApplication = ref<ApplicationResp>();
  const editForm = reactive({ name: '', code: '' });
  const editErrors = reactive({ name: '' });
  const importForm = reactive<ApplicationImportReq>({
    name: '',
    code: '',
    kind: 'standard',
    version_label: 'v1',
    version_note: undefined,
    components: [],
  });
  const importErrors = reactive({ name: '', code: '' });
  const importSummary = computed(() =>
    t('application.importSummary', {
      versionLabel: importForm.version_label || '-',
      components: importForm.components.length,
    })
  );

  async function fetchApplications() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await applicationApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
          project_id: projectId,
        });
        applications.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('application.toast.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchApplications();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchApplications();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchApplications();
  }

  function validateCreateForm(errors: { name: string; code: string }) {
    errors.name = createForm.name.trim() ? '' : t('application.validation.nameRequired');
    errors.code = /^[a-z][a-z0-9-]*$/.test(createForm.code)
      ? ''
      : t('application.validation.codeInvalid');
    return !errors.name && !errors.code;
  }

  function validateImportForm(errors: { name: string; code: string }) {
    errors.name = importForm.name.trim() ? '' : t('application.validation.nameRequired');
    errors.code = /^[a-z][a-z0-9-]*$/.test(importForm.code)
      ? ''
      : t('application.validation.codeInvalid');
    return !errors.name && !errors.code;
  }

  function openCreateDialog() {
    Object.assign(createForm, {
      name: '',
      code: '',
      kind: 'standard',
    });
    Object.assign(createErrors, { name: '', code: '' });
    isCreateDialogOpen.value = true;
  }

  function openEditDialog(application: ApplicationResp) {
    editingApplication.value = application;
    Object.assign(editForm, { name: application.name, code: application.code });
    editErrors.name = '';
    isEditDialogOpen.value = true;
  }

  async function handleEditOk() {
    editErrors.name = editForm.name.trim() ? '' : t('application.validation.nameRequired');
    const applicationId = editingApplication.value?.id;
    if (editErrors.name || !applicationId) {
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.update(applicationId, { name: editForm.name.trim() });
        toast.success(t('application.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchApplications();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function clearCreateError(field: 'name' | 'code') {
    createErrors[field] = '';
  }

  function clearImportError(field: 'name' | 'code') {
    importErrors[field] = '';
  }

  async function handleCreateOk() {
    if (!validateCreateForm(createErrors)) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.create(
          {
            name: createForm.name,
            code: createForm.code,
            kind: createForm.kind,
          },
          { project_id: projectId }
        );
        toast.success(t('application.toast.createSuccess'));
        isCreateDialogOpen.value = false;
        await fetchApplications();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.createFailed'));
    }
  }

  function triggerImport() {
    fileInput.value?.click();
  }

  async function handleFileImport(event: Event) {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (!file) {
      return;
    }
    try {
      const data = JSON.parse(await file.text()) as ApplicationImportReq;
      if (!data.name || !data.code) {
        toast.error(t('application.validation.importMissingRequiredFields'));
        return;
      }
      Object.assign(importForm, data);
      Object.assign(importErrors, { name: '', code: '' });
      isImportDialogOpen.value = true;
    } catch {
      toast.error(t('application.toast.parseImportFailed'));
    } finally {
      target.value = '';
    }
  }

  async function handleImportOk() {
    if (!validateImportForm(importErrors)) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.importApplication(importForm, { project_id: projectId });
        toast.success(t('application.toast.importSuccess'));
        isImportDialogOpen.value = false;
        await fetchApplications();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.importFailed'));
    }
  }

  onMounted(fetchApplications);
</script>
