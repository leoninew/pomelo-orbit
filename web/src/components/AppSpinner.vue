<template>
  <div
    class="flex justify-center"
    role="status"
    aria-live="polite"
    :aria-label="resolvedLabel"
    v-bind="$attrs"
  >
    <div
      class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent"
      aria-hidden="true"
    />
    <span class="sr-only">{{ resolvedLabel }}</span>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  defineOptions({ inheritAttrs: false });

  const props = withDefaults(
    defineProps<{
      label?: string;
    }>(),
    {
      label: '',
    }
  );

  const { t } = useI18n({ useScope: 'global' });
  const resolvedLabel = computed(() => props.label || t('common.loading'));
</script>
