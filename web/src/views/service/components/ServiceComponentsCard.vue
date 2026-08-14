<template>
  <DetailInfoCard :title="t('application.detail.fields.components')">
    <template #actions>
      <SearchControl
        v-model="searchText"
        :placeholder="t('service.searchComponentsPlaceholder')"
        :loading="loading"
        class="shrink-0"
        @search="handleSearch"
      />
    </template>
    <AppEmptyState v-if="filteredComponents.length === 0" size="compact" />
    <div v-else class="overflow-x-auto">
      <table class="app-data-table min-w-[1180px]">
        <thead>
          <tr>
            <th>{{ t('application.detail.fields.component') }}</th>
            <th>{{ t('service.fields.container') }}</th>
            <th>{{ t('application.detail.fields.image') }}</th>
            <th>{{ t('service.fields.ports') }}</th>
            <th>{{ t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="component in filteredComponents" :key="component.id">
            <td>
              <router-link
                :to="`/service/${service.id}/component/${component.id}`"
                class="app-link"
              >
                {{ component.component_name }}
              </router-link>
            </td>
            <td class="max-w-sm break-all text-muted-foreground">
              {{ component.container_name }}
            </td>
            <td
              class="w-[480px] max-w-[480px] truncate text-muted-foreground"
              :title="component.image"
            >
              {{ component.image }}
            </td>
            <td class="min-w-72">
              <div
                v-if="publishedServiceComponentEndpoints(component.effective_endpoints).length"
                class="flex flex-wrap gap-1.5"
              >
                <div
                  v-for="endpoint in publishedServiceComponentEndpoints(
                    component.effective_endpoints
                  )"
                  :key="`${endpoint.protocol}:${endpoint.container_port}`"
                  class="flex items-center gap-2 text-sm"
                >
                  <AppBadge variant="pill">{{ endpoint.mode }}</AppBadge>
                  <span class="text-foreground">
                    {{ endpoint.protocol }}/{{ endpoint.container_port }}
                  </span>
                </div>
              </div>
            </td>
            <td>
              <div class="flex items-center gap-3">
                <router-link
                  class="app-link"
                  :to="`/service/${service.id}/component/${component.id}`"
                >
                  {{ t('common.edit') }}
                </router-link>
                <button
                  class="app-link"
                  type="button"
                  @click="emit('view-logs', component.component_name)"
                >
                  {{ t('service.actions.logs') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </DetailInfoCard>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { publishedServiceComponentEndpoints } from './serviceComponentPorts';

  const props = withDefaults(
    defineProps<{
      service: ServiceResp;
      loading?: boolean;
    }>(),
    {
      loading: false,
    }
  );

  const emit = defineEmits<{
    'view-logs': [component: string];
    search: [];
  }>();

  const { t } = useI18n();
  const searchText = ref('');
  const appliedSearch = ref('');

  const filteredComponents = computed(() => {
    const keyword = appliedSearch.value.trim().toLowerCase();
    if (!keyword) {
      return props.service.components;
    }
    return props.service.components.filter(
      (component) =>
        component.component_name.toLowerCase().includes(keyword) ||
        component.container_name.toLowerCase().includes(keyword) ||
        component.image.toLowerCase().includes(keyword)
    );
  });

  function handleSearch() {
    appliedSearch.value = searchText.value;
    emit('search');
  }
</script>
