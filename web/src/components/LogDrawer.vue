<template>
  <AppDrawer
    :open="open"
    :title="title"
    width-class="w-[min(960px,100vw)]"
    body-class="min-h-0 flex-1 overflow-hidden p-0"
    @update:open="emit('update:open', $event)"
  >
    <template #actions>
      <LogActions :state="state" @find="view?.find()" @follow="view?.follow()" />
    </template>
    <LogView ref="view" :state="state" />
  </AppDrawer>
</template>

<script setup lang="ts">
  import AppDrawer from '@/components/AppDrawer.vue';
  import { ref } from 'vue';
  import LogActions from '@/components/LogActions.vue';
  import LogView from '@/components/LogView.vue';
  import type { LogState } from '@/composables/useLogStream';
  defineProps<{ open: boolean; title: string; state?: LogState }>();
  const emit = defineEmits<{ 'update:open': [open: boolean] }>();
  const view = ref<InstanceType<typeof LogView>>();
</script>
