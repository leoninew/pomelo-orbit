<template>
  <DetailInfoCard
    :title="t('userManagement.rolesAndPermissions')"
    :editable="Boolean(user && editable)"
    :disabled="disabled"
    @edit="emit('edit')"
  >
    <div v-if="user && user.role_items.length > 0" class="px-5 py-4">
      <TabsRoot :default-value="user.role_items[0]?.id">
        <TabsList class="flex gap-1 border-b border-border">
          <TabsTrigger
            v-for="role in user.role_items"
            :key="role.id"
            :value="role.id"
            class="app-tab"
          >
            {{ role.name }}
          </TabsTrigger>
        </TabsList>
        <TabsContent v-for="role in user.role_items" :key="role.id" :value="role.id" class="pt-4">
          <div v-if="role.permission_codes.length > 0" class="flex flex-wrap gap-2">
            <AppBadge v-for="permission in role.permission_codes" :key="permission">
              {{ permission }}
            </AppBadge>
          </div>
          <p v-else class="text-sm text-muted-foreground">{{ t('common.noData') }}</p>
        </TabsContent>
      </TabsRoot>
    </div>
    <div v-else class="px-5 py-4">
      <p class="text-sm text-muted-foreground">{{ t('common.noData') }}</p>
    </div>
  </DetailInfoCard>
</template>

<script setup lang="ts">
  import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/ui/AppBadge.vue';
  import DetailInfoCard from '@/components/ui/DetailInfoCard.vue';
  import type { UserResp } from '@/gen/proto/orbit/v1/user/user';

  defineProps<{
    user?: UserResp;
    editable: boolean;
    disabled: boolean;
  }>();

  const emit = defineEmits<{ edit: [] }>();
  const { t } = useI18n();
</script>
