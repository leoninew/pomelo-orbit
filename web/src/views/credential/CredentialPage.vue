<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" aria-label="凭据工具栏">
      <SearchControl
        v-model="searchText"
        placeholder="搜索凭据名称"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button px-5" @click="triggerImport">
          <Upload class="size-4" />
          导入
        </button>
        <button class="app-button-primary px-5" @click="openCreateModal">
          <Plus class="size-4" />
          创建
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
        <p class="text-sm">{{ error || '加载失败' }}</p>
      </div>
      <AppEmptyState v-else-if="credentials.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[880px]">
          <thead>
            <tr>
              <th>名称</th>
              <th>类型</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="cred in credentials" :key="cred.id">
              <td>
                <router-link :to="`/credential/${cred.id}`" class="app-link">
                  {{ cred.name }}
                </router-link>
              </td>
              <td>
                <AppBadge variant="pill" tone="info">
                  {{ cred.type }}
                </AppBadge>
              </td>
              <td class="text-foreground">{{ formatTime(cred.created_at) }}</td>
              <td>
                <button class="app-link mr-3" @click="openEditModal(cred)">编辑</button>
                <button class="app-link-danger" @click="confirmDelete(cred.id)">删除</button>
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

    <input ref="fileInput" type="file" accept=".json" class="hidden" @change="handleFileImport" />
  </div>

  <AppDialog
    v-model:open="showCredentialDialog"
    :title="isEditing ? '编辑凭据' : '创建凭据'"
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
          placeholder="输入凭据名称"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
          :aria-invalid="errors.name ? 'true' : undefined"
          @input="errors.name = ''"
        />
        <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          凭据类型
          <span class="text-destructive">*</span>
        </label>
        <RawValueSelect
          v-model="form.type"
          :values="credentialTypeValues"
          :disabled="isEditing"
          placeholder="选择凭据类型"
        />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          凭据内容
          <span class="text-destructive">*</span>
        </label>
        <textarea
          v-model="form.data"
          rows="8"
          :placeholder="getDataPlaceholder(form.type)"
          class="app-textarea"
          :class="errors.data ? 'app-input-error' : ''"
          :aria-invalid="errors.data ? 'true' : undefined"
          @input="errors.data = ''"
        />
        <p v-if="errors.data" class="app-field-error text-xs">{{ errors.data }}</p>
      </div>
    </div>
    <template #footer>
      <AppDialogActions
        :busy="operating"
        @cancel="showCredentialDialog = false"
        @confirm="handleModalOk"
      />
    </template>
  </AppDialog>

  <AppDialog
    v-model:open="showDeleteDialog"
    title="确认删除"
    width-class="w-[min(420px,calc(100vw-32px))]"
  >
    <p class="text-sm text-foreground">确定要删除这个凭据吗？此操作不可恢复。</p>
    <template #footer>
      <AppDialogActions
        :busy="operating"
        variant="destructive"
        @cancel="showDeleteDialog = false"
        @confirm="handleDelete"
      />
    </template>
  </AppDialog>

  <AppDialog
    v-model:open="showImportDialog"
    title="导入凭据"
    width-class="w-[min(600px,calc(100vw-32px))]"
  >
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="importForm.name"
          type="text"
          placeholder="输入凭据名称"
          class="app-input"
          :class="importErrors.name ? 'app-input-error' : ''"
          :aria-invalid="importErrors.name ? 'true' : undefined"
          @input="importErrors.name = ''"
        />
        <p v-if="importErrors.name" class="app-field-error text-xs">{{ importErrors.name }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          凭据类型
          <span class="text-destructive">*</span>
        </label>
        <RawValueSelect
          v-model="importForm.type"
          :values="credentialTypeValues"
          placeholder="选择凭据类型"
        />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          凭据内容
          <span class="text-destructive">*</span>
        </label>
        <textarea
          v-model="importForm.data"
          rows="8"
          class="app-textarea"
          :class="importErrors.data ? 'app-input-error' : ''"
          :aria-invalid="importErrors.data ? 'true' : undefined"
          @input="importErrors.data = ''"
        />
        <p v-if="importErrors.data" class="app-field-error text-xs">
          {{ importErrors.data }}
        </p>
      </div>
    </div>
    <template #footer>
      <AppDialogActions
        :busy="operating"
        @cancel="showImportDialog = false"
        @confirm="handleImportOk"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { Plus, Upload } from 'lucide-vue-next';
  import { computed, nextTick, onMounted, reactive, ref } from 'vue';
  import { credentialApi } from '@/api/credential/credential';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type {
    CredentialImportReq,
    CredentialResp,
  } from '@/gen/proto/orbit/v1/credential/credential';
  import { formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const credentials = ref<CredentialResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const searchText = ref('');

  const fileInput = ref<HTMLInputElement>();
  const showCredentialDialog = ref(false);
  const showDeleteDialog = ref(false);
  const showImportDialog = ref(false);
  const isEditing = ref(false);
  const currentId = ref('');
  const pendingDeleteId = ref('');

  const form = reactive({
    name: '',
    type: 'github_token' as string,
    data: '',
  });
  const credentialTypeValues = ['github_token', 'gitee_token', 'git_ssh', 'registry_token'];
  const errors = reactive({ name: '', data: '' });
  const importForm = reactive<CredentialImportReq>({
    version: '',
    name: '',
    type: 'github_token',
    data: '',
  });
  const importErrors = reactive({ name: '', data: '' });

  function validate() {
    errors.name = form.name.trim() ? '' : '请输入凭据名称';
    errors.data = form.data.trim() ? '' : '请输入凭据内容';
    return !errors.name && !errors.data;
  }

  async function fetchCredentials() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await execute(async () => {
        const res = await credentialApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
          project_id: projectId,
        });
        credentials.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error('获取凭据列表失败');
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchCredentials();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchCredentials();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchCredentials();
  }

  function openCreateModal() {
    isEditing.value = false;
    currentId.value = '';
    Object.assign(form, { name: '', type: 'github_token', data: '' });
    Object.assign(errors, { name: '', data: '' });
    showCredentialDialog.value = true;
  }

  async function openEditModal(record: CredentialResp) {
    try {
      await executeOp(async () => {
        const detail = await credentialApi.get(record.id);
        isEditing.value = true;
        currentId.value = record.id;
        Object.assign(form, {
          name: detail.name,
          type: detail.type,
          data: detail.data,
        });
        Object.assign(errors, { name: '', data: '' });
        showCredentialDialog.value = true;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '获取凭据详情失败');
    }
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
        if (isEditing.value) {
          await credentialApi.update(currentId.value, {
            name: form.name,
            data: form.data,
          });
          toast.success('更新成功');
        } else {
          await credentialApi.create(
            {
              name: form.name,
              type: form.type,
              data: form.data,
            },
            { project_id: projectId }
          );
          toast.success('创建成功');
        }
        showCredentialDialog.value = false;
        fetchCredentials();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败');
    }
  }

  async function confirmDelete(id: string) {
    pendingDeleteId.value = id;
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
    showDeleteDialog.value = true;
  }

  async function handleDelete() {
    try {
      await executeOp(async () => {
        await credentialApi.delete(pendingDeleteId.value);
        toast.success('删除成功');
        showDeleteDialog.value = false;
        fetchCredentials();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    }
  }

  function getDataPlaceholder(type: string) {
    if (type === 'git_ssh') {
      return '-----BEGIN OPENSSH PRIVATE KEY-----\n...';
    }
    if (type === 'github_token') {
      return 'ghp_xxxxxxxxxxxxxxxxxxxx';
    }
    if (type === 'gitee_token') {
      return 'your_username:your_gitee_token';
    }
    return 'registry_token_here';
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
      const data = JSON.parse(await file.text()) as CredentialImportReq;
      if (!data.version) {
        toast.error('凭据导入文件缺少版本号');
        return;
      }
      Object.assign(importForm, {
        version: data.version,
        name: data.name || '',
        type: data.type,
        data: data.data || '',
      });
      Object.assign(importErrors, { name: '', data: '' });
      showImportDialog.value = true;
    } catch {
      toast.error('解析文件失败');
    } finally {
      target.value = '';
    }
  }

  async function handleImportOk() {
    importErrors.name = importForm.name.trim() ? '' : '请输入凭据名称';
    importErrors.data = importForm.data.trim() ? '' : '请输入凭据内容';
    if (importErrors.name || importErrors.data) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await executeOp(async () => {
        await credentialApi.importCredential(
          {
            version: importForm.version,
            name: importForm.name,
            type: importForm.type,
            data: importForm.data,
          },
          { project_id: projectId }
        );
        toast.success('导入成功');
        showImportDialog.value = false;
        fetchCredentials();
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '导入失败');
    }
  }

  onMounted(fetchCredentials);
</script>
