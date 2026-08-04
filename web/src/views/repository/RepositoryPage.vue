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
        <table class="app-data-table min-w-[960px]">
          <thead>
            <tr>
              <th>名称</th>
              <th>编码</th>
              <th>仓库类型</th>
              <th>默认分支</th>
              <th>地址/目录</th>
              <th>Git 凭据</th>
              <th>创建时间</th>
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
        <select
          id="create-repository-type"
          v-model="form.repository_type"
          class="app-input"
          @change="handleRepositoryTypeChange"
        >
          <option value="remote_git">远程 Git</option>
          <option value="local_directory">本地目录</option>
        </select>
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
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
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
      (c) => c.type === 'git_ssh' || c.type === 'github_token' || c.type === 'gitee_token'
    )
  );
  const gitCredentialOptions = computed(() =>
    gitCredentials.value.map((cred) => ({
      value: cred.id,
      label: cred.name,
    }))
  );
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const searchText = ref('');
  const showCreateModal = ref(false);
  const createFormError = ref('');

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

  function handleCreateModalOpenChange(open: boolean) {
    showCreateModal.value = open;
    if (!open) {
      resetCreateFormFeedback();
    }
  }

  function closeCreateModal() {
    handleCreateModalOpenChange(false);
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

  async function handleCreateOk() {
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
        toast.error(error instanceof Error ? error.message : '创建失败');
      }
    }
  }

  function repositoryLocation(repository: RepositoryResp) {
    return repository.repository_url;
  }

  onMounted(fetchProjects);
</script>
