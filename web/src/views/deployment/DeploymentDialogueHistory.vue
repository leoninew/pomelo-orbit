<template>
  <nav class="flex min-h-0 min-w-0 flex-1 flex-col" :aria-label="t('deploymentDialogue.history')">
    <div class="flex h-12 shrink-0 items-center justify-between border-b border-border px-3">
      <span class="text-sm font-medium text-foreground">{{ t('deploymentDialogue.history') }}</span>
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
    <ol v-else class="min-h-0 flex-1 space-y-1 overflow-y-auto p-2">
      <li
        v-for="conversation in conversations"
        :key="conversation.id"
        class="group flex min-w-0 rounded-md transition-colors"
        :class="
          conversation.id === activeConversationId
            ? 'bg-accent text-accent-foreground'
            : 'text-foreground hover:bg-muted'
        "
      >
        <button
          type="button"
          class="min-w-0 flex-1 px-2.5 py-2 text-left text-sm"
          @click="emit('select', conversation.id)"
        >
          <span class="block truncate" :title="conversation.title">{{ conversation.title }}</span>
          <span class="mt-0.5 block text-xs text-muted-foreground">
            {{ formatRelativeTime(conversation.updated_at) }}
          </span>
        </button>
        <SelectControl
          v-if="conversation.id === activeConversationId"
          :options="conversationActions"
          :placeholder="t('deploymentDialogue.conversationActions')"
          width-class="w-9 shrink-0 self-stretch !h-auto !rounded-none !border-0 !bg-transparent !px-2 hover:!bg-muted/50"
          @update:model-value="handleConversationAction($event, conversation)"
        >
          <template #trigger>
            <span
              :aria-label="t('deploymentDialogue.conversationActions')"
              :title="t('deploymentDialogue.conversationActions')"
            >
              <Ellipsis class="size-4" />
            </span>
          </template>
        </SelectControl>
      </li>
    </ol>
  </nav>
</template>

<script setup lang="ts">
  import { Ellipsis, Plus } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppSpinner from '@/components/AppSpinner.vue';
  import SelectControl, { type SelectOptionValue } from '@/components/SelectControl.vue';
  import type { DeploymentDialogueConversation } from '@/gen/proto/orbit/v1/dialogue/dialogue';
  import { formatRelativeTime } from '@/utils/time';

  defineProps<{
    conversations: DeploymentDialogueConversation[];
    activeConversationId?: string;
    loading: boolean;
  }>();

  const emit = defineEmits<{
    new: [];
    select: [conversationId: string];
    delete: [conversation: DeploymentDialogueConversation];
  }>();

  const { t } = useI18n();
  const conversationActions = computed(() => [{ value: 'delete', label: t('common.delete') }]);

  function handleConversationAction(
    action: SelectOptionValue,
    conversation: DeploymentDialogueConversation
  ) {
    if (action === 'delete') {
      emit('delete', conversation);
    }
  }
</script>
