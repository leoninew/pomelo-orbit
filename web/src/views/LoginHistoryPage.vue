<template>
  <div class="space-y-6">
    <ToolbarRoot class="flex items-center" aria-label="登录历史工具栏">
      <SearchControl
        v-model="searchText"
        :placeholder="t('loginHistory.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
    </ToolbarRoot>

    <!-- Table Card -->
    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="history.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[920px]">
          <thead>
            <tr>
              <th>{{ t('loginHistory.loginTime') }}</th>
              <th>{{ t('loginHistory.username') }}</th>
              <th>{{ t('loginHistory.ipAddress') }}</th>
              <th>{{ t('loginHistory.userAgent') }}</th>
              <th>{{ t('common.status') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="record in history" :key="record.id">
              <td class="text-foreground">{{ formatTime(record.login_at) }}</td>
              <td class="text-foreground">{{ record.username }}</td>
              <td class="text-foreground">{{ record.ip_address }}</td>
              <td class="max-w-md truncate text-foreground" :title="record.user_agent || undefined">
                {{ record.user_agent || '-' }}
              </td>
              <td>
                <AppBadge variant="status" :tone="record.success ? 'success' : 'error'">
                  {{
                    record.success
                      ? t('loginHistory.statusSuccess')
                      : t('loginHistory.statusFailed')
                  }}
                </AppBadge>
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
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { authApi } from '@/api/auth';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { LoginHistoryResp } from '@/gen/proto/orbit/api/v1/auth';
  import { formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const { t } = useI18n();
  const toast = useToast();
  const { status, execute } = useStatusAsync();
  const history = ref<LoginHistoryResp[]>([]);
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  async function fetchHistory() {
    try {
      await execute(async () => {
        const res = await authApi.listLoginHistory({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        history.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('loginHistory.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchHistory();
  }
  function goPage(p: number) {
    pagination.current = p;
    fetchHistory();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchHistory();
  }

  onMounted(fetchHistory);
</script>
