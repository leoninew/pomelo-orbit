<template>
  <div
    class="flex flex-col items-center justify-center gap-4 px-6 text-center text-muted-foreground"
    :class="paddingClass"
  >
    <div class="flex size-16 items-center justify-center rounded-full bg-muted">
      <Inbox :class="iconClass" :stroke-width="1.5" aria-hidden="true" />
    </div>
    <div class="max-w-md text-sm leading-6">
      <slot>{{ displayMessage }}</slot>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { Inbox } from '@lucide/vue';

  const { t } = useI18n();

  const props = withDefaults(
    defineProps<{
      message?: string;
      size?: 'page' | 'section' | 'compact';
    }>(),
    {
      size: 'page',
    }
  );

  const displayMessage = computed(() => props.message || t('common.noData'));
  const paddingClass = computed(() => {
    if (props.size === 'compact') {
      return 'py-8';
    }
    if (props.size === 'section') {
      return 'py-12';
    }
    return 'py-16';
  });
  const iconClass = computed(() => (props.size === 'compact' ? 'size-6' : 'size-8'));
</script>
