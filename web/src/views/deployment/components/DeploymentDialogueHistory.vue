<template>
  <nav
    class="flex h-full min-h-0 min-w-0 flex-1 flex-col"
    :aria-label="t('deploymentDialogue.history')"
  >
    <div
      v-if="showHeader"
      class="flex h-12 shrink-0 items-center justify-between border-b border-border px-3"
    >
      <span class="text-sm text-foreground">{{ t('deploymentDialogue.history') }}</span>
      <button
        type="button"
        class="app-icon-button"
        :aria-label="t('deploymentDialogue.newConversation')"
        :title="t('deploymentDialogue.newConversation')"
        @click="emit('new')"
      >
        <Plus class="size-4" />
      </button>
    </div>

    <div v-if="loading" class="flex flex-1 items-center justify-center">
      <AppSpinner class="size-5" />
    </div>
    <p
      v-else-if="conversations.length === 0"
      class="px-4 py-5 text-sm leading-6 text-muted-foreground"
    >
      {{ t('deploymentDialogue.historyEmpty') }}
    </p>
    <ol v-else class="min-h-0 flex-1 space-y-2 overflow-y-auto p-3">
      <li
        v-for="conversation in conversations"
        :key="conversation.id"
        class="dialogue-history-item relative flex min-w-0 items-center rounded-md border transition-colors"
        :data-active="conversation.id === activeConversationId ? '' : undefined"
        :class="
          conversation.id === activeConversationId
            ? 'border-border text-foreground'
            : 'border-transparent text-foreground hover:bg-muted'
        "
      >
        <button
          type="button"
          class="min-w-0 flex-1 rounded-md px-3 py-2.5 text-left text-sm outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
          :aria-current="conversation.id === activeConversationId ? 'true' : undefined"
          @click="emit('select', conversation.id)"
        >
          <AppTruncatedText :text="conversation.title" as-child>
            <span>{{ conversation.title }}</span>
          </AppTruncatedText>
          <span class="mt-0.5 block text-xs text-muted-foreground">
            {{ formatRelativeTime(conversation.updated_at) }}
          </span>
        </button>
        <AppActionMenu
          v-if="conversation.id === activeConversationId"
          :options="conversationActions"
          @select="handleConversationAction($event, conversation)"
        >
          <template #trigger>
            <button
              type="button"
              class="app-icon-button mr-1 shrink-0"
              :aria-label="t('deploymentDialogue.conversationActions')"
              :title="t('deploymentDialogue.conversationActions')"
            >
              <Ellipsis class="size-4" />
            </button>
          </template>
        </AppActionMenu>
      </li>
    </ol>
  </nav>
</template>

<script setup lang="ts">
  import { Ellipsis, Plus } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppSpinner from '@/components/ui/AppSpinner.vue';
  import AppTruncatedText from '@/components/ui/AppTruncatedText.vue';
  import AppActionMenu from '@/components/ui/AppActionMenu.vue';
  import type { DeploymentDialogueConversation } from '@/gen/proto/orbit/v1/dialogue/dialogue';
  import { formatRelativeTime } from '@/utils/time';

  withDefaults(
    defineProps<{
      conversations: DeploymentDialogueConversation[];
      activeConversationId?: string;
      loading: boolean;
      showHeader?: boolean;
    }>(),
    { showHeader: true }
  );

  const emit = defineEmits<{
    new: [];
    select: [conversationId: string];
    delete: [conversation: DeploymentDialogueConversation];
  }>();

  const { t } = useI18n();
  const conversationActions = computed(() => [
    { value: 'delete', label: t('common.delete'), destructive: true },
  ]);

  function handleConversationAction(action: string, conversation: DeploymentDialogueConversation) {
    if (action === 'delete') {
      emit('delete', conversation);
    }
  }
</script>
