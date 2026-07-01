<template>
  <div class="app-surface">
    <div class="app-section-header flex items-center justify-between">
      <h3 class="font-semibold text-foreground">Webhook 配置</h3>
      <button class="app-button-primary h-9 px-3" @click="openCreateModal">
        <Plus class="size-4" />
        添加 Webhook
      </button>
    </div>

    <div class="overflow-x-auto">
      <table class="app-table-detail min-w-[960px]">
        <thead>
          <tr>
            <th>名称</th>
            <th>模板</th>
            <th>分支过滤</th>
            <th>Webhook URL</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="webhooks.length === 0">
            <td colspan="6" class="text-center text-muted-foreground">暂无 Webhook 配置</td>
          </tr>
          <tr v-for="wh in webhooks" :key="wh.id">
            <td class="text-foreground">{{ wh.name }}</td>
            <td>
              <router-link :to="`/ci/template/${wh.template_id}`" class="app-link">
                {{ getTemplateName(wh.template_id) }}
              </router-link>
            </td>
            <td>
              <span v-if="wh.branch_filter" class="text-foreground">{{ wh.branch_filter }}</span>
              <span v-else class="text-destructive">拒绝所有分支</span>
            </td>
            <td>
              <div class="flex min-w-0 items-center gap-2">
                <span class="max-w-72 truncate">
                  <AppBadge>
                    {{ webhookUrl(wh.id) }}
                  </AppBadge>
                </span>
                <button class="app-icon-button size-7" title="复制 URL" @click="copyUrl(wh.id)">
                  <Copy class="size-3.5" />
                </button>
              </div>
            </td>
            <td>
              <AppBadge variant="status" :tone="wh.enabled ? 'success' : 'default'">
                {{ wh.enabled ? '启用' : '停用' }}
              </AppBadge>
            </td>
            <td>
              <div class="flex items-center gap-3">
                <button class="app-link" @click="openEditModal(wh)">编辑</button>
                <button class="app-link-danger" @click="handleDelete(wh)">删除</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <AppDialog
      v-model:open="isDialogOpen"
      :title="editingWebhook ? '编辑 Webhook' : '添加 Webhook'"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">名称</label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            placeholder="例如: main-branch-webhook"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">流水线模板</label>
          <ComboboxSelect
            v-model="form.template_id"
            :options="templateOptions"
            placeholder="请选择模板"
          />
          <p v-if="errors.template_id" class="app-field-error text-xs">
            {{ errors.template_id }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            分支过滤
            <span class="font-normal text-muted-foreground">（可选，支持正则）</span>
          </label>
          <input
            v-model="form.branch_filter"
            type="text"
            class="app-input"
            placeholder="例如: ^main$"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">签名密钥</label>
          <div class="relative">
            <input
              v-model="form.secret"
              :type="isSecretVisible ? 'text' : 'password'"
              class="app-input pr-10"
              :class="errors.secret ? 'app-input-error' : ''"
              placeholder="用于验证 Webhook 请求"
            />
            <button
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground transition-colors hover:text-foreground"
              :aria-label="isSecretVisible ? '隐藏签名密钥' : '显示签名密钥'"
              @click="isSecretVisible = !isSecretVisible"
            >
              <EyeOff v-if="isSecretVisible" class="size-4" />
              <Eye v-else class="size-4" />
            </button>
          </div>
          <p v-if="errors.secret" class="app-field-error text-xs">{{ errors.secret }}</p>
        </div>
        <label v-if="editingWebhook" class="flex cursor-pointer items-center gap-2">
          <input v-model="form.enabled" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">启用</span>
        </label>
      </div>
      <template #footer>
        <button class="app-button" @click="isDialogOpen = false">取消</button>
        <button class="app-button-primary" :disabled="operating" @click="handleOk">
          {{ editingWebhook ? '保存' : '添加' }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      title="删除 Webhook"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">确定删除此 Webhook？此操作不可撤销。</p>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">取消</button>
        <button class="app-button-destructive" :disabled="operating" @click="confirmDelete">
          删除
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Copy, Eye, EyeOff, Plus } from 'lucide-vue-next';
  import { computed, nextTick, reactive, ref } from 'vue';
  import { webhookApi } from '@/api/ci';
  import AppBadge from '@/components/AppBadge.vue';
  import { buildApiUrl } from '@/config';
  import AppDialog from '@/components/AppDialog.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { PipelineTemplate } from '@/types/ci/template';
  import type { RepositoryWebhook } from '@/types/ci/webhook';

  const props = defineProps<{
    repositoryId: string;
    webhooks: RepositoryWebhook[];
    templates: PipelineTemplate[];
  }>();

  const emit = defineEmits<{
    refresh: [];
  }>();

  const toast = useToast();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const isDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isSecretVisible = ref(true);
  const editingWebhook = ref<RepositoryWebhook>();
  const deletingWebhook = ref<RepositoryWebhook>();

  const form = reactive({
    name: '',
    template_id: '',
    branch_filter: '',
    secret: '',
    enabled: true,
  });
  const errors = reactive({
    name: '',
    template_id: '',
    secret: '',
  });
  const templateOptions = computed(() =>
    props.templates.map((template) => ({
      value: template.id,
      label: template.name,
    }))
  );

  function getTemplateName(templateId: string) {
    const tpl = props.templates.find((t) => t.id === templateId);
    return tpl?.name || templateId;
  }

  function webhookUrl(webhookId: string) {
    return buildApiUrl(`/api/ci/webhook/${webhookId}`);
  }

  async function copyUrl(webhookId: string) {
    try {
      await navigator.clipboard.writeText(webhookUrl(webhookId));
      toast.success('URL 已复制');
    } catch {
      toast.error('复制失败，请手动复制');
    }
  }

  function resetErrors() {
    Object.assign(errors, { name: '', template_id: '', secret: '' });
  }

  function openCreateModal() {
    editingWebhook.value = undefined;
    Object.assign(form, {
      name: '',
      template_id: '',
      branch_filter: '',
      secret: '',
      enabled: true,
    });
    resetErrors();
    isSecretVisible.value = false;
    isDialogOpen.value = true;
  }

  function openEditModal(wh: RepositoryWebhook) {
    editingWebhook.value = wh;
    Object.assign(form, {
      name: wh.name,
      template_id: wh.template_id,
      branch_filter: wh.branch_filter || '',
      secret: wh.secret,
      enabled: wh.enabled,
    });
    resetErrors();
    isSecretVisible.value = false;
    isDialogOpen.value = true;
  }

  function validateForm() {
    errors.name = form.name.trim() ? '' : '请输入名称';
    errors.template_id = form.template_id ? '' : '请选择模板';
    errors.secret = form.secret ? '' : '请输入签名密钥';
    return !errors.name && !errors.template_id && !errors.secret;
  }

  async function handleOk() {
    if (!validateForm()) {
      return;
    }

    try {
      await executeOp(async () => {
        if (editingWebhook.value) {
          await webhookApi.update(props.repositoryId, editingWebhook.value.id, {
            name: form.name,
            template_id: form.template_id,
            branch_filter: form.branch_filter || null,
            secret: form.secret,
            enabled: form.enabled,
          });
          toast.success('更新成功');
        } else {
          await webhookApi.create(props.repositoryId, {
            name: form.name,
            template_id: form.template_id,
            secret: form.secret,
            branch_filter: form.branch_filter || null,
          });
          toast.success('创建成功');
        }
        isDialogOpen.value = false;
        emit('refresh');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败');
    }
  }

  async function handleDelete(wh: RepositoryWebhook) {
    deletingWebhook.value = wh;
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
    isDeleteDialogOpen.value = true;
  }

  async function confirmDelete() {
    if (!deletingWebhook.value) {
      return;
    }

    const webhookId = deletingWebhook.value.id;
    try {
      await executeOp(async () => {
        await webhookApi.delete(props.repositoryId, webhookId);
        toast.success('删除成功');
        isDeleteDialogOpen.value = false;
        emit('refresh');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    }
  }
</script>
