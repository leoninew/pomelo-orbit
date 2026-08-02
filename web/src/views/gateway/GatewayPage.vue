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
        <button class="app-button-primary px-5" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('gateway.create') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="gateways.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[960px]">
          <thead>
            <tr>
              <th>{{ t('gateway.fields.name') }}</th>
              <th>{{ t('gateway.fields.code') }}</th>
              <th>{{ t('gateway.fields.restApiUrl') }}</th>
              <th>{{ t('gateway.fields.baseDomain') }}</th>
              <th>{{ t('gateway.fields.defaultEntrypoint') }}</th>
              <th>{{ t('gateway.fields.tlsMode') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in gateways" :key="item.id">
              <td>
                <router-link :to="`/gateway/${item.id}`" class="app-link">
                  {{ item.name }}
                </router-link>
              </td>
              <td class="text-foreground">{{ item.code }}</td>
              <td class="text-foreground">{{ item.rest_api_url }}</td>
              <td class="text-foreground">{{ item.base_domain }}</td>
              <td>
                <AppBadge variant="pill">{{ item.default_entrypoint }}</AppBadge>
              </td>
              <td>
                <AppBadge variant="pill">{{ item.tls_mode }}</AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(item.created_at) }}
              </td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <router-link :to="`/gateway/edit/${item.id}`" class="app-link">
                    {{ t('common.edit') }}
                  </router-link>
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
      v-model:open="isCreateDialogOpen"
      :title="t('gateway.dialog.create')"
      width-class="w-[min(720px,calc(100vw-32px))]"
      body-class="space-y-4 px-6 py-4 text-sm"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="createForm.name"
            type="text"
            class="app-input"
            :class="createErrors.name ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.name')"
            :aria-invalid="createErrors.name ? 'true' : undefined"
            @input="createErrors.name = ''"
          />
          <p v-if="createErrors.name" class="app-field-error mt-1 text-xs">
            {{ createErrors.name }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.code') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="createForm.code"
            type="text"
            class="app-input"
            :class="createErrors.code ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.code')"
            :aria-invalid="createErrors.code ? 'true' : undefined"
            @input="createErrors.code = ''"
          />
          <p v-if="createErrors.code" class="app-field-error mt-1 text-xs">
            {{ createErrors.code }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.restApiUrl') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="createForm.rest_api_url"
            type="text"
            class="app-input"
            :class="createErrors.rest_api_url ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.restApiUrl')"
            :aria-invalid="createErrors.rest_api_url ? 'true' : undefined"
            @input="createErrors.rest_api_url = ''"
          />
          <p v-if="createErrors.rest_api_url" class="app-field-error mt-1 text-xs">
            {{ createErrors.rest_api_url }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.baseDomain') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="createForm.base_domain"
            type="text"
            class="app-input"
            :class="createErrors.base_domain ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.baseDomain')"
            :aria-invalid="createErrors.base_domain ? 'true' : undefined"
            @input="createErrors.base_domain = ''"
          />
          <p v-if="createErrors.base_domain" class="app-field-error mt-1 text-xs">
            {{ createErrors.base_domain }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.defaultEntrypoint') }}
          </label>
          <RawValueSelect
            v-model="createForm.default_entrypoint"
            :values="entrypointValues"
            :placeholder="t('gateway.placeholders.defaultEntrypoint')"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">{{ t('gateway.fields.tlsMode') }}</label>
          <RawValueSelect
            v-model="createForm.tls_mode"
            :values="tlsModeValues"
            :placeholder="t('gateway.placeholders.tlsMode')"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.imagePullPolicy') }}
          </label>
          <RawValueSelect
            v-model="createForm.initial_component_pull_policy"
            :values="imagePullPolicyValues"
            :placeholder="t('application.imagePullPolicyPlaceholder')"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.image') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="createForm.initial_component_image"
            type="text"
            class="app-input"
            :class="createErrors.image ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.image')"
            :aria-invalid="createErrors.image ? 'true' : undefined"
            @input="createErrors.image = ''"
          />
          <p v-if="createErrors.image" class="app-field-error mt-1 text-xs">
            {{ createErrors.image }}
          </p>
        </div>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.create')"
          @cancel="isCreateDialogOpen = false"
          @confirm="handleCreate"
        />
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
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { ToolbarRoot } from 'reka-ui';
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const gateways = ref<GatewayResp[]>([]);
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);

  const isDeleteDialogOpen = ref(false);
  const isCreateDialogOpen = ref(false);
  const pendingDelete = ref<GatewayResp | null>(null);
  const createForm = reactive(emptyGatewayForm());
  const createErrors = reactive({
    code: '',
    name: '',
    rest_api_url: '',
    base_domain: '',
    image: '',
  });
  const imagePullPolicyValues = ['missing', 'always', 'never'];
  const entrypointValues = ['web', 'websecure'];
  const tlsModeValues = ['none', 'letsencrypt', 'tls'];
  const codePattern = /^[a-z][a-z0-9-]*$/;

  watch(
    () => projectStore.activeProjectId,
    () => {
      pagination.current = 1;
      fetchData();
    }
  );

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

  function openDeleteModal(item: GatewayResp) {
    pendingDelete.value = item;
    isDeleteDialogOpen.value = true;
  }

  function emptyGatewayForm() {
    return {
      name: 'Traefik',
      code: 'traefik',
      rest_api_url: 'http://localhost:8080',
      base_domain: 'lvh.me',
      initial_component_image: 'traefik:3.6',
      initial_component_pull_policy: 'missing',
      default_entrypoint: 'web',
      tls_mode: 'none',
    };
  }

  function openCreateDialog() {
    Object.assign(createForm, emptyGatewayForm());
    Object.assign(createErrors, {
      code: '',
      name: '',
      rest_api_url: '',
      base_domain: '',
      image: '',
    });
    isCreateDialogOpen.value = true;
  }

  function validateCreateForm() {
    createErrors.code = codePattern.test(createForm.code.trim())
      ? ''
      : t('gateway.validation.codeInvalid');
    createErrors.name = createForm.name.trim() ? '' : t('gateway.validation.nameRequired');
    createErrors.rest_api_url = createForm.rest_api_url.trim()
      ? ''
      : t('gateway.validation.restApiUrlRequired');
    createErrors.base_domain = createForm.base_domain.trim()
      ? ''
      : t('gateway.validation.baseDomainRequired');
    createErrors.image = createForm.initial_component_image.trim()
      ? ''
      : t('gateway.validation.imageRequired');
    return (
      !createErrors.code &&
      !createErrors.name &&
      !createErrors.rest_api_url &&
      !createErrors.base_domain &&
      !createErrors.image
    );
  }

  async function handleCreate() {
    if (!validateCreateForm()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('gateway.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        await gatewayApi.create(
          {
            project_id: projectId,
            code: createForm.code.trim(),
            name: createForm.name.trim(),
            rest_api_url: createForm.rest_api_url.trim(),
            base_domain: createForm.base_domain.trim(),
            initial_component_image: createForm.initial_component_image.trim(),
            initial_component_pull_policy: createForm.initial_component_pull_policy,
            default_entrypoint: createForm.default_entrypoint,
            tls_mode: createForm.tls_mode,
          },
          { project_id: projectId }
        );
        toast.success(t('gateway.toast.saveSuccess'));
        isCreateDialogOpen.value = false;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('gateway.toast.saveFailed'));
    }
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
