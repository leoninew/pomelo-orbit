<template>
  <div
    class="flex flex-col items-center justify-center text-muted-foreground"
    :class="paddingClass"
  >
    <Inbox :class="iconClass" />
    <p class="mt-2 text-sm">{{ displayMessage }}</p>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { Inbox } from 'lucide-vue-next';

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
  const iconClass = computed(() => (props.size === 'compact' ? 'size-8' : 'size-12'));
</script>
