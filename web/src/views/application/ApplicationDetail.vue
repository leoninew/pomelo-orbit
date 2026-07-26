<template>
  <div class="flex flex-col gap-4">
    <div v-if="!versionsOnly" class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="flex flex-wrap items-center gap-2 text-xl font-semibold text-foreground">
        {{ application?.name || t('application.detail.title') }}
        <AppBadge v-if="application" variant="pill" :tone="applicationKindTone(application.kind)">
          {{ application.kind }}
        </AppBadge>
      </h1>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="application" class="app-button-primary h-9 px-3" @click="openEditModal">
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
        <button v-if="application" class="app-button h-9 px-3" @click="handleExport">
          <Download class="size-4" />
          {{ t('application.detail.actions.export') }}
        </button>
        <button
          v-if="application"
          :disabled="operating"
          class="app-button-danger h-9 px-3"
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/applications')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="basicInfoLoading" class="py-12" />

    <div v-else-if="application" class="flex flex-col gap-4">
      <!-- 基本信息 -->
      <div v-if="!versionsOnly" class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.basicInfo') }}
          </h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.name') }}
            </dt>
            <dd class="text-foreground">{{ application.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.code') }}
            </dt>
            <dd class="text-foreground">{{ application.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.kind') }}
            </dt>
            <dd>
              <AppBadge variant="pill" :tone="applicationKindTone(application.kind)">
                {{ application.kind }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.imagePullPolicy') }}
            </dt>
            <dd class="text-foreground">
              <AppBadge variant="pill">{{ application.image_pull_policy }}</AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.detail.fields.currentVersion') }}
            </dt>
            <dd class="text-foreground">
              {{ currentVersionLabel }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.detail.sections.versions') }}
            </dt>
            <dd>
              <router-link
                :to="`/versions?application_id=${application.id}`"
                class="text-primary hover:underline"
              >
                {{ t('nav.versions') }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('common.createdAt') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(application.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('common.updatedAt') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(application.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- 版本 -->
      <div v-if="versionsOnly" class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.versions') }}
          </h2>
          <button class="app-button-primary h-8 px-3" @click="openCreateVersionModal">
            <Plus class="size-4" />
            {{ t('application.detail.actions.createVersion') }}
          </button>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[880px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.versionLabel') }}</th>
                <th>{{ t('common.status') }}</th>
                <th>{{ t('application.detail.fields.components') }}</th>
                <th>{{ t('application.detail.fields.note') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="versionListLoading">
                <td colspan="6" class="text-center text-muted-foreground">
                  <AppSpinner />
                </td>
              </tr>
              <tr v-else-if="versions.length === 0">
                <td colspan="6" class="text-center text-muted-foreground">
                  {{ t('application.detail.empty.versions') }}
                </td>
              </tr>
              <tr v-for="version in versions" :key="version.id">
                <td class="text-foreground">
                  <button class="app-link font-medium" @click="goVersionDetail(version.id)">
                    {{ version.label }}
                  </button>
                  <span
                    v-if="application.version_id === version.id"
                    class="ml-2 text-xs text-muted-foreground"
                  >
                    {{ t('application.detail.fields.boundVersion') }}
                  </span>
                </td>
                <td>
                  <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
                    {{ version.status }}
                  </AppBadge>
                </td>
                <td
                  class="max-w-xs truncate text-muted-foreground"
                  :title="componentSummary(version)"
                >
                  {{ componentSummary(version) }}
                </td>
                <td class="max-w-xs truncate text-muted-foreground" :title="version.note || ''">
                  {{ version.note || '—' }}
                </td>
                <td class="text-muted-foreground">{{ formatTime(version.created_at) }}</td>
                <td>
                  <div class="flex flex-wrap items-center gap-3">
                    <button class="app-link" @click="goVersionDetail(version.id)">
                      {{ t('application.view') }}
                    </button>
                    <button class="app-link" @click="openPreview(version.id)">
                      {{ t('application.detail.actions.preview') }}
                    </button>
                    <button
                      v-if="version.status === 'unpublished'"
                      class="app-link"
                      :disabled="operating"
                      @click="handlePublish(version.id)"
                    >
                      {{ t('application.detail.actions.publish') }}
                    </button>
                    <button class="app-link" @click="openForkModal(version)">
                      {{ t('application.detail.actions.fork') }}
                    </button>
                    <button
                      v-if="version.status === 'unpublished'"
                      class="app-link-danger"
                      :disabled="operating"
                      @click="openDeleteVersionModal(version)"
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
      <ListPagination
        v-if="versionsOnly"
        :current="versionPagination.current"
        :page-size="versionPagination.pageSize"
        :total="versionPagination.total"
        :total-pages="versionTotalPages"
        @change-page="handleVersionPageChange"
        @change-page-size="handleVersionPageSizeChange"
      />
    </div>

    <!-- 编辑应用 -->
    <AppDialog
      v-if="!versionsOnly"
      v-model:open="isEditDialogOpen"
      :title="t('application.detail.dialog.editApplication')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.name') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="editForm.name"
          type="text"
          class="app-input"
          :class="editErrors.name ? 'app-input-error' : ''"
        />
        <p v-if="editErrors.name" class="app-field-error mt-1 text-xs">{{ editErrors.name }}</p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('application.code') }}</label>
        <input v-model="editForm.code" type="text" disabled class="app-input" />
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('application.imagePullPolicy') }}</label>
        <RawValueSelect v-model="editForm.image_pull_policy" :values="imagePullPolicyValues" />
      </div>
      <template #footer>
        <button class="app-button" @click="isEditDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleEditOk">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <!-- 删除应用 -->
    <AppDialog
      v-if="!versionsOnly"
      v-model:open="isDeleteDialogOpen"
      :title="t('application.detail.dialog.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="mb-4 text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.deleteApplicationConfirm', {
            name: application?.name || '-',
          })
        }}
      </p>
      <label class="flex items-center gap-2">
        <input v-model="deleteDir" type="checkbox" class="app-checkbox" />
        <span class="text-sm text-foreground">
          {{ t('application.detail.dialog.deleteWorkDir', { code: application?.code || '-' }) }}
        </span>
      </label>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-destructive" @click="handleDeleteOk">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>

    <!-- 删除未发布版本 -->
    <AppDialog
      v-if="versionsOnly"
      v-model:open="isDeleteVersionDialogOpen"
      :title="t('application.detail.dialog.deleteVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.deleteVersionConfirm', {
            label: pendingDeleteVersion?.label || '-',
          })
        }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteVersionDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-destructive" @click="handleDeleteVersionOk">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>

    <!-- 新建版本（仅基本信息） -->
    <AppDialog
      v-if="versionsOnly"
      v-model:open="isVersionDialogOpen"
      :title="t('application.detail.dialog.createVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.versionLabel') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="versionForm.label"
            type="text"
            class="app-input"
            :class="versionFormErrors.label ? 'app-input-error' : ''"
            :placeholder="t('application.detail.placeholders.versionLabel')"
          />
          <p v-if="versionFormErrors.label" class="app-field-error mt-1 text-xs">
            {{ versionFormErrors.label }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.note') }}
          </label>
          <input
            v-model="versionForm.note"
            type="text"
            class="app-input"
            :placeholder="t('application.detail.placeholders.note')"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isVersionDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleVersionCreate">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <!-- Fork 版本 -->
    <AppDialog
      v-if="versionsOnly"
      v-model:open="isForkDialogOpen"
      :title="t('application.detail.dialog.forkVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.versionLabel') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="forkLabel"
          type="text"
          class="app-input"
          :class="forkLabelError ? 'app-input-error' : ''"
          :placeholder="t('application.detail.placeholders.versionLabel')"
        />
        <p v-if="forkLabelError" class="app-field-error mt-1 text-xs">{{ forkLabelError }}</p>
      </div>
      <template #footer>
        <button class="app-button" @click="isForkDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleForkOk">
          {{ t('application.detail.actions.fork') }}
        </button>
      </template>
    </AppDialog>

    <!-- Compose 预览 -->
    <AppDrawer
      v-if="versionsOnly"
      :open="composePreviewDrawerOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="handleComposePreviewDrawerOpenChange"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <p class="text-sm text-muted-foreground">
          {{ t('application.detail.drawer.composePreviewDescription') }}
        </p>
        <div v-if="composePreviewLoading" class="flex flex-1 items-center justify-center">
          <AppSpinner />
        </div>
        <div
          v-else-if="composePreviewError"
          class="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {{ composePreviewError }}
        </div>
        <div v-else class="min-h-0 flex-1">
          <MonacoEditor
            :model-value="composePreviewYaml"
            language="yaml"
            height="100%"
            :readonly="true"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="composePreviewDrawerOpen = false">
          {{ t('application.detail.actions.close') }}
        </button>
      </template>
    </AppDrawer>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Download, Pencil, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { environmentApi } from '@/api/environment/environment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { applicationKindTone, versionStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  const {
    versionsOnly = false,
    applicationId: applicationIdProp = '',
    versionSearch = '',
  } = defineProps<{
    versionsOnly?: boolean;
    applicationId?: string;
    versionSearch?: string;
  }>();

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const applicationId = applicationIdProp || (route.params.id as string);
  const toast = useToast();

  const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: versionListLoading, execute: executeVersionList } = useStatusAsync();
  const { loading: composePreviewLoading, execute: executeComposePreview } = useStatusAsync();

  const application = ref<ApplicationResp>();
  const versions = ref<VersionResp[]>([]);
  const versionPagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const versionTotalPages = computed(() =>
    Math.ceil(versionPagination.total / versionPagination.pageSize)
  );
  const environments = ref<EnvironmentResp[]>([]);

  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeleteVersionDialogOpen = ref(false);
  const pendingDeleteVersion = ref<VersionResp | null>(null);
  const isVersionDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const composePreviewDrawerOpen = ref(false);
  const forkingVersionId = ref('');
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const deleteDir = ref(false);
  const composePreviewYaml = ref('');
  const composePreviewError = ref('');

  const editForm = reactive({
    name: '',
    code: '',
    image_pull_policy: 'missing',
  });
  const editErrors = reactive({ name: '' });
  const imagePullPolicyValues = ['missing', 'always', 'never'];

  const versionForm = reactive({
    label: '',
    note: '',
  });
  const versionFormErrors = reactive({ label: '' });

  const currentVersionLabel = computed(() => {
    const boundId = application.value?.version_id;
    if (!boundId) {
      return '—';
    }
    const found = versions.value.find((item) => item.id === boundId);
    return found?.label || boundId;
  });

  function componentSummary(version: VersionResp) {
    if (!version.components?.length) {
      return '—';
    }
    return version.components.map((c) => `${c.name}:${c.image}`).join(', ');
  }

  function goVersionDetail(versionId: string) {
    router.push(`/version/${versionId}`);
  }

  function defaultEnvironmentId() {
    const local = environments.value.find((item) => item.code === 'local');
    return local?.id || environments.value[0]?.id || '';
  }

  async function fetchApplication() {
    try {
      await executeBasicInfo(async () => {
        const data = await applicationApi.get(applicationId);
        application.value = data;
        Object.assign(editForm, {
          name: data.name,
          code: data.code,
          image_pull_policy: data.image_pull_policy,
        });
      });
    } catch {
      toast.error(t('application.toast.loadDetailFailed'));
      router.push('/applications');
    }
  }

  async function loadVersions() {
    try {
      await executeVersionList(async () => {
        const resp = await applicationApi.listVersions(applicationId, {
          page: versionsOnly ? versionPagination.current : 1,
          per_page: versionsOnly ? versionPagination.pageSize : 100,
          search: versionsOnly ? versionSearch : '',
        });
        versions.value = resp.items ?? [];
        if (versionsOnly) {
          versionPagination.total = resp.total;
          versionPagination.current = resp.page;
          versionPagination.pageSize = resp.per_page;
        }
      });
    } catch {
      toast.error(t('application.toast.loadVersionsFailed'));
    }
  }

  function handleVersionPageChange(page: number) {
    versionPagination.current = page;
    void loadVersions();
  }

  function handleVersionPageSizeChange(pageSize: number) {
    versionPagination.pageSize = pageSize;
    versionPagination.current = 1;
    void loadVersions();
  }

  async function loadEnvironments() {
    const projectId = application.value?.project_id;
    if (!projectId) {
      environments.value = [];
      return;
    }
    try {
      const resp = await environmentApi.list({ project_id: projectId, per_page: 100 });
      environments.value = resp.items ?? [];
    } catch {
      toast.error(t('application.toast.loadEnvironmentsFailed'));
    }
  }

  async function handleExport() {
    try {
      const data = await applicationApi.exportApplication(applicationId);
      const blob = new Blob([JSON.stringify(data, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${data.code || 'application'}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast.success(t('application.toast.exportSuccess'));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.exportFailed'));
    }
  }

  function openEditModal() {
    editErrors.name = '';
    if (application.value) {
      Object.assign(editForm, {
        name: application.value.name,
        code: application.value.code,
        image_pull_policy: application.value.image_pull_policy,
      });
    }
    isEditDialogOpen.value = true;
  }

  async function handleEditOk() {
    editErrors.name = editForm.name.trim() ? '' : t('application.validation.nameRequired');
    if (editErrors.name) {
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.update(applicationId, {
          name: editForm.name,
          image_pull_policy: editForm.image_pull_policy,
        });
        toast.success(t('application.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchApplication();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openDeleteModal() {
    deleteDir.value = false;
    isDeleteDialogOpen.value = true;
  }

  async function handleDeleteOk() {
    try {
      await executeOp(async () => {
        await applicationApi.delete(applicationId, deleteDir.value);
        toast.success(t('application.toast.deleteSuccess'));
        router.push('/applications');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.deleteFailed'));
    }
  }

  function openCreateVersionModal() {
    versionForm.label = suggestNextLabel();
    versionForm.note = '';
    versionFormErrors.label = '';
    isVersionDialogOpen.value = true;
  }

  function suggestNextLabel() {
    const labels = versions.value.map((v) => v.label);
    let n = versions.value.length + 1;
    while (labels.includes(`v${n}`)) {
      n += 1;
    }
    return `v${n}`;
  }

  async function handleVersionCreate() {
    versionFormErrors.label = versionForm.label.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (versionFormErrors.label) {
      return;
    }
    const note = versionForm.note.trim() || undefined;
    try {
      await executeOp(async () => {
        const created = await applicationApi.createVersion(applicationId, {
          application_id: applicationId,
          label: versionForm.label.trim(),
          note,
          components: [],
          exposes: [],
        });
        toast.success(t('application.toast.createVersionSuccess'));
        isVersionDialogOpen.value = false;
        router.push(`/version/${created.id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.saveFailed'));
    }
  }

  async function handlePublish(versionId: string) {
    try {
      await executeOp(async () => {
        await applicationApi.publishVersion(versionId);
        toast.success(t('application.toast.publishSuccess'));
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.publishFailed'));
    }
  }

  function openDeleteVersionModal(version: VersionResp) {
    pendingDeleteVersion.value = version;
    isDeleteVersionDialogOpen.value = true;
  }

  async function handleDeleteVersionOk() {
    const target = pendingDeleteVersion.value;
    if (!target) {
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersion(target.id);
        toast.success(t('application.toast.deleteVersionSuccess'));
        isDeleteVersionDialogOpen.value = false;
        pendingDeleteVersion.value = null;
        await loadVersions();
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.deleteVersionFailed')
      );
    }
  }

  function openForkModal(version: VersionResp) {
    forkingVersionId.value = version.id;
    forkLabel.value = `${version.label}-fork`;
    forkLabelError.value = '';
    isForkDialogOpen.value = true;
  }

  async function handleForkOk() {
    forkLabelError.value = forkLabel.value.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (forkLabelError.value || !forkingVersionId.value) {
      return;
    }
    try {
      await executeOp(async () => {
        const created = await applicationApi.forkVersion(forkingVersionId.value, {
          label: forkLabel.value.trim(),
        });
        toast.success(t('application.toast.forkSuccess'));
        isForkDialogOpen.value = false;
        router.push(`/version/${created.id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.forkFailed'));
    }
  }

  async function openPreview(versionId: string) {
    if (environments.value.length === 0) {
      await loadEnvironments();
    }
    const environmentId = defaultEnvironmentId();
    if (!environmentId) {
      toast.error(t('application.toast.environmentRequired'));
      return;
    }
    composePreviewYaml.value = '';
    composePreviewError.value = '';
    composePreviewDrawerOpen.value = true;
    try {
      await executeComposePreview(async () => {
        const { compose_yaml } = await applicationApi.previewVersion(versionId, {
          environment_id: environmentId,
          instance_key: 'default',
        });
        composePreviewYaml.value = compose_yaml;
      });
    } catch (error) {
      composePreviewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    }
  }

  function handleComposePreviewDrawerOpenChange(open: boolean) {
    composePreviewDrawerOpen.value = open;
    if (!open) {
      composePreviewYaml.value = '';
      composePreviewError.value = '';
    }
  }

  onMounted(async () => {
    await fetchApplication();
    await loadVersions();
  });
</script>
