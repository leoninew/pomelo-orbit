<template>
  <AppTruncatedText
    v-if="route.service_id && route.service_code"
    :text="route.service_code"
    as-child
  >
    <router-link :to="`/service/${route.service_id}`" class="app-link">
      {{ route.service_code }}
    </router-link>
  </AppTruncatedText>
  <span v-else class="text-muted-foreground">
    {{ route.service_id ? t('route.targetServiceDeleted') : '—' }}
  </span>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import AppTruncatedText from '@/components/ui/AppTruncatedText.vue';
  import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';

  defineProps<{ route: Pick<RouteResp, 'service_id' | 'service_code'> }>();
  const { t } = useI18n();
</script>
