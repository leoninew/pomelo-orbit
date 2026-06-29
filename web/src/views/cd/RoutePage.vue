<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('route.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('route.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateModal">
          <Plus class="size-4" />
          {{ t('route.addRoute') }}
        </button>
        <button class="app-button px-5" :disabled="operating" @click="handleSync">
          <RefreshCw class="size-4" :class="{ 'animate-spin': operating }" />
          {{ t('route.syncAll') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="routes.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[1200px]">
          <colgroup>
            <col class="w-[14%]" />
            <col class="w-[16%]" />
            <col class="w-[10%]" />
            <col class="w-[20%]" />
            <col class="w-[8%]" />
            <col class="w-[8%]" />
            <col class="w-[14%]" />
            <col class="w-[10%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('route.fields.name') }}</th>
              <th>{{ t('route.fields.domain') }}</th>
              <th>{{ t('route.fields.pathPrefix') }}</th>
              <th>{{ t('route.fields.targetUrl') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('route.fields.protocol') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="route in routes" :key="route.id">
              <td>
                <router-link :to="`/cd/routes/${route.id}`" class="app-link whitespace-nowrap">
                  {{ route.name }}
                </router-link>
              </td>
              <td>
                <a
                  :href="`${route.https_enabled ? 'https' : 'http'}://${route.domain}`"
                  target="_blank"
                  class="app-link inline-flex items-center gap-1 whitespace-nowrap"
                >
                  {{ route.domain }}
                  <ExternalLink class="size-3" />
                </a>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ route.path_prefix }}</td>
              <td class="max-w-0 truncate text-foreground" :title="route.target_url">
                {{ route.target_url }}
              </td>
              <td>
                <AppBadge variant="status" :tone="route.enabled ? 'success' : 'default'">
                  {{ route.enabled ? t('route.status.enabled') : t('route.status.disabled') }}
                </AppBadge>
              </td>
              <td>
                <AppBadge variant="status" :tone="route.https_enabled ? 'info' : 'default'">
                  {{ route.https_enabled ? 'HTTPS' : 'HTTP' }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(route.created_at) }}</td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <router-link :to="`/cd/routes/${route.id}`" class="app-link">
                    {{ t('application.view') }}
                  </router-link>
                  <button
                    v-if="!route.enabled"
                    class="app-link-success"
                    @click="handleEnable(route.id)"
                  >
                    {{ t('route.status.enabled') }}
                  </button>
                  <button v-else class="app-link-warning" @click="handleDisable(route.id)">
                    {{ t('route.status.disabled') }}
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
  </div>

  <AppDialog v-model:open="isCreateDialogOpen" :title="t('route.addRoute')">
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('route.fields.name') }}</label>
        <input
          v-model="form.name"
          type="text"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
          placeholder="example-route"
        />
        <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        <p v-else class="app-field-hint">{{ t('route.hints.name') }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('route.fields.domain') }}</label>
        <input
          v-model="form.domain"
          type="text"
          class="app-input"
          :class="errors.domain ? 'app-input-error' : ''"
          placeholder="example.com"
        />
        <p v-if="errors.domain" class="app-field-error text-xs">{{ errors.domain }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('route.fields.pathPrefix') }}</label>
        <input v-model="form.path_prefix" type="text" class="app-input" placeholder="/" />
        <p class="app-field-hint">{{ t('route.hints.pathPrefix') }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('route.fields.targetUrl') }}</label>
        <input
          v-model="form.target_url"
          type="text"
          class="app-input"
          :class="errors.target_url ? 'app-input-error' : ''"
          placeholder="http://host:port"
        />
        <p v-if="errors.target_url" class="app-field-error text-xs">
          {{ errors.target_url }}
        </p>
        <p v-else class="app-field-hint">{{ t('route.hints.targetUrl') }}</p>
      </div>
      <label class="flex cursor-pointer items-center gap-3">
        <SwitchRoot v-model:checked="form.enabled" class="app-switch-root">
          <SwitchThumb class="app-switch-thumb" />
        </SwitchRoot>
        <span class="text-sm text-foreground">{{ t('route.status.enabled') }}</span>
      </label>
    </div>

    <template #footer>
      <button class="app-button" @click="isCreateDialogOpen = false">
        {{ t('common.cancel') }}
      </button>
      <button class="app-button-primary" :disabled="operating" @click="handleSave">
        {{ t('common.add') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { ExternalLink, Plus, RefreshCw } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { SwitchRoot, SwitchThumb, ToolbarRoot } from 'reka-ui';
  import type { Route } from '@/api/cd/route';
  import { routeApi } from '@/api/cd/route';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const routes = ref<Route[]>([]);
  const searchText = ref('');
  const isCreateDialogOpen = ref(false);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  const form = reactive({
    name: '',
    domain: '',
    path_prefix: '/',
    target_url: 'http://',
    enabled: false,
  });
  const errors = reactive({ name: '', domain: '', target_url: '' });

  function validate() {
    errors.name = /^[a-z][a-z0-9._-]*$/.test(form.name) ? '' : t('route.validation.nameInvalid');
    errors.domain = form.domain.trim() ? '' : t('route.validation.domainRequired');
    errors.target_url = /^https?:\/\/[a-zA-Z0-9.-]+:\d+$/.test(form.target_url)
      ? ''
      : t('route.validation.targetUrlInvalid');
    return !errors.name && !errors.domain && !errors.target_url;
  }

  async function fetchData() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await routeApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
          project_id: projectId,
        });
        routes.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('route.toast.loadFailed'));
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
    Object.assign(form, {
      name: '',
      domain: '',
      path_prefix: '/',
      target_url: 'http://',
      enabled: false,
    });
    Object.assign(errors, { name: '', domain: '', target_url: '' });
    isCreateDialogOpen.value = true;
  }

  async function handleSave() {
    if (!validate()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        await routeApi.create(form, { project_id: projectId });
        toast.success(t('route.toast.addSuccess'));
        isCreateDialogOpen.value = false;
        fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.addFailed'));
    }
  }

  async function handleEnable(id: string) {
    try {
      await executeOp(async () => {
        await routeApi.enable(id);
        toast.success(t('route.toast.enableSuccess'));
        fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.enableFailed'));
    }
  }

  async function handleDisable(id: string) {
    try {
      await executeOp(async () => {
        await routeApi.disable(id);
        toast.success(t('route.toast.disableSuccess'));
        fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.disableFailed'));
    }
  }

  async function handleSync() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        await routeApi.sync({ project_id: projectId });
        toast.success(t('route.syncSuccess'));
        fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.syncFailed'));
    }
  }

  onMounted(fetchData);
</script>
