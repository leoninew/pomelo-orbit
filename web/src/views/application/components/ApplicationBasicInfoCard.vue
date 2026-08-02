<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ t('application.detail.sections.basicInfo') }}</h2>
      <button class="app-button-primary h-9 px-3" :disabled="disabled" @click="emit('edit')">
        <Pencil class="size-4" />
        {{ t('common.edit') }}
      </button>
    </div>
    <dl class="app-detail-info-grid">
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('common.name') }}</dt>
        <dd class="text-foreground">{{ application.name }}</dd>
      </div>
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('application.code') }}</dt>
        <dd class="text-foreground">{{ application.code }}</dd>
      </div>
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('application.kind') }}</dt>
        <dd>
          <AppBadge variant="pill" :tone="applicationKindTone(application.kind)">
            {{ application.kind }}
          </AppBadge>
        </dd>
      </div>
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('common.createdAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(application.created_at) }}</dd>
      </div>
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('common.updatedAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(application.updated_at) }}</dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
  import { Pencil } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import { applicationKindTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  defineProps<{
    application: ApplicationResp;
    disabled: boolean;
  }>();

  const emit = defineEmits<{ edit: [] }>();
  const { t } = useI18n();
</script>
