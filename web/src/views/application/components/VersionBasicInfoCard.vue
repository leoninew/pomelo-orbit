<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ t('application.detail.sections.basicInfo') }}</h2>
      <button
        v-if="editable"
        class="app-button-primary h-9 px-3"
        :disabled="disabled"
        @click="emit('edit')"
      >
        <Pencil class="size-4" />
        {{ t('common.edit') }}
      </button>
    </div>
    <dl class="app-detail-info-grid">
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('application.detail.fields.versionLabel') }}</dt>
        <dd class="text-foreground">{{ version.label }}</dd>
      </div>
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('common.status') }}</dt>
        <dd>
          <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
            {{ version.status }}
          </AppBadge>
        </dd>
      </div>
      <div v-if="version.note" class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('application.detail.fields.note') }}</dt>
        <dd class="text-foreground">{{ version.note }}</dd>
      </div>
      <div class="flex gap-2">
        <dt class="whitespace-nowrap">{{ t('common.createdAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(version.created_at) }}</dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
  import { Pencil } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { versionStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  defineProps<{
    version: VersionResp;
    editable: boolean;
    disabled: boolean;
  }>();

  const emit = defineEmits<{ edit: [] }>();
  const { t } = useI18n();
</script>
