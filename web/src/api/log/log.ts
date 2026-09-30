import runtimeConfig from '@/config';
import { useAuthStore } from '@/stores/auth';
import { handleUnauthorized } from '@/utils/handle-unauthorized';
import { ApiError, toApiError } from '@/utils/request';
import { EventStreamError, EventStreamParser } from '@/utils/event-stream';
import type { LogStreamEvent } from '@/gen/proto/orbit/v1/common/log_stream';

export interface LogResource {
  projectId: string;
  path: string;
  params?: Record<string, string>;
  kind: 'file' | 'container';
}

const CONNECT_TIMEOUT = 15_000;
const INACTIVITY_TIMEOUT = 45_000;
const EVENT_TYPES = new Set([
  'ready',
  'chunk',
  'waiting',
  'source_changed',
  'gap',
  'complete',
  'error',
]);

export function logResourceKey(resource: LogResource) {
  return JSON.stringify([
    resource.projectId,
    resource.path,
    Object.entries(resource.params ?? {}).sort(),
  ]);
}

export async function readLogStream(
  resource: LogResource,
  cursor: string,
  signal: AbortSignal,
  onEvent: (event: LogStreamEvent) => void
) {
  const controller = new AbortController();
  const abort = () => controller.abort();
  signal.addEventListener('abort', abort, { once: true });
  if (signal.aborted) {
    controller.abort();
  }
  let timedOut = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const deadline = (ms: number) => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, ms);
  };
  deadline(CONNECT_TIMEOUT);
  const params = new URLSearchParams({ ...resource.params, project_id: resource.projectId });
  if (cursor) {
    params.set('cursor', cursor);
  }
  const headers: Record<string, string> = { Accept: 'text/event-stream' };
  const token = useAuthStore().token;
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
  try {
    const response = await fetch(`${runtimeConfig.publicUrl}${resource.path}?${params}`, {
      headers,
      signal: controller.signal,
    });
    if (!response.ok) {
      let body: unknown;
      try {
        body = await response.json();
      } catch {
        body = undefined;
      }
      const error = toApiError(response.status, body);
      if (response.status === 401 && error.kind !== 'contract_mismatch') {
        handleUnauthorized();
      }
      throw error;
    }
    if (!response.body || !response.headers.get('content-type')?.includes('text/event-stream')) {
      throw new ApiError(
        'Invalid log stream response',
        response.status,
        'contract_mismatch',
        undefined,
        'contract_mismatch'
      );
    }
    deadline(INACTIVITY_TIMEOUT);
    const decoder = new TextDecoder();
    const parser = new EventStreamParser((data) => {
      let event: LogStreamEvent;
      try {
        event = JSON.parse(data) as LogStreamEvent;
      } catch {
        throw new ApiError(
          'Invalid log event',
          undefined,
          'contract_mismatch',
          undefined,
          'contract_mismatch'
        );
      }
      if (
        !event ||
        !EVENT_TYPES.has(event.type) ||
        (event.type === 'chunk' && (!event.source_id || typeof event.data_base64 !== 'string'))
      ) {
        throw new ApiError(
          'Invalid log event',
          undefined,
          'contract_mismatch',
          undefined,
          'contract_mismatch'
        );
      }
      onEvent(event);
    });
    reader = response.body.getReader();
    for (;;) {
      const { value, done } = await reader.read();
      if (done) {
        parser.feed(decoder.decode());
        parser.finish();
        return;
      }
      deadline(INACTIVITY_TIMEOUT);
      parser.feed(decoder.decode(value, { stream: true }));
    }
  } catch (error) {
    if (signal.aborted) {
      throw error;
    }
    if (error instanceof ApiError) {
      throw error;
    }
    if (error instanceof EventStreamError && error.message !== 'Incomplete SSE event') {
      throw new ApiError(
        error.message,
        undefined,
        'contract_mismatch',
        undefined,
        'contract_mismatch'
      );
    }
    throw new ApiError(
      timedOut ? 'Log stream timed out' : 'Log connection interrupted',
      undefined,
      'network_error',
      undefined,
      'network'
    );
  } finally {
    clearTimeout(timer);
    signal.removeEventListener('abort', abort);
    controller.abort();
    await reader?.cancel().catch(() => undefined);
    reader?.releaseLock();
  }
}
