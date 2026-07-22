<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('gateway.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('gateway.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateModal">
          <Plus class="size-4" />
          {{ t('gateway.create') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="gateways.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[960px]">
          <thead>
            <tr>
              <th>{{ t('gateway.fields.name') }}</th>
              <th>{{ t('gateway.fields.code') }}</th>
              <th>{{ t('gateway.fields.restApiUrl') }}</th>
              <th>{{ t('gateway.fields.baseDomain') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in gateways"
              :key="item.id"
              class="cursor-pointer"
              @click="router.push(`/cd/gateways/${item.id}`)"
            >
              <td class="text-foreground">{{ item.name }}</td>
              <td class="font-mono text-foreground">{{ item.code }}</td>
              <td class="font-mono text-sm text-foreground">{{ item.rest_api_url }}</td>
              <td class="text-foreground">{{ item.base_domain }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(item.created_at) }}
              </td>
              <td class="whitespace-nowrap" @click.stop>
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditModal(item)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    class="app-link-danger"
                    :disabled="operating"
                    @click="openDeleteModal(item)"
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
      :title="editingId ? t('gateway.dialog.edit') : t('gateway.dialog.create')"
      width-class="w-[min(560px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('gateway.fields.code') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.code"
            type="text"
            class="app-input"
            :class="errors.code ? 'app-input-error' : ''"
            :disabled="!!editingId"
            :placeholder="t('gateway.placeholders.code')"
          />
          <p v-if="errors.code" class="app-field-error">{{ errors.code }}</p>
          <p v-else class="app-field-hint">{{ t('gateway.hints.code') }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('gateway.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.name')"
          />
          <p v-if="errors.name" class="app-field-error">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('gateway.fields.restApiUrl') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.rest_api_url"
            type="text"
            class="app-input"
            :class="errors.rest_api_url ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.restApiUrl')"
          />
          <p v-if="errors.rest_api_url" class="app-field-error">{{ errors.rest_api_url }}</p>
          <p v-else class="app-field-hint">{{ t('gateway.hints.restApiUrl') }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('gateway.fields.baseDomain') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.base_domain"
            type="text"
            class="app-input"
            :class="errors.base_domain ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.baseDomain')"
          />
          <p v-if="errors.base_domain" class="app-field-error">{{ errors.base_domain }}</p>
          <p v-else class="app-field-hint">{{ t('gateway.hints.baseDomain') }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('gateway.fields.imagePullPolicy') }}</label>
          <SelectControl
            v-model="form.image_pull_policy"
            :options="imagePullPolicyOptions"
            :placeholder="t('application.imagePullPolicyPlaceholder')"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('gateway.fields.image') }}</label>
          <input
            v-model="form.image"
            type="text"
            class="app-input"
            :placeholder="t('gateway.placeholders.image')"
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
      v-model:open="isDeleteDialogOpen"
      :title="t('gateway.dialog.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{ t('gateway.dialog.deleteConfirm', { name: pendingDelete?.name || '' }) }}
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
  import { useRouter } from 'vue-router';
  import { ToolbarRoot } from 'reka-ui';
  import { gatewayApi } from '@/api/cd/gateway';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const router = useRouter();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const gateways = ref<GatewayResp[]>([]);
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);

  const isFormDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const editingId = ref('');
  const pendingDelete = ref<GatewayResp | null>(null);

  const form = reactive({
    code: '',
    name: '',
    rest_api_url: 'http://traefik:8080',
    base_domain: 'local.test',
    image: '',
    image_pull_policy: 'missing',
  });
  const errors = reactive({ code: '', name: '', rest_api_url: '', base_domain: '' });

  const imagePullPolicyOptions = computed(() => [
    { value: 'missing', label: t('application.imagePullPolicyOptions.missing') },
    { value: 'always', label: t('application.imagePullPolicyOptions.always') },
    { value: 'never', label: t('application.imagePullPolicyOptions.never') },
  ]);

  const codePattern = /^[a-z][a-z0-9-]*$/;

  watch(
    () => projectStore.activeProjectId,
    () => {
      pagination.current = 1;
      fetchData();
    }
  );

  function validateForm() {
    if (editingId.value) {
      errors.code = '';
    } else {
      errors.code = codePattern.test(form.code.trim()) ? '' : t('gateway.validation.codeInvalid');
    }
    errors.name = form.name.trim() ? '' : t('gateway.validation.nameRequired');
    errors.rest_api_url = form.rest_api_url.trim()
      ? ''
      : t('gateway.validation.restApiUrlRequired');
    errors.base_domain = form.base_domain.trim() ? '' : t('gateway.validation.baseDomainRequired');
    return !errors.code && !errors.name && !errors.rest_api_url && !errors.base_domain;
  }

  async function fetchData() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      gateways.value = [];
      pagination.total = 0;
      toast.error(t('gateway.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await gatewayApi.list({
          project_id: projectId,
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        gateways.value = res.items ?? [];
        pagination.total = res.total ?? 0;
      });
    } catch {
      toast.error(t('gateway.toast.loadFailed'));
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

  function resetForm() {
    Object.assign(form, {
      code: '',
      name: '',
      rest_api_url: 'http://traefik:8080',
      base_domain: 'local.test',
      image: '',
      image_pull_policy: 'missing',
    });
    Object.assign(errors, { code: '', name: '', rest_api_url: '', base_domain: '' });
  }

  function openCreateModal() {
    editingId.value = '';
    resetForm();
    isFormDialogOpen.value = true;
  }

  function openEditModal(item: GatewayResp) {
    editingId.value = item.id;
    Object.assign(form, {
      code: item.code,
      name: item.name,
      rest_api_url: item.rest_api_url || '',
      base_domain: item.base_domain || '',
      image: item.image || '',
      image_pull_policy: item.image_pull_policy || 'missing',
    });
    Object.assign(errors, { code: '', name: '', rest_api_url: '', base_domain: '' });
    isFormDialogOpen.value = true;
  }

  async function handleSave() {
    if (!validateForm()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('gateway.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        if (editingId.value) {
          await gatewayApi.update(editingId.value, {
            name: form.name.trim(),
            rest_api_url: form.rest_api_url.trim(),
            base_domain: form.base_domain.trim(),
            image: form.image.trim() || undefined,
            image_pull_policy: form.image_pull_policy,
          });
        } else {
          await gatewayApi.create(
            {
              project_id: projectId,
              code: form.code.trim(),
              name: form.name.trim(),
              rest_api_url: form.rest_api_url.trim(),
              base_domain: form.base_domain.trim(),
              image: form.image.trim() || undefined,
              image_pull_policy: form.image_pull_policy,
            },
            { project_id: projectId }
          );
        }
        toast.success(t('gateway.toast.saveSuccess'));
        isFormDialogOpen.value = false;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('gateway.toast.saveFailed'));
    }
  }

  function openDeleteModal(item: GatewayResp) {
    pendingDelete.value = item;
    isDeleteDialogOpen.value = true;
  }

  async function handleDelete() {
    const target = pendingDelete.value;
    if (!target) {
      return;
    }
    try {
      await executeOp(async () => {
        await gatewayApi.delete(target.id);
        toast.success(t('gateway.toast.deleteSuccess'));
        isDeleteDialogOpen.value = false;
        pendingDelete.value = null;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('gateway.toast.deleteFailed'));
    }
  }

  onMounted(fetchData);
</script>
