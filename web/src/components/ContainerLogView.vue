<template>
  <div class="flex h-full min-h-0 flex-1 flex-col">
    <div v-if="showAutoRefresh" class="mb-3 flex shrink-0 justify-end">
      <button
        class="app-button inline-flex h-9 items-center gap-2 px-3"
        :class="autoRefreshing ? 'text-primary' : ''"
        @click="emit('toggle-auto-refresh')"
      >
        <Loader2 class="size-4" :class="autoRefreshing ? 'animate-spin' : ''" />
        {{ autoRefreshing ? t('service.logs.autoRefreshing') : t('service.logs.refreshPaused') }}
      </button>
    </div>
    <div v-if="!logs" class="flex min-h-0 flex-1 items-center justify-center text-muted-foreground">
      <div class="text-center">
        <p v-if="message" class="text-sm">{{ message }}</p>
        <template v-else>
          <AppSpinner v-if="status === 'loading' || status === 'streaming'" />
          <p v-if="status === 'loading'" class="mt-2 text-sm">
            {{ t('service.logs.loading') }}
          </p>
          <p v-else-if="status === 'streaming'" class="mt-2 text-sm">
            {{ t('service.logs.streaming') }}
          </p>
          <p v-else-if="status === 'empty' || status === 'done'" class="text-sm">
            {{ t('service.logs.empty') }}
          </p>
          <div v-else-if="status === 'error'">
            <p class="text-sm text-destructive">{{ error }}</p>
            <button type="button" class="app-link mt-2 text-sm" @click="emit('retry')">
              {{ t('service.logs.retry') }}
            </button>
          </div>
        </template>
      </div>
    </div>
    <div v-else class="min-h-0 flex-1">
      <MonacoEditor
        :model-value="logs"
        language="plaintext"
        height="100%"
        :readonly="true"
        @mount="handleEditorMount"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { Loader2 } from '@lucide/vue';
  import { nextTick, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import type { editor } from 'monaco-editor';

  const props = defineProps<{
    logs: string;
    status: 'loading' | 'streaming' | 'done' | 'empty' | 'error';
    error: string;
    message: string;
    autoRefreshing: boolean;
    showAutoRefresh: boolean;
  }>();

  const emit = defineEmits<{
    retry: [];
    'toggle-auto-refresh': [];
  }>();

  const { t } = useI18n({ useScope: 'global' });
  let logEditor: editor.IStandaloneCodeEditor | null = null;

  function revealLastLine() {
    const lineCount = logEditor?.getModel()?.getLineCount() ?? 0;
    if (lineCount > 0) logEditor?.revealLine(lineCount);
  }

  function handleEditorMount(ed: editor.IStandaloneCodeEditor) {
    logEditor = ed;
    revealLastLine();
  }

  watch(
    () => props.logs,
    () => void nextTick(revealLastLine)
  );
</script>
