<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ t('userManagement.basicInfo') }}</h2>
      <button
        v-if="user && editable"
        class="app-button-primary h-9 px-3"
        :disabled="disabled"
        @click="emit('edit')"
      >
        <Pencil class="size-4" />
        {{ t('common.edit') }}
      </button>
    </div>

    <AppSpinner v-if="loading" class="px-5 py-10" />
    <dl v-else-if="user" class="app-detail-info-grid">
      <div class="flex gap-2">
        <dt>{{ t('userManagement.username') }}</dt>
        <dd class="text-foreground">{{ user.username }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('userManagement.email') }}</dt>
        <dd class="text-foreground">{{ user.email || '-' }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.status') }}</dt>
        <dd>
          <AppBadge variant="status" :tone="user.status === 'enabled' ? 'success' : 'default'">
            {{ user.status }}
          </AppBadge>
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('userManagement.authSource') }}</dt>
        <dd>
          <AppBadge variant="pill">{{ user.auth_source }}</AppBadge>
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.createdAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(user.created_at) }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('common.updatedAt') }}</dt>
        <dd class="text-muted-foreground">{{ formatTime(user.updated_at) }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('userManagement.lastLoginAt') }}</dt>
        <dd class="text-muted-foreground">
          {{ user.last_login_at ? formatTime(user.last_login_at) : '-' }}
        </dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
  import { Pencil } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import type { UserResp } from '@/gen/proto/orbit/v1/user/user';
  import { formatTime } from '@/utils/time';

  defineProps<{
    user?: UserResp;
    loading: boolean;
    editable: boolean;
    disabled: boolean;
  }>();

  const emit = defineEmits<{ edit: [] }>();
  const { t } = useI18n();
</script>
