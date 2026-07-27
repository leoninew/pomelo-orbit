<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="min-w-0 break-words text-xl font-semibold text-foreground">
          {{ credential?.name ?? '凭据详情' }}
        </h1>
        <DetailHeaderMeta v-if="credential">
          <AppBadge>{{ credential.type }}</AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="credential"
          class="app-button-primary h-9 px-3"
          :disabled="operating"
          @click="openEditModal"
        >
          <Pencil class="size-4" />
          编辑
        </button>
        <button
          v-if="credential"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="handleExport"
        >
          <Download class="size-4" />
          导出
        </button>
        <button
          v-if="credential"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          删除
        </button>
        <button class="app-button h-9 px-4" @click="$router.push('/credential')">
          <ArrowLeft class="size-4" />
          返回
        </button>
      </div>
    </div>

    <div class="app-surface">
      <div class="app-section-header">
        <h2 class="font-semibold text-foreground">基本信息</h2>
      </div>

      <AppSpinner v-if="loading" class="px-5 py-10" />
      <dl
        v-else-if="credential"
        class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2"
      >
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">凭据名称</dt>
          <dd class="text-foreground">{{ credential.name }}</dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">类型</dt>
          <dd>
            <AppBadge variant="pill">
              {{ credential.type }}
            </AppBadge>
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">创建时间</dt>
          <dd class="text-muted-foreground">{{ formatTime(credential.created_at) }}</dd>
        </div>
        <div v-if="credential.type === 'runtime_env'" class="flex gap-2 sm:col-span-2">
          <dt class="w-32 shrink-0 text-muted-foreground">运行时环境变量</dt>
          <dd class="min-w-0 flex-1">
            <div class="mb-2 flex justify-end">
              <button
                type="button"
                class="inline-flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
                :aria-label="isCredentialDataVisible ? '隐藏凭据内容' : '显示凭据内容'"
                @click="isCredentialDataVisible = !isCredentialDataVisible"
              >
                <EyeOff v-if="isCredentialDataVisible" class="size-4" />
                <Eye v-else class="size-4" />
                {{ isCredentialDataVisible ? '隐藏值' : '显示值' }}
              </button>
            </div>
            <div class="overflow-x-auto rounded border border-border">
              <table class="app-table-detail min-w-[420px]">
                <thead>
                  <tr>
                    <th>键名</th>
                    <th>值</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="runtimeEnvRows.length === 0">
                    <td colspan="2" class="text-center text-muted-foreground">—</td>
                  </tr>
                  <tr v-for="row in runtimeEnvRows" :key="row.key">
                    <td class="break-all font-mono text-xs text-foreground">{{ row.key }}</td>
                    <td
                      class="min-w-[220px] whitespace-pre-wrap break-all font-mono text-xs text-foreground"
                    >
                      {{ isCredentialDataVisible ? row.value : '********' }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </dd>
        </div>
        <div v-else class="flex gap-2 sm:col-span-2">
          <dt class="w-32 shrink-0 text-muted-foreground">凭据内容</dt>
          <dd class="flex min-w-0 flex-1 items-start gap-2">
            <span class="min-w-0 whitespace-pre-wrap break-all text-foreground">
              {{ isCredentialDataVisible ? credential.data : '********' }}
            </span>
            <button
              type="button"
              class="shrink-0 p-0.5 text-muted-foreground transition-colors hover:text-foreground"
              :aria-label="isCredentialDataVisible ? '隐藏凭据内容' : '显示凭据内容'"
              @click="isCredentialDataVisible = !isCredentialDataVisible"
            >
              <EyeOff v-if="isCredentialDataVisible" class="size-4" />
              <Eye v-else class="size-4" />
            </button>
          </dd>
        </div>
      </dl>
    </div>

    <AppDialog
      v-model:open="isEditModalOpen"
      title="编辑凭据"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">凭据名称</label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">凭据内容</label>
          <RuntimeEnvEditor
            v-if="credential?.type === 'runtime_env'"
            v-model="form.runtimeEnvRows"
            :error="errors.data"
          />
          <template v-else>
            <textarea
              v-model="form.data"
              class="app-textarea text-xs"
              :class="errors.data ? 'app-input-error' : ''"
              rows="8"
              :placeholder="credential ? getDataPlaceholder(credential.type) : ''"
            />
            <p v-if="errors.data" class="app-field-error text-xs">{{ errors.data }}</p>
          </template>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isEditModalOpen = false">取消</button>
        <button class="app-button-primary" :disabled="operating" @click="handleEditOk">保存</button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteModalOpen"
      title="删除凭据"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">确定删除此凭据？</p>
      <template #footer>
        <button class="app-button" @click="isDeleteModalOpen = false">取消</button>
        <button class="app-button-destructive" :disabled="operating" @click="handleDelete">
          删除
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Download, Eye, EyeOff, Pencil, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { credentialApi } from '@/api/credential/credential';
  import AppDialog from '@/components/AppDialog.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import RuntimeEnvEditor from '@/components/RuntimeEnvEditor.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { CredentialDetailResp } from '@/gen/proto/orbit/v1/credential/credential';
  import { formatTime } from '@/utils/time';
  import {
    parseRuntimeEnvRows,
    serializeRuntimeEnvRows,
    validateRuntimeEnvRows,
    type RuntimeEnvRow,
  } from '@/utils/runtimeEnv';

  const props = defineProps<{ id: string }>();
  const $router = useRouter();
  const toast = useToast();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const credential = ref<CredentialDetailResp>();
  const isCredentialDataVisible = ref(false);
  const isEditModalOpen = ref(false);
  const isDeleteModalOpen = ref(false);
  const form = reactive({ name: '', data: '', runtimeEnvRows: [] as RuntimeEnvRow[] });
  const errors = reactive({ name: '', data: '' });
  const runtimeEnvRows = computed(() => parseRuntimeEnvRows(credential.value?.data));

  async function fetchCredential() {
    try {
      await execute(async () => {
        credential.value = await credentialApi.get(props.id);
        isCredentialDataVisible.value = false;
      });
    } catch {
      toast.error('获取凭据详情失败');
    }
  }

  function openEditModal() {
    Object.assign(form, {
      name: credential.value?.name ?? '',
      data: credential.value?.data ?? '',
      runtimeEnvRows:
        credential.value?.type === 'runtime_env' ? parseRuntimeEnvRows(credential.value.data) : [],
    });
    Object.assign(errors, { name: '', data: '' });
    isEditModalOpen.value = true;
  }

  async function handleEditOk() {
    errors.name = form.name.trim() ? '' : '请输入凭据名称';
    errors.data =
      credential.value?.type === 'runtime_env'
        ? validateRuntimeEnvRows(form.runtimeEnvRows) || ''
        : form.data.trim()
          ? ''
          : '请输入凭据内容';
    if (errors.name || errors.data) {
      return;
    }
    try {
      await executeOp(async () => {
        await credentialApi.update(props.id, {
          name: form.name,
          data:
            credential.value?.type === 'runtime_env'
              ? serializeRuntimeEnvRows(form.runtimeEnvRows) || ''
              : form.data,
        });
        toast.success('更新成功');
        isEditModalOpen.value = false;
        fetchCredential();
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败');
    }
  }

  function openDeleteModal() {
    isDeleteModalOpen.value = true;
  }

  async function handleDelete() {
    try {
      await executeOp(async () => {
        await credentialApi.delete(props.id);
        toast.success('删除成功');
        $router.push('/credential');
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '删除失败');
    }
  }

  async function handleExport() {
    try {
      const data = await credentialApi.exportCredential(props.id);
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${credential.value?.name ?? 'credential'}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '导出失败');
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

  onMounted(fetchCredential);
</script>
