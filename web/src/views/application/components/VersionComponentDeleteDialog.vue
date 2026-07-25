<template>
  <AppDialog
    v-model:open="isOpen"
    :title="t('application.versionDetail.dialog.deleteComponent')"
    width-class="w-[min(420px,calc(100vw-32px))]"
  >
    <p class="text-sm text-muted-foreground">
      {{ t('application.versionDetail.deleteComponentDescription', { name: componentName }) }}
    </p>
    <template #footer>
      <button class="app-button" @click="isOpen = false">{{ t('common.cancel') }}</button>
      <button :disabled="saving" class="app-button-danger" @click="emit('confirm')">
        {{ t('common.delete') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/AppDialog.vue';

  const props = withDefaults(
    defineProps<{
      open: boolean;
      componentName?: string;
      saving?: boolean;
    }>(),
    { componentName: '', saving: false }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
    confirm: [];
  }>();

  const { t } = useI18n();
  const isOpen = computed({
    get: () => props.open,
    set: (open) => emit('update:open', open),
  });
</script>
