<template>
  <div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div>
          <h1 class="text-xl font-semibold text-foreground">
            {{ stage?.name ?? t('buildStageDetail.title') }}
          </h1>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="stage" class="app-button-primary h-9 px-3" @click="openEditModal">
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
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
        <button class="app-button h-9 px-4" @click="router.push('/ci/build-stage')">
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
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('buildStageDetail.basicInfo') }}</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.name') }}</dt>
            <dd class="text-foreground">{{ stage.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('buildStageDetail.version') }}</dt>
            <dd class="text-foreground">v{{ stage.version }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('buildStageDetail.image') }}</dt>
            <dd class="text-foreground">{{ stage.image }}</dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.description') }}</dt>
            <dd class="text-foreground">{{ stage.description || '—' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(stage.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(stage.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- Script Card -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">{{ t('buildStageDetail.script') }}</h2>
          <div class="flex items-center gap-2">
            <button class="app-button-primary h-8 px-3" @click="openScriptDrawer">
              <Pencil class="size-4" />
              {{ t('common.edit') }}
            </button>
            <button v-if="stage.script" class="app-button h-8 px-3" @click="handleCopyScript">
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
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">{{ t('buildStageDetail.artifactConfig') }}</h2>
          <button class="app-button-primary h-8 px-3" @click="openAddArtifactModal">
            <Plus class="size-4" />
            {{ t('buildStageDetail.addArtifact') }}
          </button>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[720px]">
            <thead>
              <tr>
                <th>#</th>
                <th>{{ t('buildStageDetail.artifactType') }}</th>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('buildStageDetail.pathOrImage') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="sortableArtifacts.length === 0">
                <td colspan="5" class="text-center text-muted-foreground">
                  {{ t('buildStageDetail.noArtifactConfigResp') }}
                </td>
              </tr>
              <tr v-for="(artifact, idx) in sortableArtifacts" :key="idx">
                <td class="text-muted-foreground">{{ idx + 1 }}</td>
                <td>
                  <AppBadge>
                    {{ getArtifactTypeLabel(artifact.type) }}
                  </AppBadge>
                </td>
                <td class="text-foreground">{{ artifact.name }}</td>
                <td class="text-muted-foreground">{{ artifact.path }}</td>
                <td>
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openEditArtifactModal(idx)">
                      {{ t('common.edit') }}
                    </button>
                    <button class="app-link-danger" @click="confirmRemoveArtifact(idx)">
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
          <label class="app-field-label block">{{ t('common.name') }}</label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :placeholder="t('buildStageDetail.namePlaceholder')"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('buildStageDetail.image') }}</label>
          <input
            v-model="form.image"
            type="text"
            class="app-input"
            :placeholder="t('buildStageDetail.imagePlaceholder')"
          />
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
        <button class="app-button" @click="isEditDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="saving" @click="handleSave">
          {{ t('common.save') }}
        </button>
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
        <button class="app-button" @click="closeScriptDrawer">{{ t('common.cancel') }}</button>
        <button class="app-button-primary" :disabled="saving" @click="confirmScript">
          {{ t('common.save') }}
        </button>
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
          <label class="app-field-label block">{{ t('buildStageDetail.artifactType') }}</label>
          <SelectControl
            v-model="artifactForm.type"
            :options="artifactTypeOptions"
            :placeholder="t('buildStageDetail.artifactTypePlaceholder')"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('common.name') }}</label>
          <input
            v-model="artifactForm.name"
            type="text"
            class="app-input"
            :placeholder="t('buildStageDetail.artifactNamePlaceholder')"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('buildStageDetail.pathOrImage') }}</label>
          <input
            v-model="artifactForm.path"
            type="text"
            class="app-input"
            :placeholder="t('buildStageDetail.artifactPathPlaceholder')"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isArtifactDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="saving" @click="handleSaveArtifact">
          {{ artifactForm.isEdit ? t('common.save') : t('common.add') }}
        </button>
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
        <button class="app-button" @click="isDeleteArtifactDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="saving" @click="removeArtifact">
          {{ t('common.delete') }}
        </button>
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
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="deleting" @click="handleDelete">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Copy, Pencil, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { buildStageApi } from '@/api/ci';
  import AppDialog from '@/components/AppDialog.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ArtifactConfigResp, BuildStageResp } from '@/gen/proto/orbit/v1/build_stage';
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

  const stage = ref<BuildStageResp>();
  const isDeleteDialogOpen = ref(false);
  const isEditDialogOpen = ref(false);
  const isArtifactDialogOpen = ref(false);
  const isDeleteArtifactDialogOpen = ref(false);
  const showScriptDrawer = ref(false);
  const scriptTemp = ref('');
  const artifactTypeOptions = computed(() => [
    { value: 'docker_image', label: t('buildStageDetail.artifactTypes.dockerImage') },
    { value: 'binary', label: t('buildStageDetail.artifactTypes.binary') },
  ]);
  const form = reactive({ name: '', image: '', description: '' });
  const artifactForm = reactive({
    isEdit: false,
    order: -1,
    type: 'docker_image',
    name: '',
    path: '',
  });
  const sortableArtifacts = ref<ArtifactConfigResp[]>([]);
  const artifactToDelete = ref(-1);

  async function fetchStage() {
    try {
      await execute(async () => {
        stage.value = await buildStageApi.get(stageId.value);
        sortableArtifacts.value = stage.value.artifacts ? [...stage.value.artifacts] : [];
      });
    } catch {
      toast.error(t('buildStageDetail.fetchFailed'));
      router.push('/ci/build-stage');
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
        const updated = await buildStageApi.update(stageId.value, {
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
    try {
      await executeSave(async () => {
        const updated = await buildStageApi.update(stageId.value, {
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
    isArtifactDialogOpen.value = true;
  }

  function getArtifactTypeLabel(type: string) {
    return artifactTypeOptions.value.find((option) => option.value === type)?.label ?? type;
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
        stage.value = await buildStageApi.update(stageId.value, {
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
    if (!artifactForm.name.trim() || !artifactForm.path.trim()) {
      toast.error(t('buildStageDetail.artifactFieldsRequired'));
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
        toast.error(t('buildStageDetail.artifactNameExists'));
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
        stage.value = await buildStageApi.update(stageId.value, {
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
        const newStage = await buildStageApi.duplicate(stageId.value, {});
        toast.success(t('buildStageDetail.duplicateSuccess'));
        router.push(`/ci/build-stage/${newStage.id}`);
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.duplicateFailed'));
    }
  }

  async function handleDelete() {
    try {
      await executeDelete(async () => {
        await buildStageApi.delete(stageId.value);
        toast.success(t('buildStageDetail.deleteSuccess'));
        router.push('/ci/build-stage');
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t('buildStageDetail.deleteFailed'));
    }
  }

  watch(stageId, fetchStage);
  onMounted(fetchStage);
</script>
