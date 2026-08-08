<template>
  <div class="space-y-6">
    <div class="app-toolbar-simple">
      <SearchControl
        v-model="search"
        placeholder="搜索阶段"
        :loading="status === 'loading'"
        @search="searchStages"
      />
      <button class="app-button-primary h-10 px-4" @click="openCreate">
        <Plus class="size-4" />
        新建阶段
      </button>
    </div>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <p v-else-if="status === 'error'" class="py-16 text-center text-sm text-destructive">
        {{ error || '加载阶段失败' }}
      </p>
      <AppEmptyState v-else-if="stages.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[860px]">
          <thead>
            <tr>
              <th>名称</th>
              <th>执行镜像</th>
              <th>版本</th>
              <th>说明</th>
              <th>更新时间</th>
              <th class="w-32">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="stage in stages" :key="stage.id">
              <td>
                <router-link :to="`/pipeline-stage/${stage.id}`" class="app-link">
                  {{ stage.name }}
                </router-link>
              </td>
              <td class="max-w-xs truncate text-foreground" :title="stage.image">
                {{ stage.image }}
              </td>
              <td>
                <AppBadge>v{{ stage.version }}</AppBadge>
              </td>
              <td class="max-w-xs truncate text-muted-foreground">
                {{ stage.description || '未填写' }}
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(stage.updated_at) }}</td>
              <td>
                <div class="flex items-center gap-3">
                  <router-link :to="`/pipeline-stage/${stage.id}`" class="app-link">
                    详情
                  </router-link>
                  <button class="app-link-danger" @click="openDelete(stage)">删除</button>
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
        @change-page-size="changePageSize"
      />
    </div>

    <AppDialog v-model:open="formOpen" title="新建阶段">
      <form class="space-y-4" @submit.prevent="save">
        <div class="space-y-1.5">
          <label class="app-field-label">
            名称
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            class="app-input"
            :class="errors.name && 'app-input-error'"
            :aria-invalid="errors.name ? 'true' : undefined"
            @input="errors.name = ''"
          />
          <p v-if="errors.name" class="app-field-error" role="alert">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">
            执行镜像
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.image"
            class="app-input"
            :class="errors.image && 'app-input-error'"
            :aria-invalid="errors.image ? 'true' : undefined"
            @input="errors.image = ''"
          />
          <p v-if="errors.image" class="app-field-error" role="alert">{{ errors.image }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">说明</label>
          <textarea v-model="form.description" rows="3" class="app-textarea" />
        </div>
        <p v-if="formError" class="app-field-error">{{ formError }}</p>
      </form>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="formOpen = false" @confirm="save" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="deleteOpen" title="删除阶段">
      <p class="text-sm text-muted-foreground">删除不会影响已引入到流水线的阶段或历史执行记录。</p>
      <p v-if="deleteError" class="app-field-error mt-3">{{ deleteError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          confirm-label="删除"
          variant="destructive"
          @cancel="deleteOpen = false"
          @confirm="remove"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useRouter } from 'vue-router';
  import { pipelineStageApi } from '@/api/pipeline/pipeline_stage';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { PipelineStageResp } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const projectStore = useProjectStore();
  const router = useRouter();
  const toast = useToast();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const stages = ref<PipelineStageResp[]>([]);
  const pendingDelete = ref<PipelineStageResp>();
  const search = ref('');
  const formOpen = ref(false);
  const deleteOpen = ref(false);
  const formError = ref('');
  const deleteError = ref('');
  const form = reactive({ name: '', image: '', description: '' });
  const errors = reactive({ name: '', image: '' });
  const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  async function fetchStages() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) return;
    await execute(async () => {
      const response = await pipelineStageApi.list({
        project_id: projectId,
        search: search.value.trim() || undefined,
        page: pagination.current,
        per_page: pagination.pageSize,
      });
      stages.value = response.items;
      pagination.total = response.total;
    });
  }
  function searchStages() {
    pagination.current = 1;
    void fetchStages();
  }
  function goPage(page: number) {
    pagination.current = page;
    void fetchStages();
  }
  function changePageSize(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    void fetchStages();
  }
  function clearErrors() {
    Object.assign(errors, { name: '', image: '' });
    formError.value = '';
  }
  function openCreate() {
    Object.assign(form, { name: '', image: '', description: '' });
    clearErrors();
    formOpen.value = true;
  }
  function openDelete(stage: PipelineStageResp) {
    pendingDelete.value = stage;
    deleteError.value = '';
    deleteOpen.value = true;
  }
  async function save() {
    Object.assign(errors, {
      name: form.name.trim() ? '' : '请输入阶段名称',
      image: form.image.trim() ? '' : '请输入执行镜像',
    });
    if (Object.values(errors).some(Boolean)) return;
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      formError.value = '请先选择项目';
      return;
    }
    try {
      await executeOperation(async () => {
        const stage = await pipelineStageApi.create(
          {
            name: form.name.trim(),
            image: form.image.trim(),
            script: '',
            description: form.description,
            artifacts: [],
          },
          { project_id: projectId }
        );
        formOpen.value = false;
        toast.success('阶段已创建');
        await router.push(`/pipeline-stage/${stage.id}`);
      });
    } catch (reason) {
      formError.value = reason instanceof Error ? reason.message : '保存阶段失败';
    }
  }
  async function remove() {
    const stage = pendingDelete.value;
    if (!stage) return;
    try {
      await executeOperation(async () => {
        await pipelineStageApi.delete(stage.id);
        deleteOpen.value = false;
        await fetchStages();
        toast.success('阶段已删除');
      });
    } catch (reason) {
      deleteError.value = reason instanceof Error ? reason.message : '删除阶段失败';
    }
  }
  watch(
    () => projectStore.activeProjectId,
    () => {
      pagination.current = 1;
      void fetchStages();
    }
  );
  onMounted(() => void fetchStages());
</script>
