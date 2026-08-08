<template>
  <div class="flex items-center gap-2">
    <button
      type="button"
      class="app-button"
      :disabled="busy || cancelDisabled"
      @click="emit('cancel')"
    >
      {{ resolvedCancelLabel }}
    </button>
    <button
      type="button"
      :class="confirmClass"
      :disabled="busy || confirmDisabled"
      @click="handleConfirm"
    >
      <LoaderCircle v-if="busy" class="size-4 animate-spin" aria-hidden="true" />
      {{ resolvedConfirmLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
  import { LoaderCircle } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  const props = withDefaults(
    defineProps<{
      busy?: boolean;
      cancelDisabled?: boolean;
      cancelLabel?: string;
      confirmDisabled?: boolean;
      confirmLabel?: string;
      variant?: 'primary' | 'destructive';
    }>(),
    {
      busy: false,
      cancelDisabled: false,
      cancelLabel: '',
      confirmDisabled: false,
      confirmLabel: '',
      variant: 'primary',
    }
  );

  const emit = defineEmits<{
    cancel: [];
    confirm: [];
  }>();

  const { t } = useI18n({ useScope: 'global' });
  const resolvedCancelLabel = computed(() => props.cancelLabel || t('common.cancel'));
  const resolvedConfirmLabel = computed(() => props.confirmLabel || t('common.confirm'));
  const confirmClass = computed(() =>
    props.variant === 'destructive' ? 'app-button-destructive' : 'app-button-primary'
  );

  function handleConfirm() {
    emit('confirm');
  }
</script>
