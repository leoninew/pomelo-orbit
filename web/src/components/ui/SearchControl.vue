<template>
  <AppInput
    v-model="value"
    class="app-search-control"
    type="text"
    :placeholder="placeholder || t('common.search')"
    :aria-label="placeholder || t('common.search')"
    :clear-label="t('common.clearSearch')"
    :disabled="disabled || loading"
    @clear="emit('search')"
    @keydown.enter="emit('search')"
  >
    <template #leading>
      <Search class="pointer-events-none mx-3 size-4 shrink-0 text-muted-foreground" />
    </template>
    <template v-if="showButton" #trailing>
      <button
        type="button"
        class="app-search-button"
        :disabled="disabled || loading"
        @click="emit('search')"
      >
        {{ t('common.search') }}
      </button>
    </template>
  </AppInput>
</template>

<script setup lang="ts">
  import { Search } from '@lucide/vue';
  import AppInput from '@/components/ui/AppInput.vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  const { t } = useI18n();

  const props = withDefaults(
    defineProps<{
      modelValue: string;
      placeholder?: string;
      disabled?: boolean;
      loading?: boolean;
      showButton?: boolean;
    }>(),
    {
      disabled: false,
      loading: false,
      showButton: true,
    }
  );

  const emit = defineEmits<{
    'update:modelValue': [value: string];
    search: [];
  }>();

  const value = computed({
    get: () => props.modelValue,
    set: (next: string) => emit('update:modelValue', next),
  });
</script>
