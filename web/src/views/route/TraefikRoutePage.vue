<template>
  <AppLoadingState v-if="!projectReadinessChecked" size="section" />
  <div v-else-if="!projectReady" class="app-surface">
    <p v-if="projectReadinessError" class="py-16 text-center text-sm text-destructive">
      {{ projectReadinessError }}
    </p>
    <AppEmptyState v-else>
      <div class="flex flex-wrap items-center justify-center">
        <span>{{ t('route.initializationRequired') }}</span>
        <router-link
          v-if="projectStore.activeProjectId"
          :to="{
            name: 'ProjectInitialization',
            params: { id: projectStore.activeProjectId },
            query: { returnTo: '/route/traefik' },
          }"
          class="app-link"
        >
          {{ t('project.initialization.title') }}
        </router-link>
      </div>
    </AppEmptyState>
  </div>
  <div v-else class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('traefikRoute.toolbar')">
      <SearchControl
        v-model="searchText"
        :placeholder="t('traefikRoute.searchPlaceholder')"
        :loading="status === 'loading'"
        class="shrink-0"
        @search="handleSearch"
      />
      <router-link to="/routes" class="app-button ml-auto h-10 px-3">
        <ArrowLeft class="size-4" />
        {{ t('route.sections.customConfiguration') }}
      </router-link>
    </ToolbarRoot>

    <div v-if="status === 'loading'" class="app-surface">
      <AppLoadingState />
    </div>
    <div v-else-if="status === 'error'" class="app-surface py-16 text-center">
      <p class="text-sm text-destructive">{{ error || t('traefikRoute.toast.loadFailed') }}</p>
      <p class="mt-1 text-xs text-muted-foreground">{{ t('traefikRoute.serviceCheckHint') }}</p>
      <button class="app-link mx-auto mt-3 block text-sm" @click="fetchRoutes">
        {{ t('traefikRoute.retry') }}
      </button>
    </div>
    <div v-else class="app-surface">
      <AppEmptyState v-if="filteredRoutes.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table table-fixed min-w-[1080px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[10%]" />
            <col class="w-[9%]" />
            <col class="w-[27%]" />
            <col class="w-[17%]" />
            <col class="w-[9%]" />
            <col class="w-[8%]" />
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
            <tr v-for="traefikRoute in filteredRoutes" :key="traefikRoute.name">
              <td class="max-w-0 truncate text-foreground" :title="traefikRoute.name">
                {{ traefikRoute.name }}
              </td>
              <td class="whitespace-nowrap text-foreground">{{ traefikRoute.provider }}</td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="traefikRoute.status === 'enabled' ? 'success' : 'default'"
                >
                  {{ traefikRoute.status }}
                </AppBadge>
              </td>
              <td class="max-w-0" :title="traefikRoute.rule">
                <a
                  v-if="
                    !isTCPRouter(traefikRoute) && buildRouteUrl(traefikRoute.rule, traefikRoute.tls)
                  "
                  :href="buildRouteUrl(traefikRoute.rule, traefikRoute.tls)!"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="app-link flex items-center gap-1"
                >
                  <span class="truncate">{{ traefikRoute.rule }}</span>
                  <ExternalLink class="size-3 shrink-0" />
                </a>
                <span v-else class="block truncate text-foreground">{{ traefikRoute.rule }}</span>
              </td>
              <td class="max-w-0 truncate text-foreground" :title="traefikRoute.service">
                {{ traefikRoute.service }}
              </td>
              <td class="max-w-0" :title="traefikRoute.entrypoints.join(', ')">
                <div class="flex flex-nowrap gap-1 overflow-hidden">
                  <AppBadge
                    v-for="entrypoint in traefikRoute.entrypoints"
                    :key="entrypoint"
                    variant="pill"
                  >
                    {{ entrypoint }}
                  </AppBadge>
                </div>
              </td>
              <td class="whitespace-nowrap">
                <AppBadge
                  variant="status"
                  :tone="
                    isTCPRouter(traefikRoute) ? 'warning' : traefikRoute.tls ? 'info' : 'default'
                  "
                >
                  {{ isTCPRouter(traefikRoute) ? 'TCP' : traefikRoute.tls ? 'HTTPS' : 'HTTP' }}
                </AppBadge>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, ExternalLink } from '@lucide/vue';
  import { computed, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { projectInitializationApi } from '@/api/project/initialization';
  import { traefikRouteApi } from '@/api/route/traefik';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import type { TraefikRouterResp } from '@/gen/proto/orbit/v1/route/traefik';
  import { useProjectStore } from '@/stores/project';
  import { ToolbarRoot } from 'reka-ui';

  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const projectReady = ref(false);
  const projectReadinessChecked = ref(false);
  const projectReadinessError = ref('');
  const routes = ref<TraefikRouterResp[]>([]);
  const searchText = ref('');
  const appliedSearch = ref('');

  const filteredRoutes = computed(() => {
    const search = appliedSearch.value.trim().toLowerCase();
    if (!search) {
      return routes.value;
    }
    return routes.value.filter(
      (router) =>
        router.name.toLowerCase().includes(search) ||
        router.rule.toLowerCase().includes(search) ||
        router.service.toLowerCase().includes(search) ||
        router.provider.toLowerCase().includes(search)
    );
  });

  async function fetchRoutes() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      await execute(async () => {
        const data = await traefikRouteApi.list(projectId);
        if (projectStore.activeProjectId === projectId) {
          routes.value = data.items;
        }
      });
    } catch {
      // The page renders the error state from useStatusAsync.
    }
  }

  function handleSearch() {
    appliedSearch.value = searchText.value;
    void fetchRoutes();
  }

  function buildRouteUrl(rule: string, tls: boolean): string | null {
    const match = rule.match(/Host\(`([^`]+)`\)/);
    if (!match) {
      return null;
    }
    return `${tls ? 'https' : 'http'}://${match[1]}`;
  }

  function isTCPRouter(router: TraefikRouterResp): boolean {
    return router.rule.startsWith('HostSNI(');
  }

  async function loadProjectData(projectId: string | null) {
    projectReady.value = false;
    projectReadinessChecked.value = false;
    projectReadinessError.value = '';
    routes.value = [];
    if (!projectId) {
      projectReadinessChecked.value = true;
      return;
    }
    try {
      const readiness = await projectInitializationApi.getStatus(projectId);
      if (projectStore.activeProjectId !== projectId) {
        return;
      }
      projectReadinessChecked.value = true;
      if (readiness.status !== 'ready') {
        return;
      }
    } catch {
      if (projectStore.activeProjectId === projectId) {
        projectReadinessError.value = t('route.toast.loadFailed');
        projectReadinessChecked.value = true;
      }
      return;
    }
    projectReady.value = true;
    void fetchRoutes();
  }

  watch(
    () => projectStore.activeProjectId,
    (projectId) => void loadProjectData(projectId),
    { immediate: true }
  );
</script>
