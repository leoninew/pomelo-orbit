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
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('gateway.toast.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="gateways.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[960px]">
          <thead>
            <tr>
              <th>{{ t('gateway.fields.name') }}</th>
              <th>{{ t('gateway.fields.code') }}</th>
              <th>{{ t('gateway.sections.ingressDefaults') }}</th>
              <th>{{ t('gateway.sections.routeCertificates') }}</th>
              <th>{{ t('gateway.fields.baseDomain') }}</th>
              <th>{{ t('common.updatedAt') }}</th>
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
              <td>
                <div class="flex flex-wrap gap-1.5">
                  <AppBadge variant="pill">{{ item.default_entrypoint }}</AppBadge>
                  <AppBadge variant="pill">{{ item.tls_mode }}</AppBadge>
                </div>
              </td>
              <td>
                <div class="flex flex-wrap gap-1.5">
                  <AppBadge variant="pill">{{ acmeProfileLabel(item.acme_profile) }}</AppBadge>
                </div>
              </td>
              <td class="text-foreground">{{ item.base_domain }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(item.config_updated_at) }}
              </td>
              <td class="whitespace-nowrap">
                <button
                  class="app-link-danger"
                  :disabled="operating"
                  @click="openDeleteDialog(item)"
                >
                  {{ t('common.delete') }}
                </button>
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

    <GatewayExposuresCard
      v-if="exposureGateway"
      :gateway="exposureGateway"
      :gateway-options="exposureGatewayOptions"
      :selected-gateway-id="exposureGatewayId"
      :loading="exposureLoading"
      @update:selected-gateway-id="selectExposureGateway"
    />

    <AppDialog v-model:open="isDeleteDialogOpen" :title="t('gateway.dialog.delete')">
      <p class="text-sm text-muted-foreground">
        {{ t('gateway.dialog.deleteConfirm', { name: pendingDelete?.name || '' }) }}
      </p>
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
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
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { ToolbarRoot } from 'reka-ui';
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';
  import GatewayExposuresCard from './components/GatewayExposuresCard.vue';

  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: exposureLoading, execute: executeExposure } = useStatusAsync();
  const gateways = ref<GatewayResp[]>([]);
  const exposureGateway = ref<GatewayResp>();
  const exposureGatewayId = ref('');
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);
  const exposureGatewayOptions = computed(() =>
    gateways.value.map((gateway) => ({
      value: gateway.id,
      label: `${gateway.name} (${gateway.code})`,
    }))
  );
  const isDeleteDialogOpen = ref(false);
  const pendingDelete = ref<GatewayResp>();
  const deleteSubmitError = ref('');
  let exposureRequest = 0;

  watch(
    () => projectStore.activeProjectId,
    () => {
      pagination.current = 1;
      clearExposureGateway();
      void fetchData();
    }
  );

  async function fetchData() {
    const projectID = projectStore.activeProjectId;
    if (!projectID) {
      gateways.value = [];
      pagination.total = 0;
      clearExposureGateway();
      return;
    }
    try {
      await execute(async () => {
        const result = await gatewayApi.list({
          project_id: projectID,
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        gateways.value = result.items ?? [];
        pagination.total = result.total ?? 0;
      });
      syncExposureGateway();
    } catch {
      toast.error(t('gateway.toast.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    void fetchData();
  }

  function acmeProfileLabel(profile: string) {
    switch (profile) {
      case 'http':
        return t('gateway.acmeProfiles.http');
      case 'dns':
        return t('gateway.acmeProfiles.dns');
      case 'http-dns':
        return t('gateway.acmeProfiles.httpDns');
      default:
        return t('gateway.acmeProfiles.none');
    }
  }

  function clearExposureGateway() {
    exposureRequest += 1;
    exposureGatewayId.value = '';
    exposureGateway.value = undefined;
  }

  function syncExposureGateway() {
    const selected = gateways.value.find((item) => item.id === exposureGatewayId.value);
    const id = selected?.id || gateways.value[0]?.id || '';
    if (id === exposureGatewayId.value && exposureGateway.value) {
      return;
    }
    void selectExposureGateway(id);
  }

  async function selectExposureGateway(id: string) {
    exposureGatewayId.value = id;
    const request = ++exposureRequest;
    if (!id) {
      exposureGateway.value = undefined;
      return;
    }
    if (exposureGateway.value?.id !== id) {
      exposureGateway.value = undefined;
    }
    try {
      const detail = await executeExposure(() => gatewayApi.get(id));
      if (request === exposureRequest && detail) {
        exposureGateway.value = detail;
      }
    } catch {
      if (request === exposureRequest) {
        exposureGateway.value = undefined;
        toast.error(t('gateway.toast.loadDetailFailed'));
      }
    }
  }

  function goPage(page: number) {
    pagination.current = page;
    void fetchData();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    void fetchData();
  }

  function openDeleteDialog(value: GatewayResp) {
    pendingDelete.value = value;
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  async function handleDelete() {
    const item = pendingDelete.value;
    if (!item) return;
    deleteSubmitError.value = '';
    try {
      await executeOp(async () => {
        await gatewayApi.delete(item.id);
        toast.success(t('gateway.toast.deleteSuccess'));
        isDeleteDialogOpen.value = false;
        await fetchData();
      });
    } catch (error) {
      deleteSubmitError.value =
        error instanceof Error ? error.message : t('gateway.toast.deleteFailed');
    }
  }

  onMounted(fetchData);
</script>
