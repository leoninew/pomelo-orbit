<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" :aria-label="t('deployment.toolbar')">
      <div class="app-toolbar-row">
        <ComboboxSelect
          :model-value="query.application_id"
          :options="appSelectOptions"
          :placeholder="t('deployment.filterApplication')"
          width-class="app-toolbar-select"
          @update:model-value="handleApplicationChange"
        />
        <button class="app-button h-9 px-3" type="button" @click="router.push('/dialogue')">
          <MessageSquareText :size="16" aria-hidden="true" />
          <span>{{ t('deploymentDialogue.open') }}</span>
        </button>
      </div>
    </ToolbarRoot>

    <!-- Table Card -->
    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('deployment.toast.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="deployments.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1040px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('deployment.fields.service') }}</th>
              <th>{{ t('deployment.fields.operationType') }}</th>
              <th>{{ t('deployment.fields.triggerType') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('deployment.fields.startTime') }}</th>
              <th>{{ t('deployment.fields.duration') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="deployment in deployments" :key="deployment.id">
              <td class="max-w-64 truncate">
                <router-link
                  :to="`/deployment/${deployment.id}`"
                  class="app-link font-mono"
                  :title="deployment.id"
                >
                  {{ deployment.id }}
                </router-link>
              </td>
              <td>
                <router-link
                  v-if="deployment.service_id"
                  :to="`/service/${deployment.service_id}`"
                  class="app-link"
                >
                  {{
                    deployment.application_name ||
                    deployment.application_id ||
                    deployment.service_id
                  }}
                </router-link>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <td>
                <AppBadge variant="pill">{{ deployment.operation_type }}</AppBadge>
              </td>
              <td>
                <AppBadge variant="pill">{{ deployment.trigger_type }}</AppBadge>
              </td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="statusTone(deployment.status)"
                  :title="
                    deployment.status === 'faulted'
                      ? deployment.error_message || undefined
                      : undefined
                  "
                >
                  {{ deployment.status }}
                </AppBadge>
              </td>
              <td class="text-foreground">{{ formatTime(deployment.started_at) }}</td>
              <td class="text-foreground">
                {{ formatDuration(deployment.started_at, deployment.finished_at) }}
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
  import { useRoute, useRouter } from 'vue-router';
  import { MessageSquareText } from 'lucide-vue-next';
  import { applicationApi } from '@/api/application/application';
  import { deploymentApi } from '@/api/deployment/deployment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { DeploymentResp } from '@/gen/proto/orbit/v1/deployment/deployment';
  import { statusTone } from '@/utils/status';
  import { formatDuration, formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();

  const deployments = ref<DeploymentResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  const query = reactive({
    application_id: (route.query.application_id as string) || '',
  });

  const appOptions = ref<ApplicationResp[]>([]);
  const appSelectOptions = computed(() =>
    appOptions.value.map((app) => ({
      value: app.id,
      label: app.name,
      description: app.code,
    }))
  );

  async function loadApps() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      const resp = await applicationApi.list({ per_page: 100, project_id: projectId });
      appOptions.value = resp.items;
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('application.toast.loadFailed'));
    }
  }

  async function fetchDeployments() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await deploymentApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          application_id: query.application_id || undefined,
          project_id: projectId,
        });
        deployments.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('deployment.toast.loadFailed'));
    }
  }

  function handleApplicationChange(value: string | number | boolean) {
    const nextValue = String(value || '');
    if (query.application_id === nextValue) {
      return;
    }
    query.application_id = nextValue;
    pagination.current = 1;
    fetchDeployments();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchDeployments();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchDeployments();
  }

  onMounted(async () => {
    fetchDeployments();
    await loadApps();
  });
</script>
