import type {
  DeploymentDialogueConversationDetailResp,
  DeploymentDialogueConversationListResp,
  DeploymentDialogueStreamEvent,
  DeploymentDialogueTurnReq,
  DeploymentDialogueTurnResp,
} from '@/gen/proto/orbit/v1/dialogue/dialogue';
import runtimeConfig from '@/config';
import { useAuthStore } from '@/stores/auth';
import { handleUnauthorized } from '@/utils/handle-unauthorized';
import request, { ApiError, toApiError } from '@/utils/request';

const DIALOGUE_TIMEOUT = 600000;

export const dialogueApi = {
  listConversations(projectId: string): Promise<DeploymentDialogueConversationListResp> {
    return request.get('/api/deployment-dialogue/conversation', {
      params: { project_id: projectId },
    });
  },

  getConversation(conversationId: string): Promise<DeploymentDialogueConversationDetailResp> {
    return request.get(`/api/deployment-dialogue/conversation/${conversationId}`);
  },

  deleteConversation(conversationId: string): Promise<void> {
    return request.delete(`/api/deployment-dialogue/conversation/${conversationId}`);
  },

  completeTurn(data: DeploymentDialogueTurnReq): Promise<DeploymentDialogueTurnResp> {
    return request.post('/api/deployment-dialogue/turn', data, { timeout: DIALOGUE_TIMEOUT });
  },

  async completeTurnStream(
    data: DeploymentDialogueTurnReq,
    onEvent: (event: DeploymentDialogueStreamEvent) => void
  ): Promise<void> {
    const controller = new AbortController();
    const timeoutId = window.setTimeout(() => controller.abort(), DIALOGUE_TIMEOUT);
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    const token = useAuthStore().token;
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }

    try {
      const response = await fetch(
        `${runtimeConfig.publicUrl}/api/deployment-dialogue/turn/stream`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify(data),
          signal: controller.signal,
        }
      );
      if (!response.ok) {
        await throwStreamResponseError(response);
      }
      if (!response.body || !response.headers.get('content-type')?.includes('text/event-stream')) {
        throw new ApiError(
          '服务响应格式异常',
          response.status,
          'contract_mismatch',
          undefined,
          'contract_mismatch'
        );
      }
      await readEventStream(response.body, onEvent);
    } catch (error: unknown) {
      if (error instanceof ApiError) {
        throw error;
      }
      if (error instanceof Error && error.name === 'AbortError') {
        throw new ApiError(
          'Dialogue request timed out.',
          undefined,
          'request_timeout',
          undefined,
          'network'
        );
      }
      throw new ApiError(
        '网络连接失败，请检查网络设置',
        undefined,
        'network_error',
        undefined,
        'network'
      );
    } finally {
      window.clearTimeout(timeoutId);
    }
  },
};

async function throwStreamResponseError(response: Response): Promise<never> {
  let responseBody: unknown;
  try {
    responseBody = await response.json();
  } catch {
    responseBody = undefined;
  }
  const apiError = toApiError(response.status, responseBody);
  if (response.status === 401 && apiError.kind !== 'contract_mismatch') {
    handleUnauthorized();
    throw new ApiError(
      '登录已过期，请重新登录',
      response.status,
      apiError.code,
      apiError.requestId
    );
  }
  throw apiError;
}

async function readEventStream(
  body: ReadableStream<Uint8Array>,
  onEvent: (event: DeploymentDialogueStreamEvent) => void
): Promise<void> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      buffer += decoder.decode(value, { stream: !done });
      buffer = consumeEventBlocks(buffer, onEvent, done);
      if (done) {
        if (buffer.trim() !== '') {
          consumeEventBlock(buffer, onEvent);
        }
        return;
      }
    }
  } finally {
    reader.releaseLock();
  }
}

function consumeEventBlocks(
  buffer: string,
  onEvent: (event: DeploymentDialogueStreamEvent) => void,
  isFinal: boolean
): string {
  const blocks = buffer.split(/\r?\n\r?\n/);
  const remainder = isFinal ? '' : (blocks.pop() ?? '');
  for (const block of blocks) {
    consumeEventBlock(block, onEvent);
  }
  return remainder;
}

function consumeEventBlock(block: string, onEvent: (event: DeploymentDialogueStreamEvent) => void) {
  const data = block
    .split(/\r?\n/)
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice('data:'.length).trimStart())
    .join('\n');
  if (data) {
    onEvent(parseStreamEvent(data));
  }
}

function parseStreamEvent(value: string): DeploymentDialogueStreamEvent {
  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    throw new ApiError(
      '服务响应格式异常',
      undefined,
      'contract_mismatch',
      undefined,
      'contract_mismatch'
    );
  }
  if (
    typeof parsed !== 'object' ||
    parsed === null ||
    typeof (parsed as { type?: unknown }).type !== 'string'
  ) {
    throw new ApiError(
      '服务响应格式异常',
      undefined,
      'contract_mismatch',
      undefined,
      'contract_mismatch'
    );
  }

  const event = parsed as Partial<DeploymentDialogueStreamEvent>;
  return {
    type: (parsed as { type: string }).type,
    message: typeof event.message === 'string' ? event.message : '',
    tool_call: parseToolCall(event.tool_call),
    code: typeof event.code === 'string' ? event.code : '',
    request_id: typeof event.request_id === 'string' ? event.request_id : '',
  };
}

function parseToolCall(value: unknown): DeploymentDialogueStreamEvent['tool_call'] {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (typeof value !== 'object' || value === null) {
    throw new ApiError(
      '服务响应格式异常',
      undefined,
      'contract_mismatch',
      undefined,
      'contract_mismatch'
    );
  }
  const toolCall = value as Record<string, unknown>;
  if (
    typeof toolCall.name !== 'string' ||
    typeof toolCall.arguments_json !== 'string' ||
    typeof toolCall.result_json !== 'string' ||
    typeof toolCall.is_error !== 'boolean'
  ) {
    throw new ApiError(
      '服务响应格式异常',
      undefined,
      'contract_mismatch',
      undefined,
      'contract_mismatch'
    );
  }
  return {
    name: toolCall.name,
    arguments_json: toolCall.arguments_json,
    result_json: toolCall.result_json,
    is_error: toolCall.is_error,
  };
}
