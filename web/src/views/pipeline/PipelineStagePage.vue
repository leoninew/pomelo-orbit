<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" aria-label="构建阶段工具栏">
      <SearchControl
        v-model="searchText"
        placeholder="搜索名称/描述"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <button class="app-button-primary px-5" @click="openCreateModal">
        <Plus class="size-4" />
        创建
      </button>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || '获取 Stage 列表失败' }}</p>
      </div>
      <AppEmptyState v-else-if="stages.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[920px]">
          <thead>
            <tr>
              <th>名称</th>
              <th>镜像</th>
              <th>版本</th>
              <th>描述</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="stage in stages" :key="stage.id">
              <td>
                <router-link :to="`/pipeline/stage/${stage.id}`" class="app-link whitespace-nowrap">
                  {{ stage.name }}
                </router-link>
              </td>
              <td class="max-w-64 truncate text-foreground" :title="stage.image">
                {{ stage.image }}
              </td>
              <td>
                <AppBadge>v{{ stage.version }}</AppBadge>
              </td>
              <td
                class="max-w-xs truncate text-muted-foreground"
                :title="stage.description || undefined"
              >
                {{ stage.description }}
              </td>
              <td class="whitespace-nowrap text-muted-foreground">
                {{ formatTime(stage.created_at) }}
              </td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditModal(stage)">编辑</button>
                  <button
                    class="app-link"
                    :disabled="duplicating"
                    @click="handleDuplicate(stage.id)"
                  >
                    复制
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
        :total="pagination.total"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>
  </div>

  <AppDialog
    v-model:open="isModalOpen"
    title="创建构建"
    width-class="w-[min(600px,calc(100vw-32px))]"
  >
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.name"
          type="text"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
          placeholder="例如: build"
          :aria-invalid="errors.name ? 'true' : undefined"
          @input="errors.name = ''"
        />
        <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          镜像
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.image"
          type="text"
          class="app-input"
          :class="errors.image ? 'app-input-error' : ''"
          placeholder="例如: alpine:latest"
          :aria-invalid="errors.image ? 'true' : undefined"
          @input="errors.image = ''"
        />
        <p v-if="errors.image" class="app-field-error text-xs">{{ errors.image }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">描述</label>
        <input v-model="form.description" type="text" class="app-input" placeholder="简短描述" />
      </div>
    </div>

    <template #footer>
      <AppDialogActions :busy="operating" @cancel="isModalOpen = false" @confirm="handleModalOk" />
    </template>
  </AppDialog>

  <AppDialog
    :open="showEditModal"
    title="编辑构建"
    width-class="w-[min(600px,calc(100vw-32px))]"
    @update:open="handleEditModalOpenChange"
  >
    <form class="space-y-4" novalidate @submit.prevent="handleEditOk">
      <div class="space-y-1.5">
        <label for="edit-stage-name" class="app-field-label block">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-stage-name"
          v-model="editForm.name"
          type="text"
          class="app-input"
          :class="editErrors.name ? 'app-input-error' : ''"
          placeholder="例如: build"
          :aria-invalid="editErrors.name ? 'true' : undefined"
          :aria-describedby="editErrors.name ? 'edit-stage-name-error' : undefined"
          @input="clearEditError('name')"
        />
        <p
          v-if="editErrors.name"
          id="edit-stage-name-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.name }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label for="edit-stage-image" class="app-field-label block">
          镜像
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-stage-image"
          v-model="editForm.image"
          type="text"
          class="app-input"
          :class="editErrors.image ? 'app-input-error' : ''"
          placeholder="例如: alpine:latest"
          :aria-invalid="editErrors.image ? 'true' : undefined"
          :aria-describedby="editErrors.image ? 'edit-stage-image-error' : undefined"
          @input="clearEditError('image')"
        />
        <p
          v-if="editErrors.image"
          id="edit-stage-image-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.image }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label for="edit-stage-description" class="app-field-label block">描述</label>
        <input
          id="edit-stage-description"
          v-model="editForm.description"
          type="text"
          class="app-input"
          placeholder="简短描述"
        />
      </div>
    </form>

    <template #footer>
      <AppDialogActions :busy="operating" @cancel="closeEditModal" @confirm="handleEditOk" />
    </template>
  </AppDialog>
</template>
<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { ToolbarRoot } from 'reka-ui';
  import { pipelineStageApi } from '@/api/pipeline/pipeline_stage';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { PipelineStageResp } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import { formatTime } from '@/utils/time';

  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

  const stages = ref<PipelineStageResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const searchText = ref('');
  const isModalOpen = ref(false);
  const showEditModal = ref(false);
  const editingStage = ref<PipelineStageResp>();

  const form = reactive({ name: '', image: '', description: '' });
  const errors = reactive({ name: '', image: '' });
  const editForm = reactive({ name: '', image: '', description: '' });
  const editErrors = reactive({ name: '', image: '' });

  function validate() {
    errors.name = form.name.trim() ? '' : '请输入名称';
    errors.image = form.image.trim() ? '' : '请输入镜像';
    return !errors.name && !errors.image;
  }

  async function fetchStages() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await execute(async () => {
        const res = await pipelineStageApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
          project_id: projectId,
        });
        stages.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error('获取 Stage 列表失败');
    }
  }

  function handleSearch() {
    if (status.value === 'loading') {
      return;
    }
    pagination.current = 1;
    fetchStages();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchStages();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchStages();
  }

  function openCreateModal() {
    Object.assign(form, { name: '', image: '', description: '' });
    Object.assign(errors, { name: '', image: '' });
    isModalOpen.value = true;
  }

  function resetEditForm() {
    if (!editingStage.value) {
      return;
    }
    Object.assign(editForm, {
      name: editingStage.value.name,
      image: editingStage.value.image,
      description: editingStage.value.description,
    });
    Object.assign(editErrors, { name: '', image: '' });
  }

  function clearEditError(field: 'name' | 'image') {
    editErrors[field] = '';
  }

  function openEditModal(stage: PipelineStageResp) {
    editingStage.value = stage;
    resetEditForm();
    showEditModal.value = true;
  }

  function handleEditModalOpenChange(open: boolean) {
    showEditModal.value = open;
    if (!open) {
      Object.assign(editErrors, { name: '', image: '' });
    }
  }

  function closeEditModal() {
    handleEditModalOpenChange(false);
  }

  async function handleModalOk() {
    if (!validate()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await executeOp(async () => {
        await pipelineStageApi.create(
          {
            name: form.name,
            image: form.image,
            script: '',
            artifacts: [],
            description: form.description,
          },
          { project_id: projectId }
        );
        toast.success('创建成功');
        isModalOpen.value = false;
        fetchStages();
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败');
    }
  }

  async function handleEditOk() {
    const stage = editingStage.value;
    editErrors.name = editForm.name.trim() ? '' : '请输入名称';
    editErrors.image = editForm.image.trim() ? '' : '请输入镜像';
    if (!stage || editErrors.name || editErrors.image) {
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await pipelineStageApi.update(stage.id, {
          name: editForm.name,
          image: editForm.image,
          description: editForm.description,
        });
        stages.value = stages.value.map((item) => (item.id === updated.id ? updated : item));
        editingStage.value = updated;
        toast.success('更新成功');
        closeEditModal();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '更新失败');
    }
  }

  async function handleDuplicate(id: string) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await executeDuplicate(async () => {
        const newStage = await pipelineStageApi.duplicate(id, {});
        toast.success('复制成功');
        router.push(`/pipeline/stage/${newStage.id}`);
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '复制失败');
    }
  }

  onMounted(fetchStages);
</script>
