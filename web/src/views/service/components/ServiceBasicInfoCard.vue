<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ t('service.detail.sections.basic') }}</h2>
    </div>
    <dl class="app-detail-info-grid">
      <div class="flex gap-2">
        <dt>{{ t('service.fields.application') }}</dt>
        <dd>
          <router-link :to="`/application/${service.application_id}`" class="app-link">
            {{ service.application_name || service.application_id }}
          </router-link>
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('service.fields.instanceKey') }}</dt>
        <dd class="text-foreground">{{ service.instance_key }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('service.fields.version') }}</dt>
        <dd>
          <router-link :to="`/version/${service.version_id}`" class="app-link">
            {{ service.version_label || service.version_id }}
          </router-link>
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.status') }}</dt>
        <dd>
          <AppBadge variant="status" :tone="appStatusTone(service.status)">
            {{ service.status }}
          </AppBadge>
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.createdAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(service.created_at) }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.updatedAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(service.updated_at) }}</dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { appStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  defineProps<{ service: ServiceResp }>();

  const { t } = useI18n();
</script>
