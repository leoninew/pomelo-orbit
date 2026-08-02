<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ t('application.detail.fields.components') }}</h2>
    </div>
    <AppEmptyState v-if="service.components.length === 0" size="compact" />
    <div v-else class="overflow-x-auto">
      <table class="app-data-table min-w-[840px]">
        <thead>
          <tr>
            <th>{{ t('application.detail.fields.component') }}</th>
            <th>{{ t('application.detail.fields.image') }}</th>
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
            <td class="max-w-md whitespace-normal break-all text-muted-foreground">
              {{ component.image }}
            </td>
            <td>
              <router-link
                class="app-link"
                :to="`/service/${service.id}/component/${component.id}`"
              >
                {{ t('common.edit') }}
              </router-link>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';

  defineProps<{ service: ServiceResp }>();

  const { t } = useI18n();
</script>
