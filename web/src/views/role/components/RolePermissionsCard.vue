<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ t('roleManagement.permissions') }}</h2>
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
    <div class="px-5 py-4">
      <div v-if="role && role.permission_codes.length > 0" class="flex flex-wrap gap-2">
        <AppBadge v-for="code in role.permission_codes" :key="code">
          {{ permissionName(code) }}
        </AppBadge>
      </div>
      <p v-else class="text-sm text-muted-foreground">{{ t('common.noData') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
  import { Pencil } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import type { RoleResp } from '@/gen/proto/orbit/v1/role/role';

  defineProps<{
    role?: RoleResp;
    editable: boolean;
    disabled: boolean;
    permissionName: (code: string) => string;
  }>();

  const emit = defineEmits<{ edit: [] }>();
  const { t } = useI18n();
</script>
