<template>
  <div class="flex h-full min-h-0 flex-col">
    <section class="dialogue-workspace flex min-h-0 flex-1 overflow-hidden">
      <aside
        v-if="!historyCollapsed"
        class="hidden min-h-0 w-64 shrink-0 border-r border-border lg:flex"
      >
        <DeploymentDialogueHistory
          :conversations="conversations"
          :active-conversation-id="activeConversationId"
          :loading="historyLoading"
          @new="startNewConversation"
          @select="selectConversation"
          @delete="requestDeleteConversation"
        />
      </aside>

      <div class="flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="flex h-12 shrink-0 items-center gap-2 border-b border-border px-3">
          <button
            type="button"
            class="app-icon-button shrink-0 lg:hidden"
            :aria-label="t('deploymentDialogue.history')"
            :title="t('deploymentDialogue.history')"
            @click="mobileHistoryOpen = true"
          >
            <PanelLeft class="size-4" />
          </button>
          <button
            type="button"
            class="app-icon-button hidden shrink-0 lg:inline-flex"
            :aria-label="t('deploymentDialogue.toggleHistory')"
            :title="t('deploymentDialogue.toggleHistory')"
            @click="historyCollapsed = !historyCollapsed"
          >
            <PanelLeftClose class="size-4" />
          </button>
          <AppTruncatedText
            :text="activeConversationTitle || t('deploymentDialogue.newConversation')"
            class="min-w-0 flex-1 text-sm text-foreground"
          />
          <ToolbarRoot
            v-if="activeConversation"
            class="ml-auto flex shrink-0 items-center"
            :aria-label="t('deploymentDialogue.conversationActions')"
          >
            <button
              type="button"
              class="app-icon-button text-destructive hover:bg-destructive/10 hover:text-destructive"
              :aria-label="t('common.delete')"
              :title="t('common.delete')"
              :disabled="pending || deletingConversation"
              @click="requestDeleteConversation(activeConversation)"
            >
              <Trash2 class="size-4" />
            </button>
          </ToolbarRoot>
        </div>

        <div ref="scrollContainer" class="min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
          <div class="mx-auto flex min-h-full w-full max-w-5xl flex-col gap-5">
            <div v-if="messages.length === 0" class="flex flex-1 items-center justify-center">
              <p class="text-sm text-muted-foreground">{{ t('deploymentDialogue.empty') }}</p>
            </div>

            <article
              v-for="(item, index) in messages"
              :key="`${item.role}-${index}`"
              class="flex"
              :class="item.role === 'user' ? 'justify-end' : 'justify-start'"
            >
              <div
                class="min-w-0 max-w-[88%] break-words rounded-lg px-4 py-3 text-sm leading-6 sm:max-w-[76%]"
                :class="
                  item.role === 'user'
                    ? 'border border-border bg-accent text-foreground'
                    : messageClass(item)
                "
              >
                <p v-if="item.role === 'user'" class="whitespace-pre-wrap">{{ item.content }}</p>
                <MarkdownContent v-else :content="item.content" />
                <details
                  v-for="(call, callIndex) in item.tool_calls || []"
                  :key="callIndex"
                  class="mt-3 border-t border-border/70 pt-2 text-sm"
                >
                  <summary class="cursor-pointer text-xs text-muted-foreground">
                    {{ call.name }}
                  </summary>
                  <pre
                    class="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all rounded-md border border-border bg-background p-2 text-xs text-foreground"
                    >{{ call.result_json }}</pre>
                </details>
              </div>
            </article>

            <div v-if="pending" class="flex justify-start">
              <div
                class="min-w-0 rounded-lg border border-border bg-muted/50 px-4 py-3 text-sm text-muted-foreground"
              >
                <span class="processing-status" role="status">
                  {{ t('deploymentDialogue.thinking') }}
                  <span aria-hidden="true">{{ '.'.repeat(thinkingDotCount) }}</span>
                </span>
                <ul
                  v-if="toolActivity.length > 0"
                  class="mt-2 space-y-1 border-t border-border/70 pt-2 text-sm"
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
        </div>

        <form class="shrink-0 border-t border-border p-4 sm:px-6 sm:py-5" @submit.prevent="send">
          <div class="mx-auto flex w-full max-w-5xl items-end gap-3">
            <AppTextarea
              v-model="draft"
              class="min-h-16 flex-1 resize-none data-[disabled]:opacity-60"
              :placeholder="t('deploymentDialogue.placeholder')"
              :aria-label="t('deploymentDialogue.placeholder')"
              :disabled="pending || historyLoading"
              rows="2"
              @keydown.enter.exact.prevent="send"
            />
            <button
              class="app-button-primary size-10 shrink-0 p-0"
              type="submit"
              :aria-label="t('deploymentDialogue.send')"
              :title="t('deploymentDialogue.send')"
              :disabled="pending || historyLoading || !draft.trim()"
            >
              <Send :size="17" aria-hidden="true" />
            </button>
          </div>
        </form>
      </div>
    </section>

    <AppDrawer
      v-model:open="mobileHistoryOpen"
      :title="t('deploymentDialogue.history')"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
    >
      <template #actions>
        <button
          type="button"
          class="app-icon-button"
          :aria-label="t('deploymentDialogue.newConversation')"
          :title="t('deploymentDialogue.newConversation')"
          @click="startNewConversation"
        >
          <Plus class="size-4" />
        </button>
      </template>
      <DeploymentDialogueHistory
        :conversations="conversations"
        :active-conversation-id="activeConversationId"
        :loading="historyLoading"
        :show-header="false"
        @new="startNewConversation"
        @select="selectConversation"
        @delete="requestDeleteConversation"
      />
    </AppDrawer>

    <AppDialog
      v-model:open="deleteDialogOpen"
      :title="t('deploymentDialogue.deleteTitle')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('deploymentDialogue.deleteDescription') }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="deletingConversation"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="deleteDialogOpen = false"
          @confirm="deleteConversation"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import AppTextarea from '@/components/ui/AppTextarea.vue';
  import { storeToRefs } from 'pinia';
  import { computed, nextTick, onUnmounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { PanelLeft, PanelLeftClose, Plus, Send, Trash2 } from '@lucide/vue';
  import { ToolbarRoot } from 'reka-ui';
  import { dialogueApi } from '@/api/dialogue/dialogue';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppDrawer from '@/components/ui/AppDrawer.vue';
  import AppTruncatedText from '@/components/ui/AppTruncatedText.vue';
  import MarkdownContent from '@/components/ui/MarkdownContent.vue';
  import { useToast } from '@/composables/useToast';
  import type {
    DeploymentDialogueConversation,
    DeploymentDialogueStreamEvent,
    DeploymentDialogueToolCall,
  } from '@/gen/proto/orbit/v1/dialogue/dialogue';
  import DeploymentDialogueHistory from '@/views/deployment/components/DeploymentDialogueHistory.vue';
  import { newDeploymentDialogueConversationId } from '@/views/deployment/deploymentDialogueConversationId';
  import {
    type DeploymentDialogueConversationMessage,
    useDeploymentDialogueStore,
  } from '@/stores/deploymentDialogue';
  import { useProjectStore } from '@/stores/project';
  import { ApiError } from '@/utils/request';

  type ToolActivity = {
    name: string;
    status: 'running' | 'completed' | 'failed';
  };

  const { t } = useI18n();
  const toast = useToast();
  const dialogueStore = useDeploymentDialogueStore();
  const projectStore = useProjectStore();
  const { activeConversationId, conversations, messages } = storeToRefs(dialogueStore);
  const draft = ref('');
  const pending = ref(false);
  const historyLoading = ref(false);
  const historyCollapsed = ref(false);
  const mobileHistoryOpen = ref(false);
  const deleteDialogOpen = ref(false);
  const deletingConversation = ref(false);
  const conversationToDelete = ref<DeploymentDialogueConversation>();
  const thinkingDotCount = ref(1);
  const scrollContainer = ref<HTMLElement>();
  const toolActivity = ref<ToolActivity[]>([]);
  const completedToolCalls = ref<DeploymentDialogueToolCall[]>([]);
  let thinkingDotsTimer: ReturnType<typeof setInterval> | undefined;
  let historyRequest = 0;
  let historySelectionRequest = 0;
  let historyReloadRequest = 0;

  const activeConversation = computed(() =>
    conversations.value.find((conversation) => conversation.id === activeConversationId.value)
  );
  const activeConversationTitle = computed(() => activeConversation.value?.title);

  watch(
    () => projectStore.activeProjectId,
    (projectId) => {
      void loadConversations(projectId);
    },
    { immediate: true }
  );

  async function loadConversations(projectId = projectStore.activeProjectId) {
    const requestId = ++historyRequest;
    ++historySelectionRequest;
    dialogueStore.resetConversation();
    conversations.value = [];
    if (!projectId) {
      historyLoading.value = false;
      return;
    }
    historyLoading.value = true;
    try {
      const response = await dialogueApi.listConversations(projectId);
      if (requestId !== historyRequest) {
        return;
      }
      conversations.value = response.items;
      if (response.items[0]) {
        await selectConversation(response.items[0].id, requestId);
      }
    } catch (error: unknown) {
      if (requestId === historyRequest) {
        toast.error(dialogueErrorMessage(error));
      }
    } finally {
      if (requestId === historyRequest) {
        historyLoading.value = false;
      }
    }
  }

  async function selectConversation(conversationId: string, loadRequestId?: number) {
    if (pending.value || conversationId === activeConversationId.value) {
      mobileHistoryOpen.value = false;
      return;
    }
    const requestId = ++historySelectionRequest;
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    historyLoading.value = true;
    try {
      const detail = await dialogueApi.getConversation(projectId, conversationId);
      if (
        requestId !== historySelectionRequest ||
        projectId !== projectStore.activeProjectId ||
        (loadRequestId !== undefined && loadRequestId !== historyRequest)
      ) {
        return;
      }
      activeConversationId.value = detail.conversation?.id;
      messages.value = detail.messages.map((message) => ({
        role: message.role,
        content: message.content,
      }));
      mobileHistoryOpen.value = false;
      await scrollToLatest();
    } catch (error: unknown) {
      toast.error(dialogueErrorMessage(error));
    } finally {
      historyLoading.value = false;
    }
  }

  function startNewConversation() {
    if (pending.value) {
      return;
    }
    ++historySelectionRequest;
    dialogueStore.resetConversation();
    mobileHistoryOpen.value = false;
  }

  function requestDeleteConversation(conversation: DeploymentDialogueConversation) {
    conversationToDelete.value = conversation;
    deleteDialogOpen.value = true;
    mobileHistoryOpen.value = false;
  }

  async function deleteConversation() {
    const conversation = conversationToDelete.value;
    const projectId = projectStore.activeProjectId;
    if (!conversation || !projectId || deletingConversation.value) {
      return;
    }
    deletingConversation.value = true;
    ++historySelectionRequest;
    try {
      await dialogueApi.deleteConversation(projectId, conversation.id);
      conversations.value = conversations.value.filter((item) => item.id !== conversation.id);
      deleteDialogOpen.value = false;
      conversationToDelete.value = undefined;
      if (activeConversationId.value === conversation.id) {
        const nextConversation = conversations.value[0];
        dialogueStore.resetConversation();
        if (nextConversation) {
          await selectConversation(nextConversation.id);
        }
      }
      toast.success(t('deploymentDialogue.deleteSuccess'));
    } catch (error: unknown) {
      toast.error(dialogueErrorMessage(error));
    } finally {
      deletingConversation.value = false;
    }
  }

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

    const conversationId = activeConversationId.value || newConversationId();
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
        projectId,
        {
          conversation_id: conversationId,
          messages: messages.value
            .filter((message) => !message.is_error)
            .map(({ role, content: messageContent }) => ({ role, content: messageContent })),
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
        throw contractMismatchError();
      }
      activeConversationId.value = conversationId;
      await refreshConversationList(projectId);
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

  async function refreshConversationList(projectId: string) {
    const requestId = ++historyReloadRequest;
    try {
      const response = await dialogueApi.listConversations(projectId);
      if (requestId === historyReloadRequest && projectId === projectStore.activeProjectId) {
        conversations.value = response.items;
      }
    } catch (error: unknown) {
      toast.error(dialogueErrorMessage(error));
    }
  }

  function newConversationId() {
    return newDeploymentDialogueConversationId();
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

  function contractMismatchError() {
    return new ApiError(
      '服务响应格式异常',
      undefined,
      'contract_mismatch',
      undefined,
      'contract_mismatch'
    );
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
