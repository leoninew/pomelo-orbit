<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ version?.label || t('application.versionDetail.title') }}
        </h1>
        <DetailHeaderMeta v-if="version">
          <AppBadge variant="status" :tone="versionStatusTone(version.status)">
            {{ version.status }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="version" class="app-button h-9 px-3" :disabled="operating" @click="preview">
          <FileCode2 class="size-4" />
          {{ t('application.detail.actions.preview') }}
        </button>
        <button
          v-if="version && isEditable"
          class="app-button-primary h-9 px-3"
          :disabled="operating"
          @click="handlePublish"
        >
          {{ t('application.detail.actions.publish') }}
        </button>
        <button
          v-else-if="version && isPublished"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="handleUnpublish"
        >
          <RotateCcw class="size-4" />
          {{ t('application.detail.actions.unpublish') }}
        </button>
        <button v-if="version" class="app-button h-9 px-3" @click="openForkModal">
          {{ t('application.detail.actions.fork') }}
        </button>
        <button
          v-if="version && isEditable"
          :disabled="operating"
          class="app-button-danger h-9 px-3"
          @click="isDeleteDialogOpen = true"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppLoadingState v-if="loading" size="section" />

    <template v-else-if="version">
      <VersionBasicInfoCard
        :version="version"
        :editable="isEditable"
        :disabled="operating"
        @edit="openBasicModal"
      />
      <VersionComponentsCard
        :version-id="version.id"
        :components="version.components"
        :editable="isEditable"
        :disabled="operating"
        @add="openComponentDialog"
        @edit="openComponentEditDialog"
        @delete="openComponentDeleteDialog"
      />
    </template>

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

    <!-- 编辑基本信息 -->
    <AppDialog
      v-model:open="isBasicDialogOpen"
      :title="t('application.versionDetail.dialog.editBasic')"
      width-class="w-[min(480px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.versionLabel') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="basicForm.label"
            type="text"
            class="app-input"
            :class="basicFormError ? 'app-input-error' : ''"
            :placeholder="t('application.detail.placeholders.versionLabel')"
            :aria-invalid="basicFormError ? 'true' : undefined"
            @input="basicFormError = ''"
          />
          <p v-if="basicFormError" class="app-field-error mt-1 text-xs">{{ basicFormError }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.note') }}
          </label>
          <input
            v-model="basicForm.note"
            type="text"
            class="app-input"
            :placeholder="t('application.detail.placeholders.note')"
          />
        </div>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="isBasicDialogOpen = false"
          @confirm="saveBasic"
        />
      </template>
    </AppDialog>

    <!-- 添加组件 -->
    <AppDialog
      v-model:open="isComponentDialogOpen"
      :title="t('application.versionDetail.dialog.addComponent')"
      width-class="w-[min(640px,calc(100vw-32px))]"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.component') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="componentForm.name"
            class="app-input"
            :class="componentFormErrors.name ? 'app-input-error' : ''"
            type="text"
            :placeholder="t('application.detail.placeholders.componentName')"
            :aria-invalid="componentFormErrors.name ? 'true' : undefined"
            @input="componentFormErrors.name = ''"
          />
          <p v-if="componentFormErrors.name" class="app-field-error" role="alert">
            {{ componentFormErrors.name }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.image') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="componentForm.image"
            class="app-input"
            :class="componentFormErrors.image ? 'app-input-error' : ''"
            type="text"
            :placeholder="t('application.detail.placeholders.componentImage')"
            :aria-invalid="componentFormErrors.image ? 'true' : undefined"
            @input="componentFormErrors.image = ''"
          />
          <p v-if="componentFormErrors.image" class="app-field-error" role="alert">
            {{ componentFormErrors.image }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.pullPolicy') }}
          </label>
          <RawValueSelect
            v-model="componentForm.pull_policy"
            :placeholder="t('common.notSet')"
            :values="pullPolicyValues"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.restartPolicy') }}
          </label>
          <RawValueSelect
            v-model="componentForm.restart_policy"
            :placeholder="t('common.notSet')"
            :values="restartPolicyValues"
          />
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.command') }}
          </label>
          <textarea v-model="componentForm.command" class="app-textarea" rows="3" />
        </div>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.create')"
          @cancel="closeComponentDialog"
          @confirm="createComponent"
        />
      </template>
    </AppDialog>

    <!-- 编辑组件基本信息 -->
    <AppDialog
      :open="isComponentEditDialogOpen"
      :title="t('application.componentDetail.sections.basic')"
      width-class="w-[min(640px,calc(100vw-32px))]"
      @update:open="setComponentEditDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.component') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="componentEditForm.name"
            class="app-input"
            :class="componentEditErrors.name ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="componentEditErrors.name ? 'true' : undefined"
            @input="componentEditErrors.name = ''"
          />
          <p v-if="componentEditErrors.name" class="app-field-error" role="alert">
            {{ componentEditErrors.name }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.image') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="componentEditForm.image"
            class="app-input"
            :class="componentEditErrors.image ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="componentEditErrors.image ? 'true' : undefined"
            @input="componentEditErrors.image = ''"
          />
          <p v-if="componentEditErrors.image" class="app-field-error" role="alert">
            {{ componentEditErrors.image }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.pullPolicy') }}
          </label>
          <RawValueSelect
            v-model="componentEditForm.pull_policy"
            :placeholder="t('common.notSet')"
            :values="pullPolicyValues"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.restartPolicy') }}
          </label>
          <RawValueSelect
            v-model="componentEditForm.restart_policy"
            :placeholder="t('common.notSet')"
            :values="restartPolicyValues"
          />
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.command') }}
          </label>
          <textarea v-model="componentEditForm.command" class="app-textarea" rows="3" />
        </div>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="cancelComponentEditing"
          @confirm="saveComponentBasic"
        />
      </template>
    </AppDialog>

    <!-- 删除组件 -->
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

    <!-- Fork -->
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

    <!-- Delete -->
    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('application.detail.dialog.deleteVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.deleteVersionConfirm', {
            label: version?.label || '-',
          })
        }}
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
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, FileCode2, RotateCcw, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VersionComponentResp, VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { versionStatusTone } from '@/utils/status';
  import VersionBasicInfoCard from './components/VersionBasicInfoCard.vue';
  import VersionComponentsCard from './components/VersionComponentsCard.vue';
  import {
    componentBasicRequestFromForm,
    componentCreateRequestFromForm,
    componentFormFromResponse,
    emptyComponentForm,
  } from './componentForm';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const versionId = route.params.id as string;

  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();

  const version = ref<VersionResp>();
  const previewOpen = ref(false);
  const previewContent = ref('');
  const previewError = ref('');

  const isBasicDialogOpen = ref(false);
  const isComponentDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isComponentEditDialogOpen = ref(false);
  const isComponentDeleteDialogOpen = ref(false);
  const pendingComponent = ref<VersionComponentResp>();
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const basicFormError = ref('');
  const componentFormErrors = reactive({ name: '', image: '' });
  const componentEditErrors = reactive({ name: '', image: '' });

  const basicForm = reactive({
    label: '',
    note: '',
  });
  const componentForm = reactive(emptyComponentForm());
  const componentEditForm = reactive(emptyComponentForm());
  const pullPolicyValues = ['always', 'missing', 'never'];
  const restartPolicyValues = ['no', 'unless-stopped'];
  const isEditable = computed(() => version.value?.status === 'unpublished');
  const isPublished = computed(() => version.value?.status === 'published');

  async function fetchVersion() {
    try {
      await execute(async () => {
        version.value = await applicationApi.getVersion(versionId);
      });
    } catch {
      toast.error(t('application.toast.loadVersionsFailed'));
      router.push('/versions');
    }
  }

  function goBack() {
    const appId = version.value?.application_id;
    if (appId) {
      router.push(`/application/${appId}`);
      return;
    }
    router.push('/applications');
  }

  async function preview() {
    previewContent.value = '';
    previewError.value = '';
    previewOpen.value = true;
    try {
      await executePreview(async () => {
        const result = await applicationApi.previewVersion(versionId, {});
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

  function openBasicModal() {
    if (!version.value) {
      return;
    }
    basicForm.label = version.value.label;
    basicForm.note = version.value.note === undefined ? '' : version.value.note;
    basicFormError.value = '';
    isBasicDialogOpen.value = true;
  }

  async function saveBasic() {
    basicFormError.value =
      basicForm.label === '' ? t('application.validation.versionLabelRequired') : '';
    if (basicFormError.value) {
      return;
    }
    try {
      await executeOp(async () => {
        version.value = await applicationApi.updateVersion(versionId, {
          label: basicForm.label,
          note: basicForm.note,
        });
        toast.success(t('application.toast.updateSuccess'));
        isBasicDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openComponentDialog() {
    Object.assign(componentForm, emptyComponentForm());
    Object.assign(componentFormErrors, { name: '', image: '' });
    isComponentDialogOpen.value = true;
  }

  function closeComponentDialog() {
    isComponentDialogOpen.value = false;
    Object.assign(componentFormErrors, { name: '', image: '' });
  }

  async function createComponent() {
    const result = componentCreateRequestFromForm(componentForm);
    if (!result.valid) {
      componentFormErrors.name =
        result.error === 'nameImage' && !componentForm.name.trim()
          ? t('application.componentDetail.validation.componentNameRequired')
          : result.error === 'componentName'
            ? t('application.componentDetail.validation.componentName')
            : '';
      componentFormErrors.image =
        result.error === 'nameImage' && !componentForm.image.trim()
          ? t('application.componentDetail.validation.imageRequired')
          : '';
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.createVersionComponent(versionId, result.value);
        version.value = await applicationApi.getVersion(versionId);
        toast.success(t('application.toast.updateSuccess'));
        closeComponentDialog();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function resetComponentEditErrors() {
    Object.assign(componentEditErrors, { name: '', image: '' });
  }

  function openComponentEditDialog(component: VersionComponentResp) {
    pendingComponent.value = component;
    Object.assign(componentEditForm, componentFormFromResponse(component));
    resetComponentEditErrors();
    isComponentEditDialogOpen.value = true;
  }

  function cancelComponentEditing() {
    Object.assign(componentEditForm, emptyComponentForm());
    resetComponentEditErrors();
    isComponentEditDialogOpen.value = false;
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
    if (!target) {
      return;
    }
    const result = componentBasicRequestFromForm(componentEditForm);
    if (!result.valid) {
      componentEditErrors.name =
        result.error === 'nameImage'
          ? componentEditForm.name.trim()
            ? ''
            : t('application.componentDetail.validation.componentNameRequired')
          : t('application.componentDetail.validation.componentName');
      componentEditErrors.image = componentEditForm.image.trim()
        ? ''
        : t('application.componentDetail.validation.imageRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await applicationApi.updateVersionComponentBasic(
          versionId,
          target.id,
          result.value
        );
        if (version.value) {
          const index = version.value.components.findIndex((item) => item.id === updated.id);
          if (index !== -1) {
            version.value.components.splice(index, 1, updated);
          }
        }
        pendingComponent.value = updated;
        cancelComponentEditing();
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openComponentDeleteDialog(component: VersionComponentResp) {
    pendingComponent.value = component;
    isComponentDeleteDialogOpen.value = true;
  }

  function cancelComponentDeletion() {
    isComponentDeleteDialogOpen.value = false;
    pendingComponent.value = undefined;
  }

  async function deleteComponent() {
    const target = pendingComponent.value;
    if (!target) {
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersionComponent(versionId, target.id);
        if (version.value) {
          const index = version.value.components.findIndex((item) => item.id === target.id);
          if (index !== -1) {
            version.value.components.splice(index, 1);
          }
        }
        cancelComponentDeletion();
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function handlePublish() {
    try {
      await executeOp(async () => {
        version.value = await applicationApi.publishVersion(versionId);
        toast.success(t('application.toast.publishSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.publishFailed'));
    }
  }

  async function handleUnpublish() {
    try {
      await executeOp(async () => {
        version.value = await applicationApi.unpublishVersion(versionId);
        toast.success(t('application.toast.unpublishSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.unpublishFailed'));
    }
  }

  function openForkModal() {
    forkLabel.value = version.value ? `${version.value.label}-copy` : '';
    forkLabelError.value = '';
    isForkDialogOpen.value = true;
  }

  async function handleForkOk() {
    forkLabelError.value = forkLabel.value.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (forkLabelError.value) {
      return;
    }
    try {
      await executeOp(async () => {
        const created = await applicationApi.forkVersion(versionId, {
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

  async function handleDeleteOk() {
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersion(versionId);
        toast.success(t('application.toast.deleteVersionSuccess'));
        isDeleteDialogOpen.value = false;
        goBack();
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.deleteVersionFailed')
      );
    }
  }

  onMounted(async () => {
    await fetchVersion();
  });
</script>
