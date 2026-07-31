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
    <AppSpinner v-if="status === 'loading'" class="py-12" />

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
            height="300px"
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
                <th>{{ t('buildStageDetail.artifactType') }}</th>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('buildStageDetail.pathOrImage') }}</th>
                <th class="w-32">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(artifact, idx) in sortableArtifacts" :key="idx">
                <td class="text-muted-foreground">{{ idx + 1 }}</td>
                <td>
                  <AppBadge variant="pill">
                    {{ artifact.type }}
                  </AppBadge>
                </td>
                <td class="text-foreground">{{ artifact.name }}</td>
                <td class="text-muted-foreground">{{ artifact.path }}</td>
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
            {{ t('buildStageDetail.artifactType') }}
            <span class="text-destructive">*</span>
          </label>
          <RawValueSelect
            v-model="artifactForm.type"
            :values="artifactTypeValues"
            :placeholder="t('buildStageDetail.artifactTypePlaceholder')"
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
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('buildStageDetail.pathOrImage') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="artifactForm.path"
            type="text"
            class="app-input"
            :class="artifactErrors.path ? 'app-input-error' : ''"
            :placeholder="t('buildStageDetail.artifactPathPlaceholder')"
            :aria-invalid="artifactErrors.path ? 'true' : undefined"
            @input="artifactErrors.path = ''"
          />
          <p v-if="artifactErrors.path" class="app-field-error" role="alert">
            {{ artifactErrors.path }}
          </p>
        </div>
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
  import { ArrowLeft, Copy, Pencil, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { pipelineStageApi } from '@/api/pipeline/pipeline_stage';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    ArtifactConfigResp,
    PipelineStageResp,
  } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import { formatTime } from '@/utils/time';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n({ useScope: 'global' });
  const stageId = computed(() => route.params.id as string);
  const toast = useToast();

  const { status, execute } = useStatusAsync();
  const { loading: saving, execute: executeSave } = useStatusAsync();
  const { loading: deleting, execute: executeDelete } = useStatusAsync();
  const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

  const stage = ref<PipelineStageResp>();
  const isDeleteDialogOpen = ref(false);
  const isEditDialogOpen = ref(false);
  const isArtifactDialogOpen = ref(false);
  const isDeleteArtifactDialogOpen = ref(false);
  const showScriptDrawer = ref(false);
  const scriptTemp = ref('');
  const artifactTypeValues = ['docker_image', 'binary'];
  const form = reactive({ name: '', image: '', description: '' });
  const editErrors = reactive({ name: '', image: '' });
  const artifactForm = reactive({
    isEdit: false,
    order: -1,
    type: 'docker_image',
    name: '',
    path: '',
  });
  const artifactErrors = reactive({ name: '', path: '' });
  const sortableArtifacts = ref<ArtifactConfigResp[]>([]);
  const artifactToDelete = ref(-1);

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
      type: 'docker_image',
      name: '',
      path: '',
    });
    Object.assign(artifactErrors, { name: '', path: '' });
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
      type: artifact.type,
      name: artifact.name,
      path: artifact.path,
    });
    Object.assign(artifactErrors, { name: '', path: '' });
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
    artifactErrors.path = artifactForm.path.trim()
      ? ''
      : t('buildStageDetail.artifactPathRequired');
    if (artifactErrors.name || artifactErrors.path) {
      return;
    }

    if (artifactForm.isEdit) {
      sortableArtifacts.value[artifactForm.order] = {
        type: artifactForm.type,
        name: artifactForm.name,
        path: artifactForm.path,
      };
    } else {
      if (sortableArtifacts.value.some((a) => a.name === artifactForm.name)) {
        artifactErrors.name = t('buildStageDetail.artifactNameExists');
        return;
      }
      sortableArtifacts.value.push({
        type: artifactForm.type,
        name: artifactForm.name,
        path: artifactForm.path,
      });
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
  onMounted(fetchStage);
</script>
