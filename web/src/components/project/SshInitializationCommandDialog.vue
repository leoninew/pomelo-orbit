<template>
  <AppDialog
    v-model:open="openModel"
    :title="t('project.initialization.sshCommandTitle')"
    width-class="w-[min(820px,calc(100vw-32px))]"
  >
    <AppLoadingState v-if="loading" size="section" aria-live="polite" />
    <div v-else-if="command" class="min-h-0">
      <AppTextarea
        :model-value="command"
        class="app-code-content h-[420px] resize-none font-mono"
        readonly
        spellcheck="false"
        wrap="soft"
      />
    </div>
    <p v-else-if="!error" class="text-sm text-muted-foreground" aria-live="polite">
      {{ t('project.initialization.sshCommandLoading') }}
    </p>
    <p v-if="error" class="app-field-error" role="alert">{{ error }}</p>
    <template #footer>
      <button type="button" class="app-button" :disabled="loading || !command" @click="copyCommand">
        <Copy class="size-4" aria-hidden="true" />
        {{ t('project.initialization.sshCommandCopy') }}
      </button>
      <button type="button" class="app-button-primary" @click="openModel = false">
        {{ t('common.close') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import AppTextarea from '@/components/ui/AppTextarea.vue';
  import { Copy } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppLoadingState from '@/components/ui/AppLoadingState.vue';
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
      toast.success(t('project.initialization.sshCommandCopied'));
    } catch {
      toast.error(t('project.initialization.sshCommandCopyFailed'));
    }
  }
</script>
