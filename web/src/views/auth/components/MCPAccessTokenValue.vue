<template>
  <div v-if="value" class="flex w-fit max-w-xl items-start gap-2">
    <SensitiveValue
      :value="value"
      :mask="`${value.slice(0, 4)}********${value.slice(-4)}`"
      :label="t('mcpAccessTokens.token')"
      :show-label="t('mcpAccessTokens.showToken')"
      :hide-label="t('mcpAccessTokens.hideToken')"
    />
    <button
      type="button"
      class="shrink-0 p-0.5 text-muted-foreground transition-colors hover:text-foreground"
      :title="t('mcpAccessTokens.copy')"
      :aria-label="t('mcpAccessTokens.copy')"
      @click="copyValue(value)"
    >
      <Copy class="size-4" aria-hidden="true" />
    </button>
  </div>
  <span v-else class="text-muted-foreground">-</span>
</template>

<script setup lang="ts">
  import { Copy } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';
  import SensitiveValue from '@/components/ui/SensitiveValue.vue';
  import { useToast } from '@/composables/useToast';

  defineProps<{ value?: string }>();

  const { t } = useI18n();
  const toast = useToast();

  async function copyValue(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(t('mcpAccessTokens.copied'));
    } catch {
      toast.error(t('mcpAccessTokens.copyFailed'));
    }
  }
</script>
