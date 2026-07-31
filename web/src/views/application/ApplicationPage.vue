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
      <AppSpinner class="py-16" />
    </div>

    <div v-else class="app-surface">
      <AppEmptyState v-if="applications.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1040px]">
          <thead>
            <tr>
              <th>{{ t('common.name') }}</th>
              <th>{{ t('application.code') }}</th>
              <th>{{ t('application.kind') }}</th>
              <th>{{ t('application.imagePullPolicy') }}</th>
              <th>{{ t('common.createdAt') }}</th>
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
              <td>
                <AppBadge variant="pill">{{ app.image_pull_policy }}</AppBadge>
              </td>
              <td class="text-foreground">{{ formatTime(app.created_at) }}</td>
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
  import AppSpinner from '@/components/AppSpinner.vue';
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
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const applications = ref<ApplicationResp[]>([]);
  const searchText = ref('');
  const isCreateDialogOpen = ref(false);
  const isImportDialogOpen = ref(false);
  const fileInput = ref<HTMLInputElement>();
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const createForm = reactive<ApplicationCreateReq>({
    name: '',
    code: '',
    kind: 'standard',
    image_pull_policy: 'missing',
  });
  const createErrors = reactive({ name: '', code: '' });
  const importForm = reactive<ApplicationImportReq>({
    name: '',
    code: '',
    kind: 'standard',
    image_pull_policy: 'missing',
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
      image_pull_policy: 'missing',
    });
    Object.assign(createErrors, { name: '', code: '' });
    isCreateDialogOpen.value = true;
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
            image_pull_policy: createForm.image_pull_policy,
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
