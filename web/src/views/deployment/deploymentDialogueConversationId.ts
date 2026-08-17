const conversationIdByteLength = 13;

export function newDeploymentDialogueConversationId(): string {
  if (typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID().replace(/-/g, '').slice(0, 26).toUpperCase();
  }

  const bytes = globalThis.crypto.getRandomValues(new Uint8Array(conversationIdByteLength));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0'))
    .join('')
    .toUpperCase();
}
