<template>
  <div class="flex h-full min-h-0 flex-col gap-3">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('deploymentDialogue.toolbar')">
      <div class="flex items-center gap-3">
        <h1 class="text-lg font-semibold text-foreground">{{ t('deploymentDialogue.title') }}</h1>
      </div>
      <button
        class="app-icon-button"
        type="button"
        :aria-label="t('deploymentDialogue.clear')"
        :title="t('deploymentDialogue.clear')"
        :disabled="messages.length === 0 || pending"
        @click="clearConversation"
      >
        <Trash2 :size="16" aria-hidden="true" />
      </button>
    </ToolbarRoot>

    <section class="app-surface flex min-h-0 flex-1 flex-col overflow-hidden border-border">
      <div ref="scrollContainer" class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5 sm:p-6">
        <div v-if="messages.length === 0" class="flex h-full min-h-80 items-center justify-center">
          <p class="text-sm text-muted-foreground">{{ t('deploymentDialogue.empty') }}</p>
        </div>

        <article
          v-for="(item, index) in messages"
          :key="`${item.role}-${index}`"
          class="flex"
          :class="item.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            class="max-w-[88%] break-words rounded-xl px-4 py-3 text-sm leading-6 sm:max-w-[76%]"
            :class="
              item.role === 'user' ? 'bg-primary text-primary-foreground' : messageClass(item)
            "
          >
            <p v-if="item.role === 'user'" class="whitespace-pre-wrap">{{ item.content }}</p>
            <MarkdownContent v-else :content="item.content" />
            <details
              v-for="(call, callIndex) in item.tool_calls || []"
              :key="callIndex"
              class="mt-3 border-t border-border/70 pt-2 text-xs"
            >
              <summary class="cursor-pointer text-muted-foreground">{{ call.name }}</summary>
              <pre
                class="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all rounded-xl bg-background p-2 text-foreground"
                >{{ call.result_json }}</pre>
            </details>
          </div>
        </article>

        <div v-if="pending" class="flex justify-start">
          <div
            class="rounded-xl border border-border bg-muted/50 px-4 py-3 text-sm text-muted-foreground"
          >
            <span class="processing-status" role="status">
              {{ t('deploymentDialogue.thinking') }}
              <span aria-hidden="true">{{ '.'.repeat(thinkingDotCount) }}</span>
            </span>
            <ul
              v-if="toolActivity.length > 0"
              class="mt-2 space-y-1 border-t border-border/70 pt-2 text-xs"
            >
              <li
                v-for="(activity, index) in toolActivity"
                :key="`${activity.name}-${index}`"
                :class="activity.status === 'failed' ? 'text-destructive' : ''"
              >
                {{ toolActivityText(activity) }}
              </li>
            </ul>
          </div>
        </div>
      </div>

      <form class="shrink-0 border-t border-border p-4 sm:p-5" @submit.prevent="send">
        <div class="flex items-end gap-3">
          <textarea
            v-model="draft"
            class="min-h-11 flex-1 resize-none rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-primary disabled:cursor-not-allowed disabled:opacity-60"
            :placeholder="t('deploymentDialogue.placeholder')"
            :aria-label="t('deploymentDialogue.placeholder')"
            :disabled="pending"
            rows="2"
            @keydown.enter.exact.prevent="send"
          />
          <button
            class="app-button-primary size-10 p-0"
            type="submit"
            :aria-label="t('deploymentDialogue.send')"
            :title="t('deploymentDialogue.send')"
            :disabled="pending || !draft.trim()"
          >
            <Send :size="17" aria-hidden="true" />
          </button>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup lang="ts">
  import { storeToRefs } from 'pinia';
  import { nextTick, onUnmounted, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { Send, Trash2 } from 'lucide-vue-next';
  import { ToolbarRoot } from 'reka-ui';
  import { dialogueApi } from '@/api/dialogue/dialogue';
  import MarkdownContent from '@/components/MarkdownContent.vue';
  import { useToast } from '@/composables/useToast';
  import {
    type DeploymentDialogueConversationMessage,
    useDeploymentDialogueStore,
  } from '@/stores/deploymentDialogue';
  import { useProjectStore } from '@/stores/project';
  import type {
    DeploymentDialogueStreamEvent,
    DeploymentDialogueToolCall,
  } from '@/gen/proto/orbit/v1/dialogue/dialogue';
  import { ApiError } from '@/utils/request';

  type ToolActivity = {
    name: string;
    status: 'running' | 'completed' | 'failed';
  };

  const { t } = useI18n();
  const toast = useToast();
  const dialogueStore = useDeploymentDialogueStore();
  const projectStore = useProjectStore();
  const { messages } = storeToRefs(dialogueStore);
  const draft = ref('');
  const pending = ref(false);
  const thinkingDotCount = ref(1);
  const scrollContainer = ref<HTMLElement>();
  const toolActivity = ref<ToolActivity[]>([]);
  const completedToolCalls = ref<DeploymentDialogueToolCall[]>([]);
  let thinkingDotsTimer: ReturnType<typeof setInterval> | undefined;

  async function send() {
    const content = draft.value.trim();
    const projectId = projectStore.activeProjectId;
    if (!content || pending.value) {
      return;
    }
    if (!projectId) {
      toast.error(t('deploymentDialogue.selectProjectRequired'));
      return;
    }

    const userMessage: DeploymentDialogueConversationMessage = { role: 'user', content };
    messages.value.push(userMessage);
    draft.value = '';
    pending.value = true;
    startThinkingDots();
    toolActivity.value = [];
    completedToolCalls.value = [];
    await scrollToLatest();
    try {
      let isComplete = false;
      await dialogueApi.completeTurnStream(
        {
          project_id: projectId,
          messages: messages.value
            .filter((message) => !message.is_error)
            .map(({ role, content: messageContent }) => ({
              role,
              content: messageContent,
            })),
        },
        (event) => {
          if (event.type === 'tool_call_started') {
            addToolActivity(event.tool_call);
          } else if (event.type === 'tool_call_completed') {
            completeToolActivity(event.tool_call);
          } else if (event.type === 'complete') {
            messages.value.push({
              role: 'assistant',
              content: event.message,
              tool_calls: completedToolCalls.value,
            });
            isComplete = true;
          } else if (event.type === 'error') {
            throw new ApiError(event.message, undefined, event.code, event.request_id);
          }
          void scrollToLatest();
        }
      );
      if (!isComplete) {
        throw new ApiError(
          '服务响应格式异常',
          undefined,
          'contract_mismatch',
          undefined,
          'contract_mismatch'
        );
      }
    } catch (error: unknown) {
      userMessage.is_error = true;
      const message = dialogueErrorMessage(error);
      messages.value.push({
        role: 'assistant',
        content: t('deploymentDialogue.interrupted', { message }),
        tool_calls: completedToolCalls.value,
        is_error: true,
      });
      toast.error(message);
    } finally {
      pending.value = false;
      stopThinkingDots();
      await scrollToLatest();
    }
  }

  function startThinkingDots() {
    stopThinkingDots();
    thinkingDotCount.value = 1;
    thinkingDotsTimer = setInterval(() => {
      thinkingDotCount.value = (thinkingDotCount.value % 6) + 1;
    }, 1000);
  }

  function stopThinkingDots() {
    if (thinkingDotsTimer !== undefined) {
      clearInterval(thinkingDotsTimer);
      thinkingDotsTimer = undefined;
    }
  }

  function clearConversation() {
    dialogueStore.clearMessages();
  }

  function messageClass(message: DeploymentDialogueConversationMessage) {
    return message.is_error
      ? 'border border-destructive/50 bg-destructive/10 text-destructive'
      : 'border border-border bg-muted/50 text-foreground';
  }

  function addToolActivity(toolCall: DeploymentDialogueStreamEvent['tool_call']) {
    if (toolCall) {
      toolActivity.value.push({ name: toolCall.name, status: 'running' });
    }
  }

  function completeToolActivity(toolCall: DeploymentDialogueStreamEvent['tool_call']) {
    if (!toolCall) {
      return;
    }
    const activity = [...toolActivity.value]
      .reverse()
      .find((item) => item.name === toolCall.name && item.status === 'running');
    if (activity) {
      activity.status = toolCall.is_error ? 'failed' : 'completed';
    }
    completedToolCalls.value.push(toolCall);
  }

  function toolActivityText(activity: ToolActivity) {
    if (activity.status === 'failed') {
      return t('deploymentDialogue.toolFailed', { name: activity.name });
    }
    if (activity.status === 'completed') {
      return t('deploymentDialogue.toolCompleted', { name: activity.name });
    }
    return t('deploymentDialogue.toolRunning', { name: activity.name });
  }

  function dialogueErrorMessage(error: unknown) {
    if (error instanceof ApiError) {
      if (error.code === 'deployment_dialogue_not_configured') {
        return t('deploymentDialogue.notConfigured');
      }
      if (error.code === 'request_timeout') {
        return t('deploymentDialogue.requestTimeout');
      }
      return error.message;
    }
    return error instanceof Error ? error.message : t('deploymentDialogue.sendFailed');
  }

  async function scrollToLatest() {
    await nextTick();
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight;
    }
  }

  onUnmounted(stopThinkingDots);
</script>

<style scoped>
  .processing-status {
    display: inline-block;
  }
</style>
