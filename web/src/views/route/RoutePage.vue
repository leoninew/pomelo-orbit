<template>
  <div class="flex flex-col gap-4">
    <section class="app-surface app-detail-card">
      <div class="app-section-header app-detail-section-header">
        <h2 class="app-detail-section-title">{{ t('route.sections.traefikRouters') }}</h2>
        <ToolbarRoot
          class="flex w-full flex-wrap items-center justify-between gap-3 lg:w-auto lg:flex-nowrap"
          :aria-label="t('traefikRoute.toolbar')"
        >
          <SearchControl
            v-model="traefikSearchText"
            class="w-full sm:w-[360px]"
            :placeholder="t('traefikRoute.searchPlaceholder')"
            :loading="traefikStatus === 'loading'"
          />
          <div class="flex flex-wrap items-center gap-2">
            <button class="app-button-primary h-9 px-3" @click="openDashboard">
              <ExternalLink class="size-4" />
              {{ t('traefikRoute.openDashboard') }}
            </button>
            <button
              class="app-button h-9 px-3"
              :disabled="traefikStatus === 'loading'"
              @click="fetchTraefikRoutes"
            >
              <RefreshCw class="size-4" :class="{ 'animate-spin': traefikStatus === 'loading' }" />
              {{ t('common.refresh') }}
            </button>
          </div>
        </ToolbarRoot>
      </div>

      <AppSpinner v-if="traefikStatus === 'loading'" class="py-16" />
      <div v-else-if="traefikStatus === 'error'" class="py-16 text-center">
        <p class="text-sm text-destructive">
          {{ traefikError || t('traefikRoute.toast.loadFailed') }}
        </p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('traefikRoute.serviceCheckHint') }}</p>
        <button class="app-link mx-auto mt-3 block text-sm" @click="fetchTraefikRoutes">
          {{ t('traefikRoute.retry') }}
        </button>
      </div>
      <AppEmptyState v-else-if="filteredTraefikRoutes.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1080px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[10%]" />
            <col class="w-[8%]" />
            <col class="w-[30%]" />
            <col class="w-[18%]" />
            <col class="w-[10%]" />
            <col class="w-[4%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('traefikRoute.fields.name') }}</th>
              <th>{{ t('traefikRoute.fields.provider') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('traefikRoute.fields.rule') }}</th>
              <th>{{ t('traefikRoute.fields.service') }}</th>
              <th>{{ t('traefikRoute.fields.entrypoints') }}</th>
              <th>{{ t('traefikRoute.fields.protocol') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="router in filteredTraefikRoutes" :key="router.name">
              <td class="max-w-0 truncate text-foreground" :title="router.name">
                {{ router.name }}
              </td>
              <td class="whitespace-nowrap text-foreground">{{ router.provider }}</td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="router.status === 'enabled' ? 'success' : 'default'"
                >
                  {{ router.status }}
                </AppBadge>
              </td>
              <td class="max-w-0" :title="router.rule">
                <a
                  v-if="buildRouteUrl(router.rule, router.tls)"
                  :href="buildRouteUrl(router.rule, router.tls)!"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="app-link flex items-center gap-1"
                >
                  <span class="truncate">{{ router.rule }}</span>
                  <ExternalLink class="size-3 shrink-0" />
                </a>
                <span v-else class="block truncate text-foreground">{{ router.rule }}</span>
              </td>
              <td class="max-w-0 truncate text-foreground" :title="router.service">
                {{ router.service }}
              </td>
              <td class="max-w-0" :title="router.entrypoints.join(', ')">
                <div class="flex flex-nowrap gap-1 overflow-hidden">
                  <AppBadge
                    v-for="entrypoint in router.entrypoints"
                    :key="entrypoint"
                    variant="pill"
                  >
                    {{ entrypoint }}
                  </AppBadge>
                </div>
              </td>
              <td class="whitespace-nowrap">
                <AppBadge variant="status" :tone="router.tls ? 'info' : 'default'">
                  {{ router.tls ? 'HTTPS' : 'HTTP' }}
                </AppBadge>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="app-surface app-detail-card">
      <div class="app-section-header app-detail-section-header">
        <h2 class="app-detail-section-title">{{ t('route.sections.customConfiguration') }}</h2>
        <ToolbarRoot
          class="flex w-full flex-wrap items-center justify-between gap-3 lg:w-auto lg:flex-nowrap"
          :aria-label="t('route.toolbar')"
        >
          <SearchControl
            v-model="routeSearchText"
            class="w-full sm:w-[360px]"
            :placeholder="t('route.searchPlaceholder')"
            :loading="routeStatus === 'loading'"
            @search="handleRouteSearch"
          />
          <div class="flex flex-wrap items-center gap-2">
            <button
              class="app-button-primary h-9 px-3"
              :disabled="routeOperating"
              @click="openCreateModal"
            >
              <Plus class="size-4" />
              {{ t('route.addRoute') }}
            </button>
            <button class="app-button h-9 px-3" :disabled="routeOperating" @click="handleSync">
              <RefreshCw class="size-4" :class="{ 'animate-spin': routeOperating }" />
              {{ t('route.syncAll') }}
            </button>
          </div>
        </ToolbarRoot>
      </div>

      <AppSpinner v-if="routeStatus === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="routes.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1200px]">
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
                <router-link :to="`/route/${route.id}`" class="app-link whitespace-nowrap">
                  {{ route.name }}
                </router-link>
              </td>
              <td>
                <a
                  :href="`${route.https_enabled ? 'https' : 'http'}://${route.domain}`"
                  target="_blank"
                  rel="noopener noreferrer"
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
                  <button
                    v-if="!route.enabled"
                    class="app-link-success"
                    :disabled="routeOperating"
                    @click="handleEnable(route.id)"
                  >
                    {{ t('route.status.enabled') }}
                  </button>
                  <button
                    v-else
                    class="app-link-warning"
                    :disabled="routeOperating"
                    @click="handleDisable(route.id)"
                  >
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
    </section>
  </div>

  <AppDialog v-model:open="isCreateDialogOpen" :title="t('route.addRoute')">
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('route.fields.name') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.name"
          type="text"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
          placeholder="example-route"
          :aria-invalid="errors.name ? 'true' : undefined"
          @input="errors.name = ''"
        />
        <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        <p v-else class="app-field-hint">{{ t('route.hints.name') }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('route.fields.domain') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.domain"
          type="text"
          class="app-input"
          :class="errors.domain ? 'app-input-error' : ''"
          placeholder="example.com"
          :aria-invalid="errors.domain ? 'true' : undefined"
          @input="errors.domain = ''"
        />
        <p v-if="errors.domain" class="app-field-error text-xs">{{ errors.domain }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('route.fields.pathPrefix') }}</label>
        <input v-model="form.path_prefix" type="text" class="app-input" placeholder="/" />
        <p class="app-field-hint">{{ t('route.hints.pathPrefix') }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('route.fields.targetUrl') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.target_url"
          type="text"
          class="app-input"
          :class="errors.target_url ? 'app-input-error' : ''"
          placeholder="http://host:port"
          :aria-invalid="errors.target_url ? 'true' : undefined"
          @input="errors.target_url = ''"
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
      <AppDialogActions
        :busy="routeOperating"
        @cancel="isCreateDialogOpen = false"
        @confirm="handleSave"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { ExternalLink, Plus, RefreshCw } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { SwitchRoot, SwitchThumb, ToolbarRoot } from 'reka-ui';
  import { routeApi } from '@/api/route/route';
  import { traefikRouteApi } from '@/api/route/traefik';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
  import type { TraefikRouterResp } from '@/gen/proto/orbit/v1/route/traefik';
  import { useProjectStore } from '@/stores/project';
  import { useToast } from '@/composables/useToast';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status: routeStatus, execute: executeRoutes } = useStatusAsync();
  const { loading: routeOperating, execute: executeRouteOperation } = useStatusAsync();
  const { status: traefikStatus, error: traefikError, execute: executeTraefik } = useStatusAsync();

  const routes = ref<RouteResp[]>([]);
  const traefikRoutes = ref<TraefikRouterResp[]>([]);
  const routeSearchText = ref('');
  const traefikSearchText = ref('');
  const isCreateDialogOpen = ref(false);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  const filteredTraefikRoutes = computed(() => {
    if (!traefikSearchText.value.trim()) {
      return traefikRoutes.value;
    }
    const search = traefikSearchText.value.toLowerCase();
    return traefikRoutes.value.filter(
      (router) =>
        router.name.toLowerCase().includes(search) ||
        router.rule.toLowerCase().includes(search) ||
        router.service.toLowerCase().includes(search) ||
        router.provider.toLowerCase().includes(search)
    );
  });

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

  async function fetchRoutes() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeRoutes(async () => {
        const res = await routeApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: routeSearchText.value || undefined,
          project_id: projectId,
        });
        routes.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('route.toast.loadFailed'));
    }
  }

  async function fetchTraefikRoutes() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('traefikRoute.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeTraefik(async () => {
        const data = await traefikRouteApi.list({ project_id: projectId });
        traefikRoutes.value = data.items;
      });
    } catch {
      // The card renders the error state from useStatusAsync.
    }
  }

  function handleRouteSearch() {
    pagination.current = 1;
    fetchRoutes();
  }

  function goPage(page: number) {
    pagination.current = page;
    fetchRoutes();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchRoutes();
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
      await executeRouteOperation(async () => {
        await routeApi.create(form, { project_id: projectId });
        toast.success(t('route.toast.addSuccess'));
        isCreateDialogOpen.value = false;
        fetchRoutes();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.addFailed'));
    }
  }

  async function handleEnable(id: string) {
    try {
      await executeRouteOperation(async () => {
        await routeApi.enable(id, {});
        toast.success(t('route.toast.enableSuccess'));
        fetchRoutes();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.enableFailed'));
    }
  }

  async function handleDisable(id: string) {
    try {
      await executeRouteOperation(async () => {
        await routeApi.disable(id, {});
        toast.success(t('route.toast.disableSuccess'));
        fetchRoutes();
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
      await executeRouteOperation(async () => {
        await routeApi.sync({}, { project_id: projectId });
        toast.success(t('route.syncSuccess'));
        fetchRoutes();
        fetchTraefikRoutes();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.syncFailed'));
    }
  }

  function buildRouteUrl(rule: string, tls: boolean): string | null {
    const match = rule.match(/Host\(`([^`]+)`\)/);
    if (!match) {
      return null;
    }
    return `${tls ? 'https' : 'http'}://${match[1]}`;
  }

  async function openDashboard() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('traefikRoute.toast.selectProjectRequired'));
      return;
    }
    try {
      const config = await traefikRouteApi.getConfig({ project_id: projectId });
      window.open(
        `${config.https_enabled ? 'https' : 'http'}://${config.dashboard_domain}/dashboard/`,
        '_blank'
      );
    } catch {
      toast.error(t('traefikRoute.toast.openDashboardFailed'));
    }
  }

  onMounted(() => {
    fetchTraefikRoutes();
    fetchRoutes();
  });
</script>
