import { afterEach, describe, expect, it, vi } from 'vitest';
import { newDeploymentDialogueConversationId } from './deploymentDialogueConversationId';

describe('newDeploymentDialogueConversationId', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('preserves the existing UUID-based ID format when randomUUID is available', () => {
    const randomUUID = vi.fn(() => 'e7f1cb49-48dc-47f7-b770-4a6b5c3e2ffb');
    vi.stubGlobal('crypto', { randomUUID });

    expect(newDeploymentDialogueConversationId()).toBe('E7F1CB4948DC47F7B7704A6B5C');
    expect(randomUUID).toHaveBeenCalledOnce();
  });

  it('uses getRandomValues when randomUUID is unavailable', () => {
    const getRandomValues = vi.fn((bytes: Uint8Array) => {
      bytes.set([0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
      return bytes;
    });
    vi.stubGlobal('crypto', { getRandomValues });

    expect(newDeploymentDialogueConversationId()).toBe('000102030405060708090A0B0C');
    expect(getRandomValues).toHaveBeenCalledWith(expect.any(Uint8Array));
  });
});
