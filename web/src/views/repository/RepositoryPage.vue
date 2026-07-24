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
        新建仓库
      </button>
    </ToolbarRoot>

    <!-- Table Card -->
    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="repositories.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[1120px]">
          <thead>
            <tr>
              <th>名称</th>
              <th>编码</th>
              <th>默认分支</th>
              <th>地址</th>
              <th>Git 凭据</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in repositories" :key="p.id">
              <td>
                <button
                  class="app-link whitespace-nowrap"
                  @click="router.push(`/repository/${p.id}`)"
                >
                  {{ p.name }}
                </button>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ p.code }}</td>
              <td class="whitespace-nowrap text-foreground">{{ p.default_branch || '—' }}</td>
              <td class="max-w-md truncate text-foreground" :title="p.repository_url">
                {{ p.repository_url }}
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
                <span v-else class="text-muted-foreground">—</span>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(p.created_at) }}</td>
              <td class="whitespace-nowrap">
                <button class="app-link" @click="router.push(`/repository/${p.id}`)">查看</button>
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

  <AppDialog v-model:open="showCreateModal" title="新建仓库">
    <AppSpinner v-if="modalStatus === 'loading'" class="py-8" />

    <div v-else class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">名称</label>
        <input
          v-model="form.name"
          type="text"
          placeholder="例如: my-backend"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
        />
        <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
      </div>

      <div class="space-y-1.5">
        <label class="app-field-label block">仓库编码</label>
        <input
          v-model="form.code"
          type="text"
          placeholder="例如: my-backend（固化工作目录，创建后不可修改）"
          class="app-input"
          :class="errors.code ? 'app-input-error' : ''"
        />
        <p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
      </div>

      <div class="space-y-1.5">
        <label class="app-field-label block">仓库地址</label>
        <input
          v-model="form.repository_url"
          type="text"
          placeholder="git@github.com:user/repo.git"
          class="app-input"
          :class="errors.repository_url ? 'app-input-error' : ''"
        />
        <p v-if="errors.repository_url" class="app-field-error text-xs">
          {{ errors.repository_url }}
        </p>
      </div>

      <div class="space-y-1.5">
        <label class="app-field-label block">Git 凭据（可选）</label>
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
    </div>

    <template #footer>
      <button class="app-button" @click="showCreateModal = false">取消</button>
      <button
        class="app-button-primary"
        :disabled="operating || modalStatus === 'loading'"
        @click="handleCreateOk"
      >
        创建
      </button>
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
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { CredentialResp } from '@/gen/proto/orbit/v1/credential/credential';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import { formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
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

  const form = reactive({
    name: '',
    code: '',
    repository_url: '',
    git_credential_id: '',
    default_branch: 'develop',
  });
  const errors = reactive({
    name: '',
    code: '',
    repository_url: '',
  });

  function validate() {
    errors.name = form.name.trim() ? '' : '请输入名称';
    errors.code = /^[a-z0-9-]+$/.test(form.code.trim()) ? '' : '编码只能包含小写字母、数字和连字符';
    errors.repository_url = form.repository_url.trim() ? '' : '请输入仓库地址';
    return !errors.name && !errors.code && !errors.repository_url;
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
      repository_url: '',
      git_credential_id: '',
      default_branch: 'develop',
    });
    Object.assign(errors, { name: '', code: '', repository_url: '' });
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
            repository_url: form.repository_url,
            git_credential_id: form.git_credential_id || undefined,
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
      toast.error(error instanceof Error ? error.message : '创建失败');
    }
  }

  onMounted(fetchProjects);
</script>
