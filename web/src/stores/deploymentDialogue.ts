import { defineStore } from 'pinia';
import { ref } from 'vue';
import type {
  DeploymentDialogueConversation,
  DeploymentDialogueMessage,
  DeploymentDialogueToolCall,
} from '@/gen/proto/orbit/v1/dialogue/dialogue';

export type DeploymentDialogueConversationMessage = DeploymentDialogueMessage & {
  tool_calls?: DeploymentDialogueToolCall[];
  is_error?: boolean;
};

export const useDeploymentDialogueStore = defineStore('deploymentDialogue', () => {
  const messages = ref<DeploymentDialogueConversationMessage[]>([]);
  const conversations = ref<DeploymentDialogueConversation[]>([]);
  const activeConversationId = ref<string>();

  function clearMessages() {
    messages.value = [];
  }

  function resetConversation() {
    activeConversationId.value = undefined;
    clearMessages();
  }

  return { messages, conversations, activeConversationId, clearMessages, resetConversation };
});
