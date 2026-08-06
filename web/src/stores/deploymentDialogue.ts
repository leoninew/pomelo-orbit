import { defineStore } from 'pinia';
import { ref } from 'vue';
import type {
  DeploymentDialogueMessage,
  DeploymentDialogueToolCall,
} from '@/gen/proto/orbit/v1/dialogue/dialogue';

export type DeploymentDialogueConversationMessage = DeploymentDialogueMessage & {
  tool_calls?: DeploymentDialogueToolCall[];
  is_error?: boolean;
};

export const useDeploymentDialogueStore = defineStore('deploymentDialogue', () => {
  const messages = ref<DeploymentDialogueConversationMessage[]>([]);

  function clearMessages() {
    messages.value = [];
  }

  return { messages, clearMessages };
});
