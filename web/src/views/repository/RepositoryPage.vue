<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" aria-label="仓库工具栏">
      <SearchControl
        v-model="searchText"
        placeholder="搜索名称/地址"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <button class="app-button-primary px-5" @click="openCreateModal">
        <Plus class="size-4" />
        创建
      </button>
    </ToolbarRoot>

    <!-- Table Card -->
    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || '获取项目列表失败' }}</p>
      </div>
      <AppEmptyState v-else-if="repositories.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1040px]">
          <thead>
            <tr>
              <th>名称</th>
              <th>编码</th>
              <th>仓库类型</th>
              <th>默认分支</th>
              <th>地址/目录</th>
              <th>Git 凭据</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in repositories" :key="p.id">
              <td>
                <router-link :to="`/repository/${p.id}`" class="app-link whitespace-nowrap">
                  {{ p.name }}
                </router-link>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ p.code }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ p.repository_type === 'local_directory' ? '本地目录' : '远程 Git' }}
              </td>
              <td class="whitespace-nowrap text-foreground">{{ p.default_branch }}</td>
              <td class="max-w-md truncate text-foreground" :title="repositoryLocation(p)">
                {{ repositoryLocation(p) }}
              </td>
              <td class="whitespace-nowrap text-foreground">
                <router-link
                  v-if="p.git_credential_id"
                  :to="`/credential/${p.git_credential_id}`"
                  class="app-link"
                >
                  已配置
                </router-link>
                <span v-else-if="p.has_credential">已配置</span>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(p.created_at) }}</td>
              <td class="whitespace-nowrap">
                <button class="app-link mr-3" @click="openEditModal(p)">编辑</button>
                <button class="app-link-danger" @click="openDeleteDialog(p)">删除</button>
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

  <AppDialog :open="showCreateModal" title="创建仓库" @update:open="handleCreateModalOpenChange">
    <AppLoadingState v-if="modalStatus === 'loading'" size="compact" />

    <div v-else class="space-y-4">
      <div class="space-y-1.5">
        <label for="create-repository-type" class="app-field-label block">仓库类型</label>
        <SelectControl
          id="create-repository-type"
          :model-value="form.repository_type"
          :options="repositoryTypeOptions"
          @update:model-value="updateCreateRepositoryType"
        />
      </div>

      <div class="space-y-1.5">
        <label for="create-repository-name" class="app-field-label block">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          id="create-repository-name"
          v-model="form.name"
          type="text"
          placeholder="例如: my-backend"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
          :aria-invalid="errors.name ? 'true' : undefined"
          :aria-describedby="errors.name ? 'create-repository-name-error' : undefined"
          @input="clearError('name')"
        />
        <p
          v-if="errors.name"
          id="create-repository-name-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.name }}
        </p>
      </div>

      <div class="space-y-1.5">
        <label for="create-repository-code" class="app-field-label block">
          仓库编码
          <span class="text-destructive">*</span>
        </label>
        <input
          id="create-repository-code"
          v-model="form.code"
          type="text"
          placeholder="例如: my-backend（固化工作目录，创建后不可修改）"
          class="app-input"
          :class="errors.code ? 'app-input-error' : ''"
          :aria-invalid="errors.code ? 'true' : undefined"
          :aria-describedby="errors.code ? 'create-repository-code-error' : undefined"
          @input="clearError('code')"
        />
        <p
          v-if="errors.code"
          id="create-repository-code-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.code }}
        </p>
      </div>

      <div v-if="form.repository_type === 'remote_git'" class="space-y-1.5">
        <label for="create-repository-url" class="app-field-label block">
          仓库地址
          <span class="text-destructive">*</span>
        </label>
        <input
          id="create-repository-url"
          v-model="form.repository_url"
          type="text"
          placeholder="git@github.com:user/repo.git"
          class="app-input"
          :class="errors.repository_url ? 'app-input-error' : ''"
          :aria-invalid="errors.repository_url ? 'true' : undefined"
          :aria-describedby="errors.repository_url ? 'create-repository-url-error' : undefined"
          @input="clearError('repository_url')"
        />
        <p
          v-if="errors.repository_url"
          id="create-repository-url-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.repository_url }}
        </p>
      </div>

      <div v-else class="space-y-1.5">
        <label for="create-repository-url" class="app-field-label block">
          本地目录
          <span class="text-destructive">*</span>
        </label>
        <input
          id="create-repository-url"
          v-model="form.repository_url"
          type="text"
          placeholder="例如: D:/workspace/my-backend"
          class="app-input"
          :class="errors.repository_url ? 'app-input-error' : ''"
          :aria-invalid="errors.repository_url ? 'true' : undefined"
          :aria-describedby="errors.repository_url ? 'create-repository-url-error' : undefined"
          @input="clearError('repository_url')"
        />
        <p
          v-if="errors.repository_url"
          id="create-repository-url-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.repository_url }}
        </p>
      </div>

      <div v-if="form.repository_type === 'remote_git'" class="space-y-1.5">
        <label class="app-field-label block">Git 凭据</label>
        <ComboboxSelect
          v-model="form.git_credential_id"
          :options="gitCredentialOptions"
          placeholder="不使用凭据"
        />
      </div>

      <div class="space-y-1.5">
        <label class="app-field-label block">默认分支</label>
        <input v-model="form.default_branch" type="text" placeholder="develop" class="app-input" />
      </div>

      <p v-if="createFormError" class="app-field-error text-xs" role="alert">
        {{ createFormError }}
      </p>
    </div>

    <template #footer>
      <AppDialogActions
        :busy="operating || modalStatus === 'loading'"
        @cancel="closeCreateModal"
        @confirm="handleCreateOk"
      />
    </template>
  </AppDialog>

  <AppDialog :open="showEditModal" title="编辑仓库" @update:open="handleEditModalOpenChange">
    <AppLoadingState v-if="modalStatus === 'loading'" size="compact" />

    <form v-else class="space-y-4" novalidate @submit.prevent="handleEditOk">
      <div class="space-y-1.5">
        <label for="edit-repository-type" class="app-field-label block">仓库类型</label>
        <SelectControl
          id="edit-repository-type"
          :model-value="editForm.repository_type"
          :options="repositoryTypeOptions"
          @update:model-value="updateEditRepositoryType"
        />
      </div>

      <div class="space-y-1.5">
        <label for="edit-repository-name" class="app-field-label block">
          名称
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-repository-name"
          v-model="editForm.name"
          type="text"
          class="app-input"
          :class="editErrors.name ? 'app-input-error' : ''"
          :aria-invalid="editErrors.name ? 'true' : undefined"
          :aria-describedby="editErrors.name ? 'edit-repository-name-error' : undefined"
          @input="clearEditError('name')"
        />
        <p
          v-if="editErrors.name"
          id="edit-repository-name-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.name }}
        </p>
      </div>

      <div class="space-y-1.5">
        <label class="app-field-label block">仓库编码</label>
        <input :value="editingRepository?.code" type="text" disabled class="app-input" />
      </div>

      <div v-if="editForm.repository_type === 'remote_git'" class="space-y-1.5">
        <label for="edit-repository-url" class="app-field-label block">
          仓库地址
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-repository-url"
          v-model="editForm.repository_url"
          type="text"
          placeholder="git@github.com:user/repo.git"
          class="app-input"
          :class="editErrors.repository_url ? 'app-input-error' : ''"
          :aria-invalid="editErrors.repository_url ? 'true' : undefined"
          :aria-describedby="editErrors.repository_url ? 'edit-repository-url-error' : undefined"
          @input="clearEditError('repository_url')"
        />
        <p
          v-if="editErrors.repository_url"
          id="edit-repository-url-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.repository_url }}
        </p>
      </div>

      <div v-else class="space-y-1.5">
        <label for="edit-repository-url" class="app-field-label block">
          本地目录
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-repository-url"
          v-model="editForm.repository_url"
          type="text"
          placeholder="例如: D:/workspace/my-backend"
          class="app-input"
          :class="editErrors.repository_url ? 'app-input-error' : ''"
          :aria-invalid="editErrors.repository_url ? 'true' : undefined"
          :aria-describedby="editErrors.repository_url ? 'edit-repository-url-error' : undefined"
          @input="clearEditError('repository_url')"
        />
        <p
          v-if="editErrors.repository_url"
          id="edit-repository-url-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.repository_url }}
        </p>
      </div>

      <div v-if="editForm.repository_type === 'remote_git'" class="space-y-1.5">
        <label class="app-field-label block">Git 凭据</label>
        <ComboboxSelect
          v-model="editForm.git_credential_id"
          :options="gitCredentialOptions"
          placeholder="不使用凭据"
        />
      </div>

      <div class="space-y-1.5">
        <label for="edit-repository-default-branch" class="app-field-label block">默认分支</label>
        <input
          id="edit-repository-default-branch"
          v-model="editForm.default_branch"
          type="text"
          placeholder="master"
          class="app-input"
        />
      </div>

      <p v-if="editFormError" class="app-field-error text-xs" role="alert">
        {{ editFormError }}
      </p>
    </form>

    <template #footer>
      <AppDialogActions
        :busy="operating || modalStatus === 'loading'"
        @cancel="closeEditModal"
        @confirm="handleEditOk"
      />
    </template>
  </AppDialog>

  <AppDialog
    v-model:open="showDeleteDialog"
    title="删除仓库"
    width-class="w-[min(420px,calc(100vw-32px))]"
  >
    <p class="text-sm text-foreground">
      确定要删除仓库「
      <span>{{ pendingDelete?.name ?? '' }}</span>
      」吗？此操作不可撤销。
    </p>
    <label class="mt-4 flex cursor-pointer items-center gap-2">
      <input v-model="deleteWorkspace" type="checkbox" class="size-4 accent-destructive" />
      <span class="text-sm text-foreground">
        同时删除由 CI 工作区管理的源码目录（{{ pendingDelete?.code }}）
      </span>
    </label>
    <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
      {{ deleteSubmitError }}
    </p>
    <template #footer>
      <AppDialogActions
        :busy="operating"
        variant="destructive"
        @cancel="showDeleteDialog = false"
        @confirm="handleDeleteOk"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { Plus } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { credentialApi } from '@/api/credential/credential';
  import { repositoryApi } from '@/api/repository/repository';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { CredentialResp } from '@/gen/proto/orbit/v1/credential/credential';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import { formatTime } from '@/utils/time';
  import {
    repositoryFormFeedback,
    type RepositoryFormErrors,
    type RepositoryFormFields,
    validateRepositoryForm,
  } from '@/views/repository/repositoryForm';
  import { ToolbarRoot } from 'reka-ui';

  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { status: modalStatus, execute: executeModal } = useStatusAsync();

  const repositories = ref<RepositoryResp[]>([]);
  const credentials = ref<CredentialResp[]>([]);
  const gitCredentials = computed(() =>
    credentials.value.filter(
      (c) =>
        c.type === 'git_ssh' ||
        c.type === 'github_token' ||
        c.type === 'gitee_token' ||
        c.type === 'gitea_token'
    )
  );
  const gitCredentialOptions = computed(() =>
    gitCredentials.value.map((cred) => ({
      value: cred.id,
      label: cred.name,
    }))
  );
  const repositoryTypeOptions = [
    { value: 'remote_git', label: '远程 Git' },
    { value: 'local_directory', label: '本地目录' },
  ];
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const searchText = ref('');
  const showCreateModal = ref(false);
  const showEditModal = ref(false);
  const showDeleteDialog = ref(false);
  const pendingDelete = ref<RepositoryResp>();
  const deleteWorkspace = ref(false);
  const deleteSubmitError = ref('');
  const editingRepository = ref<RepositoryResp>();
  const createFormError = ref('');
  const editFormError = ref('');

  const form = reactive({
    name: '',
    code: '',
    repository_type: 'remote_git',
    repository_url: '',
    git_credential_id: '',
    default_branch: 'develop',
  });
  const errors = reactive<Record<RepositoryFormFields, string>>({
    name: '',
    code: '',
    repository_url: '',
  });
  const editForm = reactive({
    name: '',
    repository_type: 'remote_git',
    repository_url: '',
    git_credential_id: '',
    default_branch: 'master',
  });
  const editErrors = reactive<Record<RepositoryFormFields, string>>({
    name: '',
    code: '',
    repository_url: '',
  });

  function validate() {
    applyErrors(validateRepositoryForm(form, { requireCode: true }));
    return !Object.values(errors).some(Boolean);
  }

  function applyErrors(next: RepositoryFormErrors) {
    Object.assign(errors, { name: '', code: '', repository_url: '' }, next);
  }

  function resetCreateFormFeedback() {
    applyErrors({});
    createFormError.value = '';
  }

  function clearError(field: RepositoryFormFields) {
    errors[field] = '';
  }

  function handleRepositoryTypeChange() {
    clearError('repository_url');
    createFormError.value = '';
  }

  function updateCreateRepositoryType(value: string | number) {
    if (value !== 'remote_git' && value !== 'local_directory') {
      return;
    }
    form.repository_type = value;
    handleRepositoryTypeChange();
  }

  function resetEditForm() {
    if (!editingRepository.value) {
      return;
    }
    Object.assign(editForm, {
      name: editingRepository.value.name,
      repository_type: editingRepository.value.repository_type || 'remote_git',
      repository_url: editingRepository.value.repository_url,
      git_credential_id: editingRepository.value.git_credential_id ?? '',
      default_branch: editingRepository.value.default_branch || 'master',
    });
    resetEditFormFeedback();
  }

  function validateEditForm() {
    applyEditErrors(validateRepositoryForm(editForm, { requireCode: false }));
    return !Object.values(editErrors).some(Boolean);
  }

  function applyEditErrors(next: RepositoryFormErrors) {
    Object.assign(editErrors, { name: '', code: '', repository_url: '' }, next);
  }

  function resetEditFormFeedback() {
    applyEditErrors({});
    editFormError.value = '';
  }

  function clearEditError(field: RepositoryFormFields) {
    editErrors[field] = '';
  }

  function handleEditRepositoryTypeChange() {
    clearEditError('repository_url');
    editFormError.value = '';
  }

  function updateEditRepositoryType(value: string | number) {
    if (value !== 'remote_git' && value !== 'local_directory') {
      return;
    }
    editForm.repository_type = value;
    handleEditRepositoryTypeChange();
  }

  function handleCreateModalOpenChange(open: boolean) {
    showCreateModal.value = open;
    if (!open) {
      resetCreateFormFeedback();
    }
  }

  function closeCreateModal() {
    handleCreateModalOpenChange(false);
  }

  function handleEditModalOpenChange(open: boolean) {
    showEditModal.value = open;
    if (!open) {
      resetEditFormFeedback();
    }
  }

  function closeEditModal() {
    handleEditModalOpenChange(false);
  }

  function applyCreateFailure(error: unknown) {
    const feedback = repositoryFormFeedback(error);
    if (!feedback) {
      return false;
    }
    if (feedback.kind === 'field') {
      applyErrors(feedback.errors);
    } else {
      createFormError.value = feedback.message;
    }
    return true;
  }

  function applyEditFailure(error: unknown) {
    const feedback = repositoryFormFeedback(error);
    if (!feedback) {
      return false;
    }
    if (feedback.kind === 'field') {
      applyEditErrors(feedback.errors);
    } else {
      editFormError.value = feedback.message;
    }
    return true;
  }

  async function fetchProjects() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await execute(async () => {
        const res = await repositoryApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
          project_id: projectId,
        });
        repositories.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error('获取项目列表失败');
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchProjects();
  }

  function goPage(p: number) {
    if (p < 1 || p > totalPages.value || p === pagination.current) {
      return;
    }
    pagination.current = p;
    fetchProjects();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchProjects();
  }

  async function openCreateModal() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    Object.assign(form, {
      name: '',
      code: '',
      repository_type: 'remote_git',
      repository_url: '',
      git_credential_id: '',
      default_branch: 'develop',
    });
    resetCreateFormFeedback();
    showCreateModal.value = true;
    try {
      await executeModal(async () => {
        const credRes = await credentialApi.list({
          per_page: 100,
          project_id: projectId,
        });
        credentials.value = credRes.items;
      });
    } catch {
      toast.error('加载表单数据失败');
    }
  }

  async function openEditModal(repository: RepositoryResp) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    editingRepository.value = repository;
    resetEditForm();
    showEditModal.value = true;
    try {
      await executeModal(async () => {
        const credRes = await credentialApi.list({
          per_page: 100,
          project_id: projectId,
        });
        credentials.value = credRes.items;
      });
    } catch {
      toast.error('加载表单数据失败');
    }
  }

  async function handleCreateOk() {
    createFormError.value = '';
    if (!validate()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      createFormError.value = '请先选择项目';
      return;
    }
    try {
      await executeOp(async () => {
        const repository = await repositoryApi.create(
          {
            name: form.name,
            code: form.code,
            repository_type: form.repository_type,
            repository_url: form.repository_url,
            git_credential_id:
              form.repository_type === 'remote_git'
                ? form.git_credential_id || undefined
                : undefined,
            variable_overrides: [],
            default_branch: form.default_branch,
          },
          { project_id: projectId }
        );
        toast.success('创建成功');
        showCreateModal.value = false;
        router.push(`/repository/${repository.id}`);
      });
    } catch (error) {
      if (!applyCreateFailure(error)) {
        createFormError.value = error instanceof Error ? error.message : '创建失败';
      }
    }
  }

  async function handleEditOk() {
    const repository = editingRepository.value;
    editFormError.value = '';
    if (!repository || !validateEditForm()) {
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await repositoryApi.update(repository.id, {
          name: editForm.name,
          repository_type: editForm.repository_type,
          repository_url: editForm.repository_url,
          git_credential_id:
            editForm.repository_type === 'remote_git' ? editForm.git_credential_id : '',
          default_branch: editForm.default_branch || 'master',
        });
        repositories.value = repositories.value.map((repository) =>
          repository.id === updated.id ? updated : repository
        );
        editingRepository.value = updated;
        toast.success('更新成功');
        closeEditModal();
      });
    } catch (error) {
      if (!applyEditFailure(error)) {
        editFormError.value = error instanceof Error ? error.message : '更新失败';
      }
    }
  }

  function openDeleteDialog(repository: RepositoryResp) {
    pendingDelete.value = repository;
    deleteWorkspace.value = false;
    deleteSubmitError.value = '';
    showDeleteDialog.value = true;
  }

  async function handleDeleteOk() {
    const repository = pendingDelete.value;
    if (!repository) {
      return;
    }
    deleteSubmitError.value = '';
    try {
      await executeOp(async () => {
        await repositoryApi.delete(repository.id, {
          delete_workspace: deleteWorkspace.value,
        });
        toast.success('删除成功');
        showDeleteDialog.value = false;
        pendingDelete.value = undefined;
        if (repositories.value.length === 1 && pagination.current > 1) {
          pagination.current -= 1;
        }
        await fetchProjects();
      });
    } catch (error) {
      deleteSubmitError.value = error instanceof Error ? error.message : '删除失败';
    }
  }

  function repositoryLocation(repository: RepositoryResp) {
    return repository.repository_url;
  }

  onMounted(fetchProjects);
</script>
