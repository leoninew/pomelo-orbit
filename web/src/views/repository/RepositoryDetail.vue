<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader :items="[]" :title="repository?.name ?? '仓库详情'" />
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="repository"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="openDeleteDialog"
        >
          <Trash2 class="size-4" />
          删除
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/repository')">
          <ArrowLeft class="size-4" />
          返回
        </button>
      </div>
    </div>

    <AppLoadingState v-if="status === 'loading'" size="section" />

    <template v-else-if="repository">
      <DetailInfoCard title="基本信息" editable :disabled="operating" @edit="openEditDialog">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>名称</dt>
            <dd class="min-w-0 text-foreground">{{ repository.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>编码</dt>
            <dd class="min-w-0 text-foreground">{{ repository.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>仓库类型</dt>
            <dd class="text-foreground">
              {{ repository.repository_type === 'local_directory' ? '本地目录' : '远程 Git' }}
            </dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt>
              {{ repository.repository_type === 'local_directory' ? '本地目录' : '仓库地址' }}
            </dt>
            <dd class="min-w-0 truncate text-foreground" :title="repositoryLocation(repository)">
              {{ repositoryLocation(repository) }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>默认分支</dt>
            <dd class="text-foreground">{{ repository.default_branch || 'master' }}</dd>
          </div>
          <div v-if="repository.repository_type !== 'local_directory'" class="flex gap-2">
            <dt>Git 凭据</dt>
            <dd>
              <router-link
                v-if="repository.git_credential_id"
                :to="`/credential/${repository.git_credential_id}`"
                class="app-link"
              >
                {{ repository.git_credential_name }}
              </router-link>
              <span v-else class="text-muted-foreground">未配置</span>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>流水线记录</dt>
            <dd>
              <router-link :to="`/pipeline-run?repository_id=${repository.id}`" class="app-link">
                查看所有记录
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(repository.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>更新时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(repository.updated_at) }}</dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard title="变量配置">
        <template #actions>
          <button class="app-button-primary h-9 px-3" @click="openAddVariableDialog">
            <Plus class="size-4" />
            添加自定义变量
          </button>
        </template>
        <VariableDeclarationsTable
          :declarations="repositoryVariableRows"
          :readonly="false"
          @edit="openEditVariableDialog"
          @delete="deleteVariable"
        />
      </DetailInfoCard>
    </template>

    <AppDialog :open="isEditDialogOpen" title="编辑仓库" @update:open="handleEditDialogOpenChange">
      <div class="space-y-4">
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
          <label class="app-field-label block">编码</label>
          <input :value="repository?.code" type="text" disabled class="app-input" />
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
            description-inline
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">默认分支</label>
          <input
            v-model="editForm.default_branch"
            type="text"
            placeholder="master"
            class="app-input"
          />
        </div>
        <p v-if="editFormError" class="app-field-error text-xs" role="alert">
          {{ editFormError }}
        </p>
      </div>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="closeEditDialog" @confirm="handleEditOk" />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      title="删除仓库"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        确定要删除仓库「
        <span>{{ repository?.name ?? '' }}</span>
        」吗？此操作不可撤销。
      </p>
      <label class="mt-4 flex cursor-pointer items-center gap-2">
        <input v-model="deleteWorkspace" type="checkbox" class="size-4 accent-destructive" />
        <span class="text-sm text-foreground">
          同时删除由 CI 工作区管理的源码目录（{{ repository?.code }}）
        </span>
      </label>
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="handleDeleteOk"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isAddVariableDialogOpen" title="添加自定义变量">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            变量名
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="variableForm.name"
            type="text"
            placeholder="例如: DEPLOY_ENV"
            class="app-input"
            :class="variableErrors.name ? 'app-input-error' : ''"
            :aria-invalid="variableErrors.name ? 'true' : undefined"
            @input="variableErrors.name = ''"
          />
          <p v-if="variableErrors.name" class="app-field-error text-xs">
            {{ variableErrors.name }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">变量值</label>
          <input
            v-model="variableForm.value"
            :type="variableForm.secret ? 'password' : 'text'"
            class="app-input"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">说明</label>
          <input v-model="variableForm.description" type="text" class="app-input" />
        </div>
        <label class="flex items-center gap-2 text-sm text-foreground">
          <input v-model="variableForm.secret" type="checkbox" class="app-checkbox" />
          敏感变量
        </label>
      </div>
      <p v-if="addVariableSubmitError" class="app-field-error mt-3" role="alert">
        {{ addVariableSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isAddVariableDialogOpen = false"
          @confirm="handleAddVariableOk"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isEditVariableDialogOpen" title="编辑自定义变量">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            变量名
            <span class="text-destructive">*</span>
          </label>
          <input :value="editingVariableName" type="text" disabled class="app-input" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">变量值</label>
          <input
            v-model="variableForm.value"
            :type="variableForm.secret ? 'password' : 'text'"
            class="app-input"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">说明</label>
          <input v-model="variableForm.description" type="text" class="app-input" />
        </div>
        <label class="flex items-center gap-2 text-sm text-foreground">
          <input v-model="variableForm.secret" type="checkbox" class="app-checkbox" />
          敏感变量
        </label>
      </div>
      <p v-if="editVariableSubmitError" class="app-field-error mt-3" role="alert">
        {{ editVariableSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isEditVariableDialogOpen = false"
          @confirm="handleEditVariableOk"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Plus, Trash2 } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { credentialApi } from '@/api/credential/credential';
  import { repositoryApi } from '@/api/repository/repository';
  import AppDialog from '@/components/AppDialog.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { CredentialResp } from '@/gen/proto/orbit/v1/credential/credential';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import type { VariableDeclarationResp } from '@/gen/proto/orbit/v1/common/common';
  import { formatTime } from '@/utils/time';
  import {
    repositoryFormFeedback,
    type RepositoryFormErrors,
    type RepositoryFormFields,
    validateRepositoryForm,
  } from '@/views/repository/repositoryForm';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  const route = useRoute();
  const router = useRouter();
  const repositoryId = route.params.id as string;
  const toast = useToast();
  const projectStore = useProjectStore();

  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const repository = ref<RepositoryResp>();
  const credentials = ref<CredentialResp[]>([]);

  const isEditDialogOpen = ref(false);
  const editFormError = ref('');
  const isDeleteDialogOpen = ref(false);
  const deleteSubmitError = ref('');
  const isAddVariableDialogOpen = ref(false);
  const addVariableSubmitError = ref('');
  const isEditVariableDialogOpen = ref(false);
  const editVariableSubmitError = ref('');
  const deleteWorkspace = ref(false);

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
  const variableForm = reactive({
    name: '',
    value: '',
    description: '',
    secret: false,
  });
  const variableErrors = reactive({
    name: '',
  });
  const editingVariableName = ref('');

  const gitCredentials = computed(() =>
    credentials.value.filter(
      (credential) =>
        credential.type === 'git_ssh' ||
        credential.type === 'github_token' ||
        credential.type === 'gitee_token' ||
        credential.type === 'gitea_token'
    )
  );
  const gitCredentialOptions = computed(() =>
    gitCredentials.value.map((credential) => ({
      value: credential.id,
      label: credential.name,
      description: credential.type,
    }))
  );
  const repositoryTypeOptions = [
    { value: 'remote_git', label: '远程 Git' },
    { value: 'local_directory', label: '本地目录' },
  ];
  const repositoryVariables = computed(() => repository.value?.variable_declarations ?? []);
  const repositoryCustomVariables = computed(() =>
    repositoryVariables.value.filter((variable) => variable.source === 'repository_custom')
  );
  const repositoryVariableRows = computed(() =>
    repositoryVariables.value.map((variable) => ({
      ...variable,
      editable: variable.source === 'repository_custom',
    }))
  );

  function normalizeValue(value: unknown) {
    return value == null ? '' : String(value);
  }

  function resetEditForm() {
    if (!repository.value) {
      return;
    }
    Object.assign(editForm, {
      name: repository.value.name,
      repository_type: repository.value.repository_type || 'remote_git',
      repository_url: repository.value.repository_url,
      git_credential_id: repository.value.git_credential_id ?? '',
      default_branch: repository.value.default_branch || 'master',
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

  function handleEditDialogOpenChange(open: boolean) {
    isEditDialogOpen.value = open;
    if (!open) {
      resetEditFormFeedback();
    }
  }

  function closeEditDialog() {
    handleEditDialogOpenChange(false);
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

  function variableOverridesWith(nextVariable?: VariableDeclarationResp) {
    const next = repositoryCustomVariables.value.filter(
      (variable) => variable.name !== nextVariable?.name
    );
    return nextVariable ? [...next, nextVariable] : next;
  }

  async function fetchRepository() {
    try {
      await execute(async () => {
        repository.value = await repositoryApi.get(repositoryId);
        resetEditForm();
      });
    } catch {
      toast.error('获取代码仓库信息失败');
      router.push('/repository');
    }
  }

  async function fetchCredentials() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      const res = await credentialApi.list({ per_page: 100, project_id: projectId });
      credentials.value = res.items;
    } catch {
      toast.error('获取凭据列表失败');
    }
  }

  async function openEditDialog() {
    resetEditForm();
    await fetchCredentials();
    isEditDialogOpen.value = true;
  }

  async function handleEditOk() {
    editFormError.value = '';
    if (!validateEditForm()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      editFormError.value = '请先选择项目';
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await repositoryApi.update(repositoryId, {
          name: editForm.name,
          repository_type: editForm.repository_type,
          repository_url: editForm.repository_url,
          git_credential_id:
            editForm.repository_type === 'remote_git' ? editForm.git_credential_id : '',
          default_branch: editForm.default_branch || 'master',
        });
        repository.value = updated;
        resetEditForm();
        toast.success('更新成功');
        isEditDialogOpen.value = false;
      });
    } catch (error) {
      if (!applyEditFailure(error)) {
        editFormError.value = error instanceof Error ? error.message : '更新失败';
      }
    }
  }

  function openDeleteDialog() {
    deleteWorkspace.value = false;
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  async function handleDeleteOk() {
    deleteSubmitError.value = '';
    try {
      await executeOp(async () => {
        await repositoryApi.delete(repositoryId, {
          delete_workspace: deleteWorkspace.value,
        });
        toast.success('删除成功');
        router.push('/repository');
      });
    } catch (error) {
      deleteSubmitError.value = error instanceof Error ? error.message : '删除失败';
    }
  }

  function openAddVariableDialog() {
    Object.assign(variableForm, { name: '', value: '', description: '', secret: false });
    Object.assign(variableErrors, { name: '' });
    addVariableSubmitError.value = '';
    isAddVariableDialogOpen.value = true;
  }

  function openEditVariableDialog(name: string) {
    const variable = repositoryCustomVariables.value.find((item) => item.name === name);
    if (!variable) {
      return;
    }
    editingVariableName.value = variable.name;
    Object.assign(variableForm, {
      name: variable.name,
      value: normalizeValue(variable.value ?? variable.default),
      description: variable.description ?? '',
      secret: variable.secret,
    });
    Object.assign(variableErrors, { name: '' });
    editVariableSubmitError.value = '';
    isEditVariableDialogOpen.value = true;
  }

  async function handleAddVariableOk() {
    addVariableSubmitError.value = '';
    variableErrors.name = '';
    if (!variableForm.name.trim()) {
      variableErrors.name = '请输入变量名';
      return;
    }
    if (repositoryVariables.value.some((variable) => variable.name === variableForm.name.trim())) {
      variableErrors.name = '变量名已存在';
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      addVariableSubmitError.value = '请先选择项目';
      return;
    }
    try {
      await executeOp(async () => {
        const nextVariable: VariableDeclarationResp = {
          name: variableForm.name.trim(),
          description: variableForm.description.trim(),
          default: undefined,
          value: variableForm.value,
          secret: variableForm.secret,
          source: 'repository_custom',
          editable: true,
        };
        const updated = await repositoryApi.update(repositoryId, {
          variable_overrides: { items: variableOverridesWith(nextVariable) },
        });
        repository.value = updated;
        toast.success('变量已添加');
        isAddVariableDialogOpen.value = false;
      });
    } catch (error) {
      addVariableSubmitError.value = error instanceof Error ? error.message : '添加失败';
    }
  }

  async function handleEditVariableOk() {
    if (!editingVariableName.value) {
      return;
    }
    editVariableSubmitError.value = '';
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      editVariableSubmitError.value = '请先选择项目';
      return;
    }
    try {
      await executeOp(async () => {
        const current = repositoryCustomVariables.value.find(
          (variable) => variable.name === editingVariableName.value
        );
        if (!current) {
          return;
        }
        const updated = await repositoryApi.update(repositoryId, {
          variable_overrides: {
            items: variableOverridesWith({
              ...current,
              value: variableForm.value,
              description: variableForm.description.trim(),
              secret: variableForm.secret,
            }),
          },
        });
        repository.value = updated;
        toast.success('变量已更新');
        isEditVariableDialogOpen.value = false;
      });
    } catch (error) {
      editVariableSubmitError.value = error instanceof Error ? error.message : '更新失败';
    }
  }

  async function deleteVariable(name: string) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await repositoryApi.update(repositoryId, {
          variable_overrides: {
            items: repositoryCustomVariables.value.filter((variable) => variable.name !== name),
          },
        });
        repository.value = updated;
        toast.success('变量已重置');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    }
  }

  function repositoryLocation(value: RepositoryResp) {
    return value.repository_url;
  }

  onMounted(async () => {
    await fetchRepository();
  });
</script>
