<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('environment.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('environment.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateModal">
          <Plus class="size-4" />
          {{ t('environment.create') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="environments.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[960px]">
          <thead>
            <tr>
              <th>{{ t('environment.fields.name') }}</th>
              <th>{{ t('environment.fields.code') }}</th>
              <th>{{ t('environment.fields.bindings') }}</th>
              <th>{{ t('common.description') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="env in environments" :key="env.id">
              <td class="font-medium text-foreground">{{ env.name }}</td>
              <td class="font-mono text-sm text-muted-foreground">{{ env.code }}</td>
              <td class="text-foreground">{{ env.bindings?.length ?? 0 }}</td>
              <td class="max-w-xs truncate text-muted-foreground" :title="env.description || ''">
                {{ env.description || '—' }}
              </td>
              <td class="whitespace-nowrap text-muted-foreground">
                {{ formatTime(env.created_at) }}
              </td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditModal(env)">
                    {{ t('common.edit') }}
                  </button>
                  <button class="app-link" @click="openBindingsModal(env)">
                    {{ t('environment.actions.bindings') }}
                  </button>
                  <button
                    class="app-link-danger"
                    :disabled="operating || env.code === 'local'"
                    @click="openDeleteModal(env)"
                  >
                    {{ t('common.delete') }}
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

    <AppDialog
      v-model:open="isFormDialogOpen"
      :title="editingId ? t('environment.dialog.edit') : t('environment.dialog.create')"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('environment.fields.code') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.code"
            type="text"
            class="app-input"
            :class="errors.code ? 'app-input-error' : ''"
            :disabled="!!editingId"
            :placeholder="t('environment.placeholders.code')"
          />
          <p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
          <p v-else class="app-field-hint">{{ t('environment.hints.code') }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('environment.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            :placeholder="t('environment.placeholders.name')"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('common.description') }}</label>
          <input
            v-model="form.description"
            type="text"
            class="app-input"
            :placeholder="t('environment.placeholders.description')"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isFormDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleSave">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isBindingsDialogOpen"
      :title="t('environment.dialog.bindings', { name: bindingsEnv?.name || '' })"
      width-class="w-[min(840px,calc(100vw-32px))]"
    >
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-sm text-muted-foreground">{{ t('environment.hints.bindings') }}</p>
          <button type="button" class="app-link text-sm" @click="addBindingRow">
            {{ t('environment.actions.addBinding') }}
          </button>
        </div>
        <div
          v-for="(row, index) in bindingRows"
          :key="index"
          class="grid grid-cols-1 gap-2 rounded-md border border-border p-3 md:grid-cols-2"
        >
          <input
            v-model="row.component_name"
            type="text"
            class="app-input"
            :placeholder="t('environment.placeholders.componentName')"
          />
          <select v-model="row.protocol" class="app-input">
            <option value="http">http</option>
            <option value="tcp">tcp</option>
          </select>
          <input
            v-model.number="row.container_port"
            type="number"
            min="1"
            max="65535"
            class="app-input"
            :placeholder="t('environment.placeholders.port')"
          />
          <input
            v-model="row.entrypoint"
            type="text"
            class="app-input"
            :placeholder="t('environment.placeholders.entrypoint')"
          />
          <input
            v-model="row.domains_text"
            type="text"
            class="app-input md:col-span-2"
            :placeholder="t('environment.placeholders.domains')"
          />
          <select v-model="row.tls_mode" class="app-input">
            <option value="none">none</option>
            <option value="letsencrypt">letsencrypt</option>
          </select>
          <input
            v-model="row.sni_host"
            type="text"
            class="app-input"
            :placeholder="t('environment.placeholders.sniHost')"
          />
          <div class="md:col-span-2">
            <button type="button" class="app-link-danger" @click="removeBindingRow(index)">
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isBindingsDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleBindingsSave">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('environment.dialog.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{ t('environment.dialog.deleteConfirm', { name: pendingDelete?.name || '' }) }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="operating" @click="handleDelete">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { ToolbarRoot } from 'reka-ui';
  import { environmentApi } from '@/api/cd/environment';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { EnvironmentBindingReq, EnvironmentResp } from '@/gen/proto/orbit/v1/environment';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  type BindingFormRow = {
    component_name: string;
    protocol: string;
    container_port: number;
    domains_text: string;
    entrypoint: string;
    tls_mode: string;
    sni_host: string;
  };

  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const environments = ref<EnvironmentResp[]>([]);
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);

  const isFormDialogOpen = ref(false);
  const isBindingsDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const editingId = ref('');
  const bindingsEnv = ref<EnvironmentResp | null>(null);
  const pendingDelete = ref<EnvironmentResp | null>(null);
  const bindingRows = ref<BindingFormRow[]>([]);

  const form = reactive({
    code: '',
    name: '',
    description: '',
  });
  const errors = reactive({ code: '', name: '' });

  const codePattern = /^[a-z][a-z0-9-]*$/;

  watch(
    () => projectStore.activeProjectId,
    () => {
      pagination.current = 1;
      fetchData();
    }
  );

  function emptyBindingRow(): BindingFormRow {
    return {
      component_name: '',
      protocol: 'http',
      container_port: 80,
      domains_text: '',
      entrypoint: 'websecure',
      tls_mode: 'none',
      sni_host: '',
    };
  }

  function validateForm() {
    if (editingId.value) {
      errors.code = '';
    } else {
      errors.code = codePattern.test(form.code.trim())
        ? ''
        : t('environment.validation.codeInvalid');
    }
    errors.name = form.name.trim() ? '' : t('environment.validation.nameRequired');
    return !errors.code && !errors.name;
  }

  async function fetchData() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      environments.value = [];
      pagination.total = 0;
      toast.error(t('environment.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await environmentApi.list({
          project_id: projectId,
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        environments.value = res.items ?? [];
        pagination.total = res.total ?? 0;
      });
    } catch {
      toast.error(t('environment.toast.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchData();
  }

  function goPage(page: number) {
    pagination.current = page;
    fetchData();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchData();
  }

  function openCreateModal() {
    editingId.value = '';
    Object.assign(form, { code: '', name: '', description: '' });
    Object.assign(errors, { code: '', name: '' });
    isFormDialogOpen.value = true;
  }

  function openEditModal(env: EnvironmentResp) {
    editingId.value = env.id;
    Object.assign(form, {
      code: env.code,
      name: env.name,
      description: env.description || '',
    });
    Object.assign(errors, { code: '', name: '' });
    isFormDialogOpen.value = true;
  }

  async function handleSave() {
    if (!validateForm()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('environment.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        if (editingId.value) {
          await environmentApi.update(editingId.value, {
            name: form.name.trim(),
            description: form.description.trim() || undefined,
          });
        } else {
          await environmentApi.create(
            {
              project_id: projectId,
              code: form.code.trim(),
              name: form.name.trim(),
              description: form.description.trim() || undefined,
            },
            { project_id: projectId }
          );
        }
        toast.success(t('environment.toast.saveSuccess'));
        isFormDialogOpen.value = false;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('environment.toast.saveFailed'));
    }
  }

  function openBindingsModal(env: EnvironmentResp) {
    bindingsEnv.value = env;
    bindingRows.value =
      env.bindings?.length > 0
        ? env.bindings.map((item) => ({
            component_name: item.component_name,
            protocol: item.protocol || 'http',
            container_port: item.container_port,
            domains_text: (item.domains ?? []).join(', '),
            entrypoint: item.entrypoint || (item.protocol === 'tcp' ? '' : 'websecure'),
            tls_mode: item.tls_mode || 'none',
            sni_host: item.sni_host || '',
          }))
        : [];
    isBindingsDialogOpen.value = true;
  }

  function addBindingRow() {
    bindingRows.value.push(emptyBindingRow());
  }

  function removeBindingRow(index: number) {
    bindingRows.value.splice(index, 1);
  }

  function toBindingReq(row: BindingFormRow): EnvironmentBindingReq {
    const domains = row.domains_text
      .split(/[,\s]+/)
      .map((item) => item.trim())
      .filter(Boolean);
    return {
      component_name: row.component_name.trim(),
      protocol: row.protocol,
      container_port: Number(row.container_port) || 0,
      domains,
      entrypoint: row.entrypoint.trim() || (row.protocol === 'http' ? 'websecure' : ''),
      tls_mode: row.tls_mode || 'none',
      sni_host: row.sni_host.trim() || undefined,
    };
  }

  async function handleBindingsSave() {
    const env = bindingsEnv.value;
    if (!env) {
      return;
    }
    const bindings = bindingRows.value
      .map(toBindingReq)
      .filter((item) => item.component_name && item.container_port > 0);
    try {
      await executeOp(async () => {
        await environmentApi.replaceBindings(env.id, { bindings });
        toast.success(t('environment.toast.saveSuccess'));
        isBindingsDialogOpen.value = false;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('environment.toast.saveFailed'));
    }
  }

  function openDeleteModal(env: EnvironmentResp) {
    pendingDelete.value = env;
    isDeleteDialogOpen.value = true;
  }

  async function handleDelete() {
    const target = pendingDelete.value;
    if (!target) {
      return;
    }
    try {
      await executeOp(async () => {
        await environmentApi.delete(target.id);
        toast.success(t('environment.toast.deleteSuccess'));
        isDeleteDialogOpen.value = false;
        pendingDelete.value = null;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('environment.toast.deleteFailed'));
    }
  }

  onMounted(fetchData);
</script>
