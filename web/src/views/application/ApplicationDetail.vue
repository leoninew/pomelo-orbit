<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader
        v-if="!versionsOnly"
        :items="[]"
        :title="application?.name || t('application.detail.title')"
      />
      <div class="flex min-w-0 flex-wrap items-center gap-2" :class="versionsOnly ? 'ml-auto' : ''">
        <SearchControl
          v-model="versionSearchText"
          :placeholder="t('application.detail.searchVersionsPlaceholder')"
          :loading="versionListLoading"
          class="w-72 min-w-0 max-w-full"
          @search="handleVersionSearch"
        />
        <button
          class="app-button-primary h-9 px-3"
          :disabled="!application || operating"
          @click="openCreateVersionModal"
        >
          <Plus class="size-4" />
          {{ t('application.detail.dialog.createVersion') }}
        </button>
        <template v-if="!versionsOnly">
          <button
            v-if="application"
            class="app-button h-9 px-3"
            :disabled="operating"
            @click="openEditModal"
          >
            <Pencil class="size-4" />
            {{ t('application.detail.dialog.editApplication') }}
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
        </template>
      </div>
    </div>

    <AppLoadingState v-if="basicInfoLoading" size="section" />

    <div v-else-if="application" class="flex flex-col gap-4">
      <div class="min-w-0">
        <AppLoadingState v-if="versionListLoading" size="section" />
        <div v-else-if="versionListError" class="py-8 text-center text-sm">
          <p class="text-destructive">{{ t('application.toast.loadVersionsFailed') }}</p>
          <button class="app-button mt-3" @click="loadVersions()">{{ t('common.retry') }}</button>
        </div>
        <div
          v-else
          class="grid min-h-[420px] grid-cols-1 lg:grid-cols-[minmax(240px,300px)_minmax(0,1fr)]"
        >
          <div class="min-w-0 border-b border-border lg:border-b-0 lg:border-r lg:pr-3">
            <div class="max-h-80 overflow-y-auto lg:max-h-[680px]">
              <AppEmptyState v-if="versions.length === 0" size="compact" />
              <ApplicationVersionTree
                v-else
                :versions="versions"
                :details="versionDetails"
                :load-states="versionDetailLoadState"
                :expanded-id="expandedVersionId"
                :selection="selection"
                :busy="operating"
                @create-component="handleNodeAction('create-component', { versionId: $event })"
                @select="selectNode"
                @toggle="toggleVersionComponents"
                @retry="loadVersionDetails"
              />
            </div>
          </div>
          <div class="min-w-0 py-4 lg:py-0 lg:pl-5">
            <AppLoadingState v-if="selectedDetailState === 'loading'" size="section" />
            <div v-else-if="selectedDetailState === 'error'" class="py-8 text-center text-sm">
              <p class="text-destructive">
                {{ t('application.detail.explorer.loadDetailsFailed') }}
              </p>
              <button
                v-if="selection"
                class="app-button mt-3"
                @click="loadVersionDetails(selection.versionId)"
              >
                {{ t('common.retry') }}
              </button>
            </div>
            <ApplicationNodeDetails
              v-else-if="selectedVersion"
              :version="selectedVersion"
              :component="selectedComponent"
              :busy="operating"
              @action="selection && handleNodeAction($event, selection)"
            />
            <DetailInfoCard
              v-else-if="!selection"
              :title="t('application.detail.sections.basicInfo')"
              :editable="!versionsOnly"
              :disabled="operating"
              @edit="openEditModal"
            >
              <dl class="app-detail-info-grid">
                <div class="flex gap-2">
                  <dt class="whitespace-nowrap">{{ t('common.name') }}</dt>
                  <dd class="text-foreground">{{ application.name }}</dd>
                </div>
                <div class="flex gap-2">
                  <dt class="whitespace-nowrap">{{ t('application.code') }}</dt>
                  <dd class="text-foreground text-sm">{{ application.code }}</dd>
                </div>
                <div class="flex gap-2">
                  <dt class="whitespace-nowrap">{{ t('application.kind') }}</dt>
                  <dd>
                    <AppBadge variant="pill" :tone="applicationKindTone(application.kind)">
                      {{ application.kind }}
                    </AppBadge>
                  </dd>
                </div>
                <div class="flex gap-2">
                  <dt class="whitespace-nowrap">{{ t('common.createdAt') }}</dt>
                  <dd class="text-muted-foreground">{{ formatTime(application.created_at) }}</dd>
                </div>
                <div class="flex gap-2">
                  <dt class="whitespace-nowrap">{{ t('common.updatedAt') }}</dt>
                  <dd class="text-muted-foreground">{{ formatTime(application.updated_at) }}</dd>
                </div>
              </dl>
            </DetailInfoCard>
            <AppEmptyState
              v-else
              size="compact"
              :message="t('application.detail.explorer.selectNode')"
            />
          </div>
        </div>
        <ListPagination
          class="!border-t-0"
          :current="versionPagination.current"
          :page-size="versionPagination.pageSize"
          :total="versionPagination.total"
          :total-pages="versionTotalPages"
          standalone
          @change-page="handleVersionPageChange"
          @change-page-size="handleVersionPageSizeChange"
        />
      </div>
    </div>

    <!-- 编辑应用 -->
    <AppDialog
      v-if="!versionsOnly"
      v-model:open="isEditDialogOpen"
      :title="t('application.detail.dialog.editApplication')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('common.name') }}
          <span class="text-destructive">*</span>
        </label>
        <AppInput
          v-model="editForm.name"
          type="text"
          :class="editErrors.name ? 'app-input-error' : ''"
          :aria-invalid="editErrors.name ? 'true' : undefined"
          @input="editErrors.name = ''"
        />
        <p v-if="editErrors.name" class="app-field-error mt-1">{{ editErrors.name }}</p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('application.code') }}</label>
        <AppInput v-model="editForm.code" type="text" disabled />
      </div>
      <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
        {{ editSubmitError }}
      </p>
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
      <p v-if="deleteApplicationError" class="app-field-error mt-3" role="alert">
        {{ deleteApplicationError }}
      </p>
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

    <!-- 删除版本 -->
    <AppDialog
      :open="isDeleteVersionDialogOpen"
      :title="t('application.detail.dialog.deleteVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
      @update:open="setDeleteVersionDialogOpen"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.deleteVersionConfirm', {
            label: pendingDeleteVersion?.label || '-',
          })
        }}
      </p>
      <p v-if="deleteVersionError" class="app-field-error" role="alert">
        {{ deleteVersionError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="setDeleteVersionDialogOpen(false)"
          @confirm="handleDeleteVersionOk"
        />
      </template>
    </AppDialog>

    <!-- 编辑版本组件基本信息 -->
    <AppDialog
      :open="isComponentEditDialogOpen"
      :title="
        pendingComponent
          ? t('application.versionDetail.dialog.editComponent')
          : t('application.versionDetail.dialog.addComponent')
      "
      width-class="w-[min(640px,calc(100vw-32px))]"
      @update:open="setComponentEditDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.component') }}
            <span class="text-destructive">*</span>
          </label>
          <AppInput
            v-model="componentForm.name"
            :class="componentErrors.name ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="componentErrors.name ? 'true' : undefined"
            @input="componentErrors.name = ''"
          />
          <p v-if="componentErrors.name" class="app-field-error" role="alert">
            {{ componentErrors.name }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.image') }}
            <span class="text-destructive">*</span>
          </label>
          <AppInput
            v-model="componentForm.image"
            :class="componentErrors.image ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="componentErrors.image ? 'true' : undefined"
            @input="componentErrors.image = ''"
          />
          <p v-if="componentErrors.image" class="app-field-error" role="alert">
            {{ componentErrors.image }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.pullPolicy') }}
          </label>
          <SelectControl
            v-model="componentForm.pull_policy"
            :placeholder="t('common.notSet')"
            :options="componentPullPolicyValues"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.restartPolicy') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="componentForm.restart_policy"
            :invalid="Boolean(componentErrors.restartPolicy)"
            :options="componentRestartPolicyValues"
            @update:model-value="componentErrors.restartPolicy = ''"
          />
          <p v-if="componentErrors.restartPolicy" class="app-field-error" role="alert">
            {{ componentErrors.restartPolicy }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.containerEntrypoint') }}
          </label>
          <AppTextarea v-model="componentForm.entrypoint" class="app-code-content" rows="3" />
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.command') }}
          </label>
          <AppTextarea v-model="componentForm.command" class="app-code-content" rows="3" />
        </div>
      </div>
      <p v-if="componentEditError" class="app-field-error mt-3" role="alert">
        {{ componentEditError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="cancelComponentEditing"
          @confirm="saveComponentBasic"
        />
      </template>
    </AppDialog>

    <AppDialog
      :open="isComponentIdentityDialogOpen"
      :title="t('application.componentDetail.sections.identity')"
      width-class="w-[min(480px,calc(100vw-32px))]"
      @update:open="setComponentIdentityDialogOpen"
    >
      <ComponentIdentityFields
        v-model:user="componentForm.user"
        v-model:groups="componentForm.group_add"
      />
      <p v-if="componentIdentityError" class="app-field-error" role="alert">
        {{ componentIdentityError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="setComponentIdentityDialogOpen(false)"
          @confirm="saveComponentIdentity"
        />
      </template>
    </AppDialog>

    <!-- 删除版本组件 -->
    <AppDialog
      v-model:open="isComponentDeleteDialogOpen"
      :title="t('application.componentDetail.actions.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.componentDetail.deleteDescription', {
            name: pendingComponent?.name || '-',
          })
        }}
      </p>
      <p v-if="componentDeleteError" class="app-field-error mt-3" role="alert">
        {{ componentDeleteError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="cancelComponentDeletion"
          @confirm="deleteComponent"
        />
      </template>
    </AppDialog>

    <!-- 创建版本（仅基本信息） -->
    <AppDialog
      v-model:open="isVersionDialogOpen"
      :title="
        editingVersionId
          ? t('application.versionDetail.dialog.editBasic')
          : t('application.detail.dialog.createVersion')
      "
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.versionLabel') }}
            <span class="text-destructive">*</span>
          </label>
          <AppInput
            v-model="versionForm.label"
            type="text"
            :class="versionFormErrors.label ? 'app-input-error' : ''"
            :placeholder="t('application.detail.placeholders.versionLabel')"
            :aria-invalid="versionFormErrors.label ? 'true' : undefined"
            @input="versionFormErrors.label = ''"
          />
          <p v-if="versionFormErrors.label" class="app-field-error mt-1">
            {{ versionFormErrors.label }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.note') }}
          </label>
          <AppInput
            v-model="versionForm.note"
            type="text"
            :placeholder="t('application.detail.placeholders.note')"
          />
        </div>
      </div>
      <p v-if="versionSubmitError" class="app-field-error mt-3" role="alert">
        {{ versionSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="editingVersionId ? t('common.save') : t('common.create')"
          @cancel="isVersionDialogOpen = false"
          @confirm="saveVersion"
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
        <AppInput
          v-model="forkLabel"
          type="text"
          :class="forkLabelError ? 'app-input-error' : ''"
          :placeholder="t('application.detail.placeholders.versionLabel')"
          :aria-invalid="forkLabelError ? 'true' : undefined"
          @input="forkLabelError = ''"
        />
        <p v-if="forkLabelError" class="app-field-error mt-1">{{ forkLabelError }}</p>
      </div>
      <p v-if="forkSubmitError" class="app-field-error mt-3" role="alert">
        {{ forkSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.copy')"
          @cancel="isForkDialogOpen = false"
          @confirm="handleForkOk"
        />
      </template>
    </AppDialog>

    <AppDrawer
      :open="previewOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="setPreviewOpen"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <AppLoadingState v-if="previewLoading" class="flex-1 items-center" size="section" />
        <div
          v-else-if="previewError"
          class="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {{ previewError }}
        </div>
        <div v-else class="min-h-0 flex-1">
          <MonacoEditor
            :model-value="previewContent"
            language="yaml"
            height="100%"
            :readonly="true"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="setPreviewOpen(false)">
          {{ t('application.detail.actions.close') }}
        </button>
      </template>
    </AppDrawer>
  </div>
</template>

<script setup lang="ts">
  import AppInput from '@/components/ui/AppInput.vue';
  import AppTextarea from '@/components/ui/AppTextarea.vue';
  import { ArrowLeft, Pencil, Plus, Trash2 } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/ui/AppBadge.vue';
  import DetailPageHeader from '@/components/ui/DetailPageHeader.vue';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppDrawer from '@/components/ui/AppDrawer.vue';
  import AppEmptyState from '@/components/ui/AppEmptyState.vue';
  import AppLoadingState from '@/components/ui/AppLoadingState.vue';
  import ListPagination from '@/components/ui/ListPagination.vue';
  import MonacoEditor from '@/components/ui/MonacoEditor.vue';
  import SelectControl from '@/components/ui/SelectControl.vue';
  import SearchControl from '@/components/ui/SearchControl.vue';
  import ComponentIdentityFields from '@/components/application/ComponentIdentityFields.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { VersionComponentResp, VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { useProjectStore } from '@/stores/project';
  import { applicationKindTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';
  import DetailInfoCard from '@/components/ui/DetailInfoCard.vue';
  import ApplicationVersionTree from './components/ApplicationVersionTree.vue';
  import ApplicationNodeDetails from './components/ApplicationNodeDetails.vue';
  import {
    resolveVersionTreeSelection,
    type VersionTreeSelection,
    type VersionTreeLoadState,
  } from './versionTree';
  import {
    componentBasicRequestFromForm,
    componentIdentityRequestFromForm,
    componentFormFromResponse,
    emptyComponentForm,
  } from './componentForm';

  const { versionsOnly = false, applicationId: applicationIdProp = '' } = defineProps<{
    versionsOnly?: boolean;
    applicationId?: string;
  }>();

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const applicationId = applicationIdProp || (route.params.id as string);
  const toast = useToast();
  const projectStore = useProjectStore();

  const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();
  const { loading: versionListLoading, execute: executeVersionList } = useStatusAsync();

  const application = ref<ApplicationResp>();
  const versions = ref<VersionResp[]>([]);
  const versionSearchText = ref('');
  const expandedVersionId = ref('');
  const versionDetails = reactive<Record<string, VersionResp>>({});
  const versionDetailLoadState = reactive<Record<string, VersionTreeLoadState>>({});
  const selection = ref<VersionTreeSelection>();
  const versionListError = ref(false);
  const selectedVersion = computed(() =>
    selection.value ? versionDetails[selection.value.versionId] : undefined
  );
  const selectedComponent = computed(() =>
    selectedVersion.value?.components.find(
      (component) => component.id === selection.value?.componentId
    )
  );
  const selectedDetailState = computed(() =>
    selection.value ? versionDetailLoadState[selection.value.versionId] : undefined
  );
  const versionPagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const versionTotalPages = computed(() =>
    Math.ceil(versionPagination.total / versionPagination.pageSize)
  );

  const previewOpen = ref(false);
  const previewContent = ref('');
  const previewError = ref('');

  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeleteVersionDialogOpen = ref(false);
  const pendingDeleteVersion = ref<VersionResp | null>(null);
  const deleteVersionError = ref('');
  const editSubmitError = ref('');
  const deleteApplicationError = ref('');
  const versionSubmitError = ref('');
  const editingVersionId = ref('');
  const componentEditError = ref('');
  const componentIdentityError = ref('');
  const componentDeleteError = ref('');
  const isVersionDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const isComponentEditDialogOpen = ref(false);
  const isComponentIdentityDialogOpen = ref(false);
  const isComponentDeleteDialogOpen = ref(false);
  const pendingComponentVersionId = ref('');
  const pendingComponent = ref<VersionComponentResp>();
  const forkingVersionId = ref('');
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const forkSubmitError = ref('');

  const editForm = reactive({
    name: '',
    code: '',
  });
  const editErrors = reactive({ name: '' });

  const versionForm = reactive({
    label: '',
    note: '',
  });
  const versionFormErrors = reactive({ label: '' });
  const componentForm = reactive(emptyComponentForm());
  const componentErrors = reactive({ name: '', image: '', restartPolicy: '' });
  const componentPullPolicyValues = ['always', 'missing', 'never'];
  const componentRestartPolicyValues = ['no', 'on-failure', 'always', 'unless-stopped'];

  async function fetchApplication() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeBasicInfo(async () => {
        const data = await applicationApi.get(projectId, applicationId);
        if (projectStore.activeProjectId !== projectId) {
          return;
        }
        application.value = data;
        Object.assign(editForm, {
          name: data.name,
          code: data.code,
        });
      });
    } catch {
      toast.error(t('application.toast.loadDetailFailed'));
      router.push('/applications');
    }
  }

  async function loadVersions(preferredSelection = selection.value) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    versionListError.value = false;
    try {
      await executeVersionList(async () => {
        const resp = await applicationApi.listVersions(projectId, applicationId, {
          page: versionPagination.current,
          per_page: versionPagination.pageSize,
          search: versionSearchText.value.trim() || undefined,
        });
        if (projectStore.activeProjectId !== projectId) {
          return;
        }
        versionPagination.total = resp.total;
        versionPagination.current = resp.page;
        versionPagination.pageSize = resp.per_page;
        if (resp.total > 0 && resp.items.length === 0 && resp.page > 1) {
          versionPagination.current = Math.max(1, resp.pages);
          await loadVersions(preferredSelection);
          return;
        }
        versions.value = resp.items;
        selection.value = resolveVersionTreeSelection(resp.items, preferredSelection);
        if (!resp.items.some((version) => version.id === expandedVersionId.value)) {
          expandedVersionId.value = '';
        }
      });
      if (selection.value && !versionListError.value) {
        expandVersion(selection.value.versionId);
        void loadVersionDetails(selection.value.versionId);
      }
    } catch {
      versionListError.value = true;
      toast.error(t('application.toast.loadVersionsFailed'));
    }
  }

  async function loadVersionDetails(versionId: string) {
    const state = versionDetailLoadState[versionId];
    if (state === 'loading' || state === 'loaded') {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    versionDetailLoadState[versionId] = 'loading';
    try {
      const version = await applicationApi.getVersion(projectId, versionId);
      if (projectStore.activeProjectId !== projectId) {
        return;
      }
      versionDetails[versionId] = version;
      versionDetailLoadState[versionId] = 'loaded';
    } catch {
      versionDetailLoadState[versionId] = 'error';
    }
  }

  function expandVersion(versionId: string) {
    expandedVersionId.value = versionId;
  }

  function selectNode(node: VersionTreeSelection) {
    selection.value = node;
    expandVersion(node.versionId);
    void loadVersionDetails(node.versionId);
  }

  function toggleVersionComponents(versionId: string) {
    if (expandedVersionId.value === versionId) {
      expandedVersionId.value = '';
      if (selection.value?.versionId === versionId && selection.value.componentId) {
        selection.value = { versionId };
      }
      return;
    }
    selectNode({ versionId });
  }

  function updateVersion(version: VersionResp) {
    versionDetails[version.id] = version;
    versionDetailLoadState[version.id] = 'loaded';
    versions.value = versions.value.map((item) => (item.id === version.id ? version : item));
  }

  function handleNodeAction(action: string, node: VersionTreeSelection) {
    if (operating.value) {
      return;
    }
    const version = versions.value.find((item) => item.id === node.versionId);
    if (!version) {
      return;
    }
    const component = node.componentId
      ? versionDetails[node.versionId]?.components.find((item) => item.id === node.componentId)
      : undefined;
    if (node.componentId && !component) {
      return;
    }
    selectNode(node);
    switch (action) {
      case 'edit-identity':
        if (version.status === 'unpublished' && component) {
          openComponentIdentityDialog(version.id, component);
        }
        break;
      case 'edit':
        if (version.status !== 'unpublished') {
          return;
        }
        if (component) {
          openComponentEditDialog(version.id, component);
        } else {
          openEditVersionModal(version);
        }
        break;
      case 'delete':
        if (component) {
          if (version.status === 'unpublished') {
            openComponentDeleteDialog(version.id, component);
          }
        } else {
          openDeleteVersionModal(version);
        }
        break;
      case 'create-component':
        if (version.status === 'unpublished') {
          openComponentCreateDialog(version.id);
        }
        break;
      case 'fork':
        openForkModal(version);
        break;
      case 'preview':
        void previewVersion(version.id);
        break;
      case 'publish':
        void handlePublish(version.id);
        break;
      case 'unpublish':
        void handleUnpublish(version.id);
        break;
    }
  }

  function handleVersionSearch() {
    versionPagination.current = 1;
    void loadVersions();
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

  async function previewVersion(versionId: string) {
    previewContent.value = '';
    previewError.value = '';
    previewOpen.value = true;
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      previewError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executePreview(async () => {
        const result = await applicationApi.previewVersion(projectId, versionId, {});
        previewContent.value = result.compose_yaml;
      });
    } catch (error) {
      previewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    }
  }

  function setPreviewOpen(open: boolean) {
    previewOpen.value = open;
    if (!open) {
      previewContent.value = '';
      previewError.value = '';
    }
  }

  function openEditModal() {
    editErrors.name = '';
    editSubmitError.value = '';
    if (application.value) {
      Object.assign(editForm, {
        name: application.value.name,
        code: application.value.code,
      });
    }
    isEditDialogOpen.value = true;
  }

  async function handleEditOk() {
    editSubmitError.value = '';
    editErrors.name = editForm.name.trim() ? '' : t('application.validation.nameRequired');
    if (editErrors.name) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      editSubmitError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.update(projectId, applicationId, {
          name: editForm.name,
        });
        toast.success(t('application.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchApplication();
      });
    } catch (error) {
      editSubmitError.value =
        error instanceof Error ? error.message : t('application.toast.updateFailed');
    }
  }

  function openDeleteModal() {
    deleteApplicationError.value = '';
    isDeleteDialogOpen.value = true;
  }

  async function handleDeleteOk() {
    deleteApplicationError.value = '';
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      deleteApplicationError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.delete(projectId, applicationId);
        toast.success(t('application.toast.deleteSuccess'));
        router.push('/applications');
      });
    } catch (error) {
      deleteApplicationError.value =
        error instanceof Error ? error.message : t('application.toast.deleteFailed');
    }
  }

  function openCreateVersionModal() {
    editingVersionId.value = '';
    versionForm.label = '';
    versionForm.note = '';
    versionFormErrors.label = '';
    versionSubmitError.value = '';
    isVersionDialogOpen.value = true;
  }

  function openEditVersionModal(version: VersionResp) {
    editingVersionId.value = version.id;
    versionForm.label = version.label;
    versionForm.note = version.note ?? '';
    versionFormErrors.label = '';
    versionSubmitError.value = '';
    isVersionDialogOpen.value = true;
  }

  async function showCreatedVersion(version: VersionResp) {
    updateVersion(version);
    versionSearchText.value = '';
    versionPagination.current = 1;
    await loadVersions({ versionId: version.id });
  }

  async function saveVersion() {
    versionSubmitError.value = '';
    versionFormErrors.label = versionForm.label.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (versionFormErrors.label) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      versionSubmitError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const label = versionForm.label.trim();
        const note = versionForm.note.trim();
        if (editingVersionId.value) {
          const updated = await applicationApi.updateVersion(projectId, editingVersionId.value, {
            label,
            note,
          });
          updateVersion(updated);
          toast.success(t('application.toast.updateSuccess'));
          isVersionDialogOpen.value = false;
          await loadVersions();
        } else {
          const created = await applicationApi.createVersion(projectId, applicationId, {
            application_id: applicationId,
            label,
            note: note || undefined,
            components: [],
          });
          toast.success(t('application.toast.createVersionSuccess'));
          isVersionDialogOpen.value = false;
          await showCreatedVersion(created);
        }
      });
    } catch (error) {
      versionSubmitError.value =
        error instanceof Error ? error.message : t('application.toast.saveFailed');
    }
  }

  async function handlePublish(versionId: string) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        updateVersion(await applicationApi.publishVersion(projectId, versionId));
        toast.success(t('application.toast.publishSuccess'));
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.publishFailed'));
    }
  }

  async function handleUnpublish(versionId: string) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        updateVersion(await applicationApi.unpublishVersion(projectId, versionId));
        toast.success(t('application.toast.unpublishSuccess'));
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.unpublishFailed'));
    }
  }

  function openDeleteVersionModal(version: VersionResp) {
    pendingDeleteVersion.value = version;
    deleteVersionError.value = '';
    isDeleteVersionDialogOpen.value = true;
  }

  function setDeleteVersionDialogOpen(open: boolean) {
    isDeleteVersionDialogOpen.value = open;
    if (!open) {
      pendingDeleteVersion.value = null;
      deleteVersionError.value = '';
    }
  }

  async function handleDeleteVersionOk() {
    const target = pendingDeleteVersion.value;
    if (!target) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      deleteVersionError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersion(projectId, target.id);
        toast.success(t('application.toast.deleteVersionSuccess'));
        setDeleteVersionDialogOpen(false);
        delete versionDetails[target.id];
        delete versionDetailLoadState[target.id];
        await loadVersions();
      });
    } catch (error) {
      deleteVersionError.value =
        error instanceof Error ? error.message : t('application.toast.deleteVersionFailed');
    }
  }

  function resetComponentErrors() {
    Object.assign(componentErrors, { name: '', image: '', restartPolicy: '' });
  }

  function resetComponentForm() {
    Object.assign(componentForm, emptyComponentForm());
  }

  function openComponentCreateDialog(versionId: string) {
    pendingComponentVersionId.value = versionId;
    pendingComponent.value = undefined;
    resetComponentForm();
    resetComponentErrors();
    componentEditError.value = '';
    isComponentEditDialogOpen.value = true;
  }

  function openComponentEditDialog(versionId: string, component: VersionComponentResp) {
    pendingComponentVersionId.value = versionId;
    pendingComponent.value = component;
    Object.assign(componentForm, componentFormFromResponse(component));
    resetComponentErrors();
    componentEditError.value = '';
    isComponentEditDialogOpen.value = true;
  }

  function cancelComponentEditing() {
    resetComponentForm();
    resetComponentErrors();
    componentEditError.value = '';
    isComponentEditDialogOpen.value = false;
    pendingComponent.value = undefined;
    pendingComponentVersionId.value = '';
  }

  function setComponentEditDialogOpen(open: boolean) {
    if (open) {
      isComponentEditDialogOpen.value = true;
      return;
    }
    cancelComponentEditing();
  }

  async function saveComponentBasic() {
    const target = pendingComponent.value;
    const versionId = pendingComponentVersionId.value;
    if (!versionId) {
      return;
    }
    componentEditError.value = '';
    const result = componentBasicRequestFromForm(componentForm);
    if (!result.valid) {
      componentErrors.name =
        result.error === 'nameImage'
          ? componentForm.name.trim()
            ? ''
            : t('application.componentDetail.validation.componentNameRequired')
          : result.error === 'componentName'
            ? t('application.componentDetail.validation.componentName')
            : '';
      componentErrors.image =
        result.error === 'nameImage' && !componentForm.image.trim()
          ? t('application.componentDetail.validation.imageRequired')
          : '';
      componentErrors.restartPolicy =
        result.error === 'restartPolicy'
          ? t('application.componentDetail.validation.restartPolicy')
          : '';
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      componentEditError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const updated = target
          ? await applicationApi.updateVersionComponentBasic(
              projectId,
              versionId,
              target.id,
              result.value
            )
          : await applicationApi.createVersionComponent(projectId, versionId, {
              ...result.value,
              group_add: [],
            });
        const components = versionDetails[versionId]?.components;
        if (components) {
          if (target) {
            const index = components.findIndex((item) => item.id === updated.id);
            if (index !== -1) {
              components.splice(index, 1, updated);
            }
          } else {
            components.push(updated);
          }
          components.sort((a, b) => a.name.localeCompare(b.name));
        }
        selectNode({ versionId, componentId: updated.id });
        cancelComponentEditing();
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      componentEditError.value =
        error instanceof Error ? error.message : t('application.toast.updateFailed');
    }
  }

  function openComponentIdentityDialog(versionId: string, component: VersionComponentResp) {
    pendingComponentVersionId.value = versionId;
    pendingComponent.value = component;
    Object.assign(componentForm, componentFormFromResponse(component));
    componentIdentityError.value = '';
    isComponentIdentityDialogOpen.value = true;
  }

  function setComponentIdentityDialogOpen(open: boolean) {
    isComponentIdentityDialogOpen.value = open;
    if (!open) {
      resetComponentForm();
      pendingComponent.value = undefined;
      pendingComponentVersionId.value = '';
      componentIdentityError.value = '';
    }
  }

  async function saveComponentIdentity() {
    const target = pendingComponent.value;
    const versionId = pendingComponentVersionId.value;
    if (!target || !versionId) {
      return;
    }
    componentIdentityError.value = '';
    const result = componentIdentityRequestFromForm(componentForm);
    if (!result.valid) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      componentIdentityError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await applicationApi.updateVersionComponentIdentity(
          projectId,
          versionId,
          target.id,
          result.value
        );
        const components = versionDetails[versionId]?.components;
        if (components) {
          const index = components.findIndex((item) => item.id === updated.id);
          if (index !== -1) {
            components.splice(index, 1, updated);
          }
        }
        setComponentIdentityDialogOpen(false);
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      componentIdentityError.value =
        error instanceof Error ? error.message : t('application.toast.updateFailed');
    }
  }

  function openComponentDeleteDialog(versionId: string, component: VersionComponentResp) {
    pendingComponentVersionId.value = versionId;
    pendingComponent.value = component;
    componentDeleteError.value = '';
    isComponentDeleteDialogOpen.value = true;
  }

  function cancelComponentDeletion() {
    isComponentDeleteDialogOpen.value = false;
    pendingComponent.value = undefined;
    pendingComponentVersionId.value = '';
    componentDeleteError.value = '';
  }

  async function deleteComponent() {
    const target = pendingComponent.value;
    const versionId = pendingComponentVersionId.value;
    if (!target || !versionId) {
      return;
    }
    componentDeleteError.value = '';
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      componentDeleteError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersionComponent(projectId, versionId, target.id);
        const components = versionDetails[versionId]?.components;
        if (components) {
          const index = components.findIndex((item) => item.id === target.id);
          if (index !== -1) {
            components.splice(index, 1);
          }
        }
        if (selection.value?.versionId === versionId && selection.value.componentId === target.id) {
          selection.value = { versionId };
        }
        cancelComponentDeletion();
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      componentDeleteError.value =
        error instanceof Error ? error.message : t('application.toast.updateFailed');
    }
  }

  function openForkModal(version: VersionResp) {
    forkingVersionId.value = version.id;
    forkLabel.value = `${version.label}-copy`;
    forkLabelError.value = '';
    forkSubmitError.value = '';
    isForkDialogOpen.value = true;
  }

  async function handleForkOk() {
    forkSubmitError.value = '';
    forkLabelError.value = forkLabel.value.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (forkLabelError.value || !forkingVersionId.value) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      forkSubmitError.value = t('application.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const created = await applicationApi.forkVersion(projectId, forkingVersionId.value, {
          label: forkLabel.value.trim(),
        });
        toast.success(t('application.toast.forkSuccess'));
        isForkDialogOpen.value = false;
        await showCreatedVersion(created);
      });
    } catch (error) {
      forkSubmitError.value =
        error instanceof Error ? error.message : t('application.toast.forkFailed');
    }
  }

  onMounted(async () => {
    await fetchApplication();
    if (application.value) {
      await loadVersions();
    }
  });
</script>
