<template>
  <div class="flex flex-col gap-4">
    <div v-if="!versionsOnly" class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ application?.name || t('application.detail.title') }}
        </h1>
        <DetailHeaderMeta v-if="application">
          <AppBadge :tone="applicationKindTone(application.kind)">{{ application.kind }}</AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
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
      <div v-if="!versionsOnly" class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('application.detail.sections.basicInfo') }}
          </h2>
          <button class="app-button-primary h-9 px-3" @click="openEditModal">
            <Pencil class="size-4" />
            {{ t('common.edit') }}
          </button>
        </div>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">
              {{ t('application.name') }}
            </dt>
            <dd class="text-foreground">{{ application.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">
              {{ t('application.code') }}
            </dt>
            <dd class="text-foreground">{{ application.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">
              {{ t('application.kind') }}
            </dt>
            <dd>
              <AppBadge variant="pill" :tone="applicationKindTone(application.kind)">
                {{ application.kind }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">
              {{ t('application.imagePullPolicy') }}
            </dt>
            <dd class="text-foreground">
              <AppBadge variant="pill">{{ application.image_pull_policy }}</AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">
              {{ t('common.createdAt') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(application.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">
              {{ t('common.updatedAt') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(application.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- 版本 -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('application.detail.sections.versions') }}
          </h2>
          <button class="app-button-primary h-9 px-3" @click="openCreateVersionModal">
            <Plus class="size-4" />
            {{ t('application.detail.actions.createVersion') }}
          </button>
        </div>
        <AppSpinner v-if="versionListLoading" class="py-8" />
        <AppEmptyState v-else-if="versions.length === 0" size="compact" />
        <AccordionRoot
          v-else-if="!versionsOnly"
          :default-value="versions[0] ? [versions[0].id] : []"
          type="multiple"
          class="divide-y divide-border"
        >
          <AccordionItem v-for="version in versions" :key="version.id" :value="version.id">
            <div class="flex flex-wrap items-center gap-x-4 gap-y-2 px-5 py-4">
              <AccordionHeader class="min-w-0 flex-1">
                <AccordionTrigger
                  class="group flex w-full items-center justify-between gap-3 text-left hover:text-foreground"
                  @click="loadVersionComponents(version.id)"
                >
                  <ChevronDown
                    class="size-4 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-180"
                  />
                  <span
                    class="grid min-w-0 flex-1 gap-x-6 gap-y-1 sm:grid-cols-[minmax(12rem,1fr)_minmax(12rem,1fr)_minmax(10rem,0.8fr)] sm:items-center"
                  >
                    <span class="flex min-w-0 items-center gap-2">
                      <span class="break-words text-sm text-foreground">
                        {{ version.label }}
                      </span>
                      <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
                        {{ version.status }}
                      </AppBadge>
                    </span>
                    <span
                      v-if="version.component_summary"
                      class="break-words text-sm text-muted-foreground"
                    >
                      {{ version.component_summary }}
                    </span>
                    <span v-if="version.note" class="break-words text-sm text-muted-foreground">
                      {{ version.note }}
                    </span>
                  </span>
                </AccordionTrigger>
              </AccordionHeader>
              <div class="flex shrink-0 flex-wrap items-center gap-3 text-sm">
                <router-link :to="`/version/${version.id}`" class="app-link">
                  {{ t('application.view') }}
                </router-link>
                <button class="app-link" @click="openForkModal(version)">
                  {{ t('application.detail.actions.fork') }}
                </button>
              </div>
            </div>
            <AccordionContent class="border-t border-border px-5 py-4">
              <AppSpinner v-if="versionComponentLoadState[version.id] === 'loading'" class="py-4" />
              <p
                v-else-if="versionComponentLoadState[version.id] === 'error'"
                class="py-2 text-sm text-destructive"
              >
                {{ t('application.versionDetail.componentsLoadFailed') }}
              </p>
              <AppEmptyState
                v-else-if="versionComponents[version.id]?.length === 0"
                size="compact"
              />
              <div v-else-if="versionComponents[version.id]" class="overflow-x-auto">
                <table class="app-data-table min-w-[840px]">
                  <thead>
                    <tr>
                      <th>{{ t('application.detail.fields.component') }}</th>
                      <th>{{ t('application.detail.fields.image') }}</th>
                      <th>{{ t('application.componentDetail.fields.pullPolicy') }}</th>
                      <th>{{ t('application.componentDetail.fields.restartPolicy') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="component in versionComponents[version.id]" :key="component.id">
                      <td>
                        <router-link
                          :to="`/version/${version.id}/component/${component.id}`"
                          class="app-link"
                        >
                          {{ component.name }}
                        </router-link>
                      </td>
                      <td class="max-w-md whitespace-normal break-all text-muted-foreground">
                        {{ component.image }}
                      </td>
                      <td class="text-muted-foreground">{{ component.pull_policy }}</td>
                      <td class="text-muted-foreground">{{ component.restart_policy }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </AccordionContent>
          </AccordionItem>
        </AccordionRoot>
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[880px]">
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
              <tr v-for="version in versions" :key="version.id">
                <td class="text-foreground">
                  <router-link :to="`/version/${version.id}`" class="app-link">
                    {{ version.label }}
                  </router-link>
                </td>
                <td>
                  <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
                    {{ version.status }}
                  </AppBadge>
                </td>
                <td
                  class="max-w-xs truncate text-muted-foreground"
                  :title="version.component_summary || ''"
                >
                  {{ version.component_summary }}
                </td>
                <td class="max-w-xs truncate text-muted-foreground" :title="version.note || ''">
                  {{ version.note }}
                </td>
                <td class="text-muted-foreground">{{ formatTime(version.created_at) }}</td>
                <td>
                  <div class="flex flex-wrap items-center gap-3">
                    <button
                      v-if="versionsOnly && version.status === 'unpublished'"
                      class="app-link"
                      :disabled="operating"
                      @click="handlePublish(version.id)"
                    >
                      {{ t('application.detail.actions.publish') }}
                    </button>
                    <button
                      v-else-if="versionsOnly && version.status === 'published'"
                      class="app-link"
                      :disabled="operating"
                      @click="handleUnpublish(version.id)"
                    >
                      {{ t('application.detail.actions.unpublish') }}
                    </button>
                    <button class="app-link" @click="openForkModal(version)">
                      {{ t('application.detail.actions.fork') }}
                    </button>
                    <button
                      v-if="versionsOnly && version.status === 'unpublished'"
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
          :aria-invalid="editErrors.name ? 'true' : undefined"
          @input="editErrors.name = ''"
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
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="isEditDialogOpen = false"
          @confirm="handleEditOk"
        />
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
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="handleDeleteOk"
        />
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
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="isDeleteVersionDialogOpen = false"
          @confirm="handleDeleteVersionOk"
        />
      </template>
    </AppDialog>

    <!-- 创建版本（仅基本信息） -->
    <AppDialog
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
            :aria-invalid="versionFormErrors.label ? 'true' : undefined"
            @input="versionFormErrors.label = ''"
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
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.create')"
          @cancel="isVersionDialogOpen = false"
          @confirm="handleVersionCreate"
        />
      </template>
    </AppDialog>

    <!-- Fork 版本 -->
    <AppDialog
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
          :aria-invalid="forkLabelError ? 'true' : undefined"
          @input="forkLabelError = ''"
        />
        <p v-if="forkLabelError" class="app-field-error mt-1 text-xs">{{ forkLabelError }}</p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.copy')"
          @cancel="isForkDialogOpen = false"
          @confirm="handleForkOk"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, ChevronDown, Download, Pencil, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import {
    AccordionContent,
    AccordionHeader,
    AccordionItem,
    AccordionRoot,
    AccordionTrigger,
  } from 'reka-ui';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { VersionComponentResp, VersionResp } from '@/gen/proto/orbit/v1/application/version';
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

  const application = ref<ApplicationResp>();
  const versions = ref<VersionResp[]>([]);
  const versionComponents = reactive<Record<string, VersionComponentResp[]>>({});
  const versionComponentLoadState = reactive<Record<string, 'loading' | 'loaded' | 'error'>>({});
  const versionPagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const versionTotalPages = computed(() =>
    Math.ceil(versionPagination.total / versionPagination.pageSize)
  );

  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeleteVersionDialogOpen = ref(false);
  const pendingDeleteVersion = ref<VersionResp | null>(null);
  const isVersionDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const forkingVersionId = ref('');
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const deleteDir = ref(false);

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
      if (!versionsOnly && versions.value[0]) {
        void loadVersionComponents(versions.value[0].id);
      }
    } catch {
      toast.error(t('application.toast.loadVersionsFailed'));
    }
  }

  async function loadVersionComponents(versionId: string) {
    const state = versionComponentLoadState[versionId];
    if (state === 'loading' || state === 'loaded') {
      return;
    }
    versionComponentLoadState[versionId] = 'loading';
    try {
      const version = await applicationApi.getVersion(versionId);
      versionComponents[versionId] = version.components ?? [];
      versionComponentLoadState[versionId] = 'loaded';
    } catch {
      versionComponentLoadState[versionId] = 'error';
      toast.error(t('application.versionDetail.componentsLoadFailed'));
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
    versionForm.label = '';
    versionForm.note = '';
    versionFormErrors.label = '';
    isVersionDialogOpen.value = true;
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

  async function handleUnpublish(versionId: string) {
    try {
      await executeOp(async () => {
        await applicationApi.unpublishVersion(versionId);
        toast.success(t('application.toast.unpublishSuccess'));
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.unpublishFailed'));
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
    forkLabel.value = `${version.label}-copy`;
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

  onMounted(async () => {
    await fetchApplication();
    if (application.value) {
      await loadVersions();
    }
  });
</script>
