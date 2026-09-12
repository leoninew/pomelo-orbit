<template>
  <AppDialog
    v-model:open="openModel"
    :title="t('project.initialization.windowsTargetCommandTitle')"
    width-class="w-[min(820px,calc(100vw-32px))]"
  >
    <div class="space-y-4">
      <p class="text-sm text-muted-foreground">
        {{ t('project.initialization.windowsTargetCommandDescription') }}
      </p>
      <div class="space-y-2">
        <div class="flex items-center justify-between gap-2">
          <p class="text-xs font-medium text-foreground">
            {{ t('project.initialization.windowsTargetCommandLabel') }}
          </p>
          <button
            type="button"
            class="app-button h-8 px-3 text-xs"
            :disabled="loading || !command"
            @click="copyCommand"
          >
            <Copy class="size-3.5" aria-hidden="true" />
            {{ t('project.initialization.windowsTargetCopyCommand') }}
          </button>
        </div>
        <textarea
          v-if="!loading"
          :value="command"
          rows="18"
          readonly
          class="app-textarea w-full font-mono text-xs leading-5"
        />
        <p v-else class="text-sm text-muted-foreground" aria-live="polite">
          {{ t('project.initialization.windowsTargetCommandLoading') }}
        </p>
        <p v-if="error" class="app-field-error" role="alert">{{ error }}</p>
      </div>
      <p class="text-xs leading-5 text-muted-foreground">
        {{ t('project.initialization.windowsTargetCommandHint') }}
      </p>
    </div>
  </AppDialog>
</template>

<script setup lang="ts">
  import { Copy } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/AppDialog.vue';
  import { useToast } from '@/composables/useToast';

  const props = defineProps<{
    open: boolean;
    command: string;
    loading?: boolean;
    error?: string;
  }>();

  const emit = defineEmits<{
    'update:open': [open: boolean];
  }>();

  const { t } = useI18n();
  const toast = useToast();
  const openModel = computed({
    get: () => props.open,
    set: (value) => emit('update:open', value),
  });

  async function copyCommand() {
    if (!props.command) return;
    try {
      await navigator.clipboard.writeText(props.command);
      toast.success(t('project.initialization.windowsTargetCommandCopied'));
    } catch {
      toast.error(t('project.initialization.windowsTargetCopyFailed'));
    }
  }
</script>
