<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ t('roleManagement.basicInfo') }}</h2>
      <button
        v-if="role && editable"
        class="app-button-primary h-9 px-3"
        :disabled="disabled"
        @click="emit('edit')"
      >
        <Pencil class="size-4" />
        {{ t('common.edit') }}
      </button>
    </div>
    <AppSpinner v-if="loading" class="px-5 py-10" />
    <dl v-else-if="role" class="app-detail-info-grid">
      <div class="flex gap-2">
        <dt>{{ t('roleManagement.code') }}</dt>
        <dd class="text-foreground">{{ role.code }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.name') }}</dt>
        <dd class="text-foreground">{{ role.name }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.description') }}</dt>
        <dd class="text-foreground">{{ role.description || '-' }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.createdAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(role.created_at) }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.updatedAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(role.updated_at) }}</dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
  import { Pencil } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import AppSpinner from '@/components/AppSpinner.vue';
  import type { RoleResp } from '@/gen/proto/orbit/v1/role/role';
  import { formatTime } from '@/utils/time';

  defineProps<{
    role?: RoleResp;
    loading: boolean;
    editable: boolean;
    disabled: boolean;
  }>();

  const emit = defineEmits<{ edit: [] }>();
  const { t } = useI18n();
</script>
