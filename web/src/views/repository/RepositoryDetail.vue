<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div>
          <h1 class="text-xl font-semibold text-foreground">
            {{ repository?.name ?? '仓库详情' }}
          </h1>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="repository"
          class="app-button-primary h-9 px-3"
          :disabled="operating"
          @click="openEditDialog"
        >
          <Pencil class="size-4" />
          编辑
        </button>
        <button
          v-if="repository"
          class="app-button-primary h-9 px-3"
          :disabled="operating"
          @click="openTriggerModal"
        >
          <Play class="size-4" />
          触发
        </button>
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

    <AppSpinner v-if="status === 'loading'" class="py-12" />

    <template v-else-if="repository">
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">基本信息</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">名称</dt>
            <dd class="min-w-0 text-foreground">{{ repository.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">编码</dt>
            <dd class="min-w-0 text-foreground">{{ repository.code }}</dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt class="w-32 shrink-0 text-muted-foreground">仓库地址</dt>
            <dd class="min-w-0 truncate text-foreground" :title="repository.repository_url">
              {{ repository.repository_url }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">默认分支</dt>
            <dd class="text-foreground">{{ repository.default_branch || 'master' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">Git 凭据</dt>
            <dd>
              <router-link
                v-if="repository.git_credential_id"
                :to="`/credential/${repository.git_credential_id}`"
                class="app-link"
              >
                {{ repository.git_credential_name || repository.git_credential_id }}
              </router-link>
              <span v-else class="text-muted-foreground">未配置</span>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">流水线记录</dt>
            <dd>
              <router-link :to="`/pipeline-run?repository_id=${repository.id}`" class="app-link">
                查看所有记录
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(repository.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">更新时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(repository.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <div class="app-surface">
        <div class="app-section-header flex flex-wrap items-center justify-between gap-3">
          <h2 class="font-semibold text-foreground">变量配置</h2>
          <button class="app-button-primary h-8 px-3" @click="openAddVariableDialog">
            <Plus class="size-4" />
            添加自定义变量
          </button>
        </div>
        <VariableDeclarationsTable
          :declarations="repositoryVariableRows"
          :readonly="false"
          @edit="openEditVariableDialog"
          @delete="deleteVariable"
        />
      </div>

      <WebhookList
        :repository-id="repositoryId"
        :webhooks="webhooks"
        :templates="templates"
        @refresh="fetchWebhooks"
      />
    </template>

    <TriggerModal
      ref="triggerModalRef"
      :repository-id="repositoryId"
      :templates="templates"
      :default-branch="repository?.default_branch"
      :project-variables="repositoryCustomVariables"
      :repository="repository"
      @trigger="handleTrigger"
    />

    <AppDialog v-model:open="isEditDialogOpen" title="编辑仓库">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">名称</label>
          <input
            v-model="editForm.name"
            type="text"
            class="app-input"
            :class="editErrors.name ? 'app-input-error' : ''"
          />
          <p v-if="editErrors.name" class="app-field-error text-xs">{{ editErrors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">编码</label>
          <input :value="repository?.code" type="text" disabled class="app-input" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">仓库地址</label>
          <input
            v-model="editForm.repository_url"
            type="text"
            class="app-input"
            :class="editErrors.repository_url ? 'app-input-error' : ''"
          />
          <p v-if="editErrors.repository_url" class="app-field-error text-xs">
            {{ editErrors.repository_url }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">Git 凭据</label>
          <ComboboxSelect
            v-model="editForm.git_credential_id"
            :options="gitCredentialOptions"
            placeholder="不使用凭据"
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
      </div>
      <template #footer>
        <button class="app-button" @click="isEditDialogOpen = false">取消</button>
        <button class="app-button-primary" :disabled="operating" @click="handleEditOk">保存</button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      title="删除仓库"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        确定要删除仓库「
        <strong>{{ repository?.name ?? '' }}</strong>
        」吗？此操作不可撤销。
      </p>
      <label class="mt-4 flex cursor-pointer items-center gap-2">
        <input v-model="deleteWorkspace" type="checkbox" class="size-4 accent-destructive" />
        <span class="text-sm text-foreground">
          同时删除工作目录（data/pipeline/{{ repository?.code }}）
        </span>
      </label>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">取消</button>
        <button class="app-button-destructive" :disabled="operating" @click="handleDeleteOk">
          删除
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isAddVariableDialogOpen" title="添加变量">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">变量名</label>
          <input
            v-model="variableForm.name"
            type="text"
            placeholder="例如: DEPLOY_ENV"
            class="app-input"
            :class="variableErrors.name ? 'app-input-error' : ''"
          />
          <p v-if="variableErrors.name" class="app-field-error text-xs">
            {{ variableErrors.name }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">变量值</label>
          <input v-model="variableForm.value" type="text" class="app-input" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">说明</label>
          <input
            v-model="variableForm.description"
            type="text"
            placeholder="可选"
            class="app-input"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isAddVariableDialogOpen = false">取消</button>
        <button class="app-button-primary" :disabled="operating" @click="handleAddVariableOk">
          添加
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isEditVariableDialogOpen" title="编辑变量">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">变量名</label>
          <input :value="editingVariableName" type="text" disabled class="app-input" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">变量值</label>
          <input v-model="variableForm.value" type="text" class="app-input" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">说明</label>
          <input
            v-model="variableForm.description"
            type="text"
            placeholder="可选"
            class="app-input"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isEditVariableDialogOpen = false">取消</button>
        <button class="app-button-primary" :disabled="operating" @click="handleEditVariableOk">
          保存
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Pencil, Play, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { credentialApi } from '@/api/credential/credential';
  import { pipelineTemplateApi } from '@/api/pipeline/template';
  import { repositoryApi } from '@/api/repository/repository';
  import { webhookApi } from '@/api/repository/webhook';
  import AppDialog from '@/components/AppDialog.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { CredentialResp } from '@/gen/proto/orbit/v1/credential/credential';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import type { PipelineTemplateResp } from '@/gen/proto/orbit/v1/pipeline/template';
  import type { RepositoryWebhookResp } from '@/gen/proto/orbit/v1/repository/webhook';
  import type { VariableDeclarationResp } from '@/gen/proto/orbit/v1/common/common';
  import { formatTime } from '@/utils/time';
  import TriggerModal from '@/views/repository/components/TriggerModal.vue';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';
  import WebhookList from '@/views/repository/components/WebhookList.vue';

  const route = useRoute();
  const router = useRouter();
  const repositoryId = route.params.id as string;
  const toast = useToast();
  const projectStore = useProjectStore();

  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const repository = ref<RepositoryResp>();
  const templates = ref<PipelineTemplateResp[]>([]);
  const webhooks = ref<RepositoryWebhookResp[]>([]);
  const credentials = ref<CredentialResp[]>([]);

  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isAddVariableDialogOpen = ref(false);
  const isEditVariableDialogOpen = ref(false);
  const triggerModalRef = ref<InstanceType<typeof TriggerModal>>();
  const deleteWorkspace = ref(false);

  const editForm = reactive({
    name: '',
    repository_url: '',
    git_credential_id: '',
    default_branch: 'master',
  });
  const editErrors = reactive({
    name: '',
    repository_url: '',
  });
  const variableForm = reactive({
    name: '',
    value: '',
    description: '',
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
        credential.type === 'gitee_token'
    )
  );
  const gitCredentialOptions = computed(() =>
    gitCredentials.value.map((credential) => ({
      value: credential.id,
      label: credential.name,
      description: credential.type,
    }))
  );
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
      repository_url: repository.value.repository_url,
      git_credential_id: repository.value.git_credential_id ?? '',
      default_branch: repository.value.default_branch || 'master',
    });
    Object.assign(editErrors, { name: '', repository_url: '' });
  }

  function validateEditForm() {
    editErrors.name = editForm.name.trim() ? '' : '请输入名称';
    editErrors.repository_url = editForm.repository_url.trim() ? '' : '请输入仓库地址';
    return !editErrors.name && !editErrors.repository_url;
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

  async function fetchTemplates() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      const res = await pipelineTemplateApi.list({ per_page: 100, project_id: projectId });
      templates.value = res.items;
    } catch {
      toast.error('获取模板列表失败');
    }
  }

  async function fetchWebhooks() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      const resp = await webhookApi.list(repositoryId, { project_id: projectId });
      webhooks.value = resp.items;
    } catch {
      toast.error('获取 Webhook 列表失败');
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

  async function openTriggerModal() {
    if (!repository.value?.git_credential_id) {
      toast.error('请先配置 Git 凭据后再触发流水线');
      return;
    }
    await fetchTemplates();
    triggerModalRef.value?.open();
  }

  async function handleTrigger(data: {
    template_id: string;
    trigger_ref: string;
    variables: Record<string, string>;
  }) {
    try {
      await executeOp(async () => {
        const run = await repositoryApi.trigger(repositoryId, data);
        toast.success('触发成功');
        router.push(`/pipeline-run/${run.id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '触发失败');
    }
  }

  async function openEditDialog() {
    resetEditForm();
    await fetchCredentials();
    isEditDialogOpen.value = true;
  }

  async function handleEditOk() {
    if (!validateEditForm()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await repositoryApi.update(repositoryId, {
          name: editForm.name,
          repository_url: editForm.repository_url,
          git_credential_id: editForm.git_credential_id || undefined,
          default_branch: editForm.default_branch || 'master',
        });
        repository.value = updated;
        resetEditForm();
        toast.success('更新成功');
        isEditDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '更新失败');
    }
  }

  function openDeleteDialog() {
    deleteWorkspace.value = false;
    isDeleteDialogOpen.value = true;
  }

  async function handleDeleteOk() {
    try {
      await executeOp(async () => {
        await repositoryApi.delete(repositoryId, {
          delete_workspace: deleteWorkspace.value,
        });
        toast.success('删除成功');
        router.push('/repository');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    }
  }

  function openAddVariableDialog() {
    Object.assign(variableForm, { name: '', value: '', description: '' });
    Object.assign(variableErrors, { name: '' });
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
    });
    Object.assign(variableErrors, { name: '' });
    isEditVariableDialogOpen.value = true;
  }

  async function handleAddVariableOk() {
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
      toast.error('请先选择项目');
      return;
    }
    try {
      await executeOp(async () => {
        const nextVariable: VariableDeclarationResp = {
          name: variableForm.name.trim(),
          description: variableForm.description.trim(),
          default: undefined,
          value: variableForm.value,
          secret: false,
          source: 'repository_custom',
          editable: true,
        };
        const updated = await repositoryApi.update(repositoryId, {
          variable_overrides: { items: variableOverridesWith(nextVariable) },
        });
        repository.value = updated;
        toast.success('添加成功');
        isAddVariableDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '添加失败');
    }
  }

  async function handleEditVariableOk() {
    if (!editingVariableName.value) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
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
            }),
          },
        });
        repository.value = updated;
        toast.success('更新成功');
        isEditVariableDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '更新失败');
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
        toast.success('删除成功');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    }
  }

  onMounted(async () => {
    await fetchRepository();
    await Promise.all([fetchTemplates(), fetchWebhooks()]);
  });
</script>
