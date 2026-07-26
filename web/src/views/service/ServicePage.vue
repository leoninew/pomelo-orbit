<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" :aria-label="t('service.toolbar')">
      <div class="app-toolbar-row">
        <ComboboxSelect
          :model-value="query.application_id"
          :options="appSelectOptions"
          :placeholder="t('service.filterApplication')"
          width-class="app-toolbar-select"
          @update:model-value="handleApplicationChange"
        />
        <RawValueCombobox
          :model-value="query.status"
          :values="statusValues"
          :placeholder="t('service.filterStatus')"
          width-class="app-toolbar-select"
          @update:model-value="handleStatusChange"
        />
        <SearchControl
          v-model="query.search"
          :placeholder="t('service.searchPlaceholder')"
          :loading="status === 'loading'"
          class="shrink-0"
          @search="handleSearch"
        />
        <button class="app-button px-5" :disabled="status === 'loading'" @click="handleRefresh">
          <RefreshCw class="size-4" :class="{ 'animate-spin': status === 'loading' }" />
          {{ t('common.refresh') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="services.length === 0" :message="t('service.empty')" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[840px]">
          <thead>
            <tr>
              <th>{{ t('service.fields.instanceKey') }}</th>
              <th>{{ t('service.fields.application') }} / {{ t('service.fields.version') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('common.updatedAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="svc in services" :key="svc.id">
              <td>
                <router-link :to="`/service/${svc.id}`" class="app-link">
                  {{ svc.instance_key || 'default' }}
                </router-link>
              </td>
              <td>
                <div class="flex items-center gap-2 whitespace-nowrap">
                  <router-link :to="`/application/${svc.application_id}`" class="app-link">
                    {{ svc.application_name || svc.application_id }}
                  </router-link>
                  <span class="text-muted-foreground">/</span>
                  <router-link :to="`/version/${svc.version_id}`" class="app-link">
                    {{ svc.version_label || svc.version_id }}
                  </router-link>
                </div>
              </td>
              <td>
                <AppBadge variant="status" :tone="appStatusTone(svc.status)">
                  {{ svc.status }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(svc.updated_at) }}</td>
              <td>
                <div class="flex items-center gap-3">
                  <router-link
                    :to="{ path: '/deployments', query: { application_id: svc.application_id } }"
                    class="app-link"
                  >
                    {{ t('service.actions.deployments') }}
                  </router-link>
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
  </div>
</template>

<script setup lang="ts">
  import { RefreshCw } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { ToolbarRoot } from 'reka-ui';
  import { applicationApi } from '@/api/application/application';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import RawValueCombobox from '@/components/RawValueCombobox.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { useProjectStore } from '@/stores/project';
  import { appStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();

  const services = ref<ServiceResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);

  const query = reactive({
    search: '',
    application_id: '',
    status: '',
  });

  const appOptions = ref<ApplicationResp[]>([]);

  const appSelectOptions = computed(() => [
    { value: '', label: t('service.filterApplicationAll') },
    ...appOptions.value.map((app) => ({
      value: app.id,
      label: app.name,
      description: app.code,
    })),
  ]);

  const statusValues = ['deploying', 'running', 'stopped', 'faulted'];

  async function loadFilterOptions() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      appOptions.value = [];
      return;
    }
    try {
      const apps = await applicationApi.list({ per_page: 100, project_id: projectId });
      appOptions.value = apps.items ?? [];
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('service.toast.loadFailed'));
    }
  }

  async function fetchServices() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      services.value = [];
      pagination.total = 0;
      return;
    }
    try {
      await execute(async () => {
        const resp = await serviceApi.list({
          project_id: projectId,
          page: pagination.current,
          per_page: pagination.pageSize,
          application_id: query.application_id || undefined,
          status: query.status || undefined,
          search: query.search || undefined,
        });
        services.value = resp.items ?? [];
        pagination.total = resp.total ?? 0;
      });
    } catch {
      services.value = [];
      pagination.total = 0;
      toast.error(t('service.toast.loadFailed'));
    }
  }

  function handleRefresh() {
    void fetchServices();
  }

  function handleSearch() {
    pagination.current = 1;
    void fetchServices();
  }

  function handleApplicationChange(value: ComboboxOptionValue) {
    query.application_id = String(value || '');
    pagination.current = 1;
    void fetchServices();
  }

  function handleStatusChange(value: ComboboxOptionValue) {
    query.status = String(value || '');
    pagination.current = 1;
    void fetchServices();
  }

  function goPage(page: number) {
    pagination.current = page;
    void fetchServices();
  }

  function handlePageSizeChange(size: number) {
    pagination.pageSize = size;
    pagination.current = 1;
    void fetchServices();
  }

  watch(
    () => projectStore.activeProjectId,
    () => {
      query.application_id = '';
      query.status = '';
      query.search = '';
      pagination.current = 1;
      void loadFilterOptions();
      void fetchServices();
    }
  );

  onMounted(() => {
    void loadFilterOptions();
    void fetchServices();
  });
</script>
