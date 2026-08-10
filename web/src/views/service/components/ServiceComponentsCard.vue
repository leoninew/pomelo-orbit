<template>
  <DetailInfoCard :title="t('application.detail.fields.components')">
    <AppEmptyState v-if="service.components.length === 0" size="compact" />
    <div v-else class="overflow-x-auto">
      <table class="app-data-table min-w-[980px]">
        <thead>
          <tr>
            <th>{{ t('application.detail.fields.component') }}</th>
            <th>{{ t('application.detail.fields.image') }}</th>
            <th>{{ t('service.fields.container') }}</th>
            <th>{{ t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="component in service.components" :key="component.id">
            <td>
              <router-link
                :to="`/service/${service.id}/component/${component.id}`"
                class="app-link"
              >
                {{ component.component_name }}
              </router-link>
            </td>
            <td class="w-[480px] max-w-[480px] truncate text-muted-foreground" :title="component.image">
              {{ component.image }}
            </td>
            <td class="max-w-sm break-all text-muted-foreground">
              {{ component.container_name }}
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
  import { useI18n } from 'vue-i18n';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';

  defineProps<{ service: ServiceResp }>();

  const emit = defineEmits<{
    'view-logs': [component: string];
  }>();

  const { t } = useI18n();
</script>
