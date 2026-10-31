<template>
  <footer class="flex max-w-full items-center gap-2 px-1 text-sm text-muted-foreground">
    <time v-if="createdAt" class="text-xs" :datetime="createdAt">{{ formatTime(createdAt) }}</time>
    <button
      type="button"
      class="app-icon-button size-7 shrink-0"
      :aria-label="t('deploymentDialogue.copyMessage')"
      :title="t('deploymentDialogue.copyMessage')"
      @click="copyMessage"
    >
      <Copy class="size-4" aria-hidden="true" />
    </button>
  </footer>
</template>

<script setup lang="ts">
  import { Copy } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';
  import { useToast } from '@/composables/useToast';
  import { formatTime } from '@/utils/time';

  const props = defineProps<{ content: string; createdAt?: string }>();
  const { t } = useI18n();
  const toast = useToast();

  async function copyMessage() {
    try {
      await navigator.clipboard.writeText(props.content);
      toast.success(t('deploymentDialogue.messageCopied'));
    } catch {
      toast.error(t('deploymentDialogue.copyFailed'));
    }
  }
</script>
