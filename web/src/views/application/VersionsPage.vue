<template>
  <div class="space-y-6">
    <h1 class="text-base font-semibold text-foreground">
      {{ t('nav.versions') }}
    </h1>
    <ToolbarRoot class="app-toolbar-scroll" :aria-label="t('application.toolbar')">
      <div class="app-toolbar-row">
        <ComboboxSelect
          :model-value="applicationId"
          :options="applicationOptions"
          :placeholder="t('deployment.filterApplication')"
          width-class="app-toolbar-select"
          :disabled="loadingApplications"
          @update:model-value="handleApplicationChange"
        />
        <SearchControl
          v-model="search"
          :placeholder="t('application.searchPlaceholder')"
          class="shrink-0"
          @search="handleSearch"
        />
      </div>
    </ToolbarRoot>

    <div v-if="loadingApplications" class="app-surface">
      <AppLoadingState />
    </div>
    <div v-else-if="loadError" class="app-surface">
      <div class="py-16 text-center text-destructive">
        <p class="text-sm">{{ loadError }}</p>
      </div>
    </div>
    <AppEmptyState v-else-if="applications.length === 0" />
    <ApplicationDetail
      v-else-if="applicationId"
      :key="`${applicationId}:${search}`"
      :application-id="applicationId"
      :version-search="search"
      versions-only
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { ToolbarRoot } from 'reka-ui';
  import { applicationApi } from '@/api/application/application';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useToast } from '@/composables/useToast';
  import { applicationVersionsApplicationIdKey } from '@/constants/application';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import { useProjectStore } from '@/stores/project';
  import { useStorageStore } from '@/stores/storage';
  import ApplicationDetail from './ApplicationDetail.vue';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const storageStore = useStorageStore();
  const applications = ref<ApplicationResp[]>([]);
  const loadingApplications = ref(false);
  const loadError = ref('');
  const search = ref('');

  const applicationId = computed(() => String(route.query.application_id || ''));
  const applicationOptions = computed(() =>
    applications.value.map((application) => ({
      value: application.id,
      label: application.name,
    }))
  );

  function rememberApplication(projectId: string, id: string) {
    storageStore.setItem(applicationVersionsApplicationIdKey(projectId), id);
  }

  function resolveApplicationId(projectId: string, items: ApplicationResp[]): string {
    if (items.length === 0) {
      return '';
    }
    const ids = new Set(items.map((item) => item.id));
    const fromQuery = String(route.query.application_id || '');
    if (fromQuery && ids.has(fromQuery)) {
      return fromQuery;
    }
    const fromStorage = storageStore.getItem<string>(
      applicationVersionsApplicationIdKey(projectId)
    );
    if (fromStorage && ids.has(fromStorage)) {
      return fromStorage;
    }
    return items[0].id;
  }

  async function syncApplicationQuery(id: string) {
    const nextQuery = id ? { application_id: id } : {};
    const current = String(route.query.application_id || '');
    if (current === id) {
      return;
    }
    await router.replace({ path: '/versions', query: nextQuery });
  }

  async function loadApplications() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      applications.value = [];
      await syncApplicationQuery('');
      return;
    }
    loadingApplications.value = true;
    loadError.value = '';
    try {
      const response = await applicationApi.list({ per_page: 100, project_id: projectId });
      applications.value = response.items ?? [];
      const selected = resolveApplicationId(projectId, applications.value);
      if (selected) {
        rememberApplication(projectId, selected);
      }
      await syncApplicationQuery(selected);
    } catch {
      applications.value = [];
      loadError.value = t('application.toast.loadFailed');
      toast.error(loadError.value);
    } finally {
      loadingApplications.value = false;
    }
  }

  function handleApplicationChange(value: ComboboxOptionValue) {
    const id = String(value || '');
    search.value = '';
    const projectId = projectStore.activeProjectId;
    if (projectId && id) {
      rememberApplication(projectId, id);
    }
    void syncApplicationQuery(id);
  }

  function handleSearch() {}

  watch(
    () => projectStore.activeProjectId,
    () => {
      search.value = '';
      void loadApplications();
    }
  );

  onMounted(() => {
    void loadApplications();
  });
</script>
