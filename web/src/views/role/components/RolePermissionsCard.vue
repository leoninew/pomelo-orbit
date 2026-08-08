<template>
  <DetailInfoCard
    :title="t('roleManagement.permissions')"
    :editable="Boolean(role && editable)"
    :disabled="disabled"
    @edit="emit('edit')"
  >
    <div class="px-5 py-4">
      <div v-if="role && role.permission_codes.length > 0" class="flex flex-wrap gap-2">
        <AppBadge v-for="code in role.permission_codes" :key="code">
          {{ permissionName(code) }}
        </AppBadge>
      </div>
      <p v-else class="text-sm text-muted-foreground">{{ t('common.noData') }}</p>
    </div>
  </DetailInfoCard>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
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
