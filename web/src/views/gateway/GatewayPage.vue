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
        <button class="app-button-primary px-5" @click="router.push('/gateway/create')">
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
              <th>{{ t('gateway.fields.defaultEntrypoint') }}</th>
              <th>{{ t('gateway.fields.tlsMode') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in gateways"
              :key="item.id"
              class="cursor-pointer"
              @click="router.push(`/gateway/${item.id}`)"
            >
              <td class="text-foreground">{{ item.name }}</td>
              <td class="text-foreground">{{ item.code }}</td>
              <td class="text-foreground">{{ item.rest_api_url }}</td>
              <td class="text-foreground">{{ item.base_domain }}</td>
              <td class="text-foreground">{{ item.default_entrypoint || '—' }}</td>
              <td class="text-foreground">{{ item.tls_mode || 'none' }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(item.created_at) }}
              </td>
              <td class="whitespace-nowrap" @click.stop>
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="router.push(`/gateway/${item.id}`)">
                    {{ t('application.view') }}
                  </button>
                  <button class="app-link" @click="router.push(`/gateway/${item.id}/edit`)">
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
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
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

  const isDeleteDialogOpen = ref(false);
  const pendingDelete = ref<GatewayResp | null>(null);

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
