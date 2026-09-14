<template>
  <AppDialog
    v-model:open="openModel"
    :title="t('project.initialization.windowsTargetCommandTitle')"
    width-class="w-[min(820px,calc(100vw-32px))]"
    body-class="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4"
    content-class="max-h-[calc(100vh-32px)] flex flex-col"
  >
    <AppLoadingState v-if="loading" size="section" aria-live="polite" />
    <div v-else-if="command" class="min-h-0">
      <textarea
        :value="command"
        class="app-textarea h-[420px] resize-none font-mono text-xs leading-5"
        readonly
        spellcheck="false"
        wrap="soft"
      />
    </div>
    <p v-else class="text-sm text-muted-foreground" aria-live="polite">
      {{ t('project.initialization.windowsTargetCommandLoading') }}
    </p>
    <p v-if="error" class="app-field-error" role="alert">{{ error }}</p>
    <template #footer>
      <button type="button" class="app-button" :disabled="loading || !command" @click="copyCommand">
        <Copy class="size-4" aria-hidden="true" />
        {{ t('project.initialization.windowsTargetCopyCommand') }}
      </button>
      <button type="button" class="app-button-primary" @click="openModel = false">
        {{ t('common.close') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { Copy } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/AppDialog.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
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
    if (!props.command) {
      return;
    }
    try {
      await navigator.clipboard.writeText(props.command);
      toast.success(t('project.initialization.windowsTargetCommandCopied'));
    } catch {
      toast.error(t('project.initialization.windowsTargetCopyFailed'));
    }
  }
</script>
