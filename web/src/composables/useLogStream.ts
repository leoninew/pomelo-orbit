import {
  computed,
  inject,
  onScopeDispose,
  provide,
  shallowReactive,
  shallowRef,
  toValue,
  watch,
} from 'vue';
import type { InjectionKey, MaybeRefOrGetter } from 'vue';
import type { editor } from 'monaco-editor';
import { logResourceKey, readLogStream } from '@/api/log/log';
import type { LogResource } from '@/api/log/log';
import type { LogStreamEvent } from '@/gen/proto/orbit/v1/common/log_stream';
import { handleUnauthorized } from '@/utils/handle-unauthorized';
import { LogBuffer, LogReplayWindow } from '@/utils/log-buffer';
import type { LogPatch } from '@/utils/log-buffer';
import { ApiError } from '@/utils/request';
import { useToast } from '@/composables/useToast';

const BATCH_INTERVAL = 100;
const CLOSED_CACHE_LIMIT = 4;
const MAX_RETRY_DELAY = 15_000;

export interface LogState {
  text: string;
  status: 'connecting' | 'streaming' | 'waiting' | 'reconnecting' | 'complete' | 'error' | 'closed';
  taskStatus: string;
  gap: boolean;
  truncated: boolean;
  follow: boolean;
  viewState: editor.ICodeEditorViewState | null;
  version: number;
  patch: LogPatch | null;
  setFollow: (value: boolean) => void;
  saveViewState: (value: editor.ICodeEditorViewState | null) => void;
}

class LogSession {
  readonly state = shallowReactive<LogState>({
    text: '',
    status: 'connecting',
    taskStatus: '',
    gap: false,
    truncated: false,
    follow: true,
    viewState: null,
    version: 0,
    patch: null,
    setFollow: (value) => {
      this.state.follow = value;
    },
    saveViewState: (value) => {
      this.state.viewState = value;
    },
  });
  readonly buffer = new LogBuffer();
  readonly toast = useToast();
  replay = new LogReplayWindow();
  decoder = new TextDecoder();
  source = '';
  cursor = '';
  pending = '';
  pendingCR = '';
  controller?: AbortController;
  timer?: ReturnType<typeof setTimeout>;
  retryTimer?: ReturnType<typeof setTimeout>;
  complete = false;
  invalidated = false;

  constructor(readonly resource: LogResource) {}

  flush() {
    clearTimeout(this.timer);
    this.timer = undefined;
    if (!this.pending) {
      return;
    }
    this.state.patch = this.buffer.append(this.pending);
    this.pending = '';
    this.state.text = this.buffer.text;
    this.state.truncated ||= this.state.patch.removedChars > 0;
    this.state.version++;
  }

  append(text: string) {
    text = this.pendingCR + text;
    this.pendingCR = text.endsWith('\r') ? '\r' : '';
    this.pending += (this.pendingCR ? text.slice(0, -1) : text).replace(/\r\n|\r/g, '\n');
    if (!this.timer) {
      this.timer = setTimeout(() => this.flush(), BATCH_INTERVAL);
    }
  }

  stop() {
    this.controller?.abort();
    this.controller = undefined;
    clearTimeout(this.retryTimer);
    this.flush();
    if (!this.complete && this.state.status !== 'error') {
      this.state.status = 'closed';
    }
  }

  event(event: LogStreamEvent) {
    switch (event.type) {
      case 'ready':
        this.state.status = 'streaming';
        this.state.taskStatus = event.status;
        this.replay.reconnect();
        if (this.resource.kind === 'container' && this.replay.truncated) {
          this.state.gap = true;
        }
        break;
      case 'waiting':
        this.state.status = 'waiting';
        break;
      case 'gap':
        this.state.gap = true;
        break;
      case 'source_changed':
        this.append(this.decoder.decode());
        this.decoder = new TextDecoder();
        this.source = event.source_id;
        break;
      case 'chunk': {
        if (this.source && this.source !== event.source_id) {
          this.append(this.decoder.decode());
          this.decoder = new TextDecoder();
        }
        this.source = event.source_id;
        let bytes: Uint8Array;
        try {
          bytes = Uint8Array.from(atob(event.data_base64), (char) => char.charCodeAt(0));
        } catch {
          throw new ApiError(
            'Invalid log bytes',
            undefined,
            'contract_mismatch',
            undefined,
            'contract_mismatch'
          );
        }
        const text = this.decoder.decode(bytes, { stream: true });
        if (this.resource.kind === 'file' || this.replay.accept(event.source_id, text)) {
          this.append(text);
        }
        // Decoded text and partial UTF-8 state are retained before advancing the cursor.
        if (event.cursor) {
          this.cursor = event.cursor;
        }
        this.state.status = 'streaming';
        break;
      }
      case 'complete':
        this.append(this.decoder.decode());
        if (this.pendingCR) {
          this.pending += '\n';
          this.pendingCR = '';
        }
        this.flush();
        if (event.cursor) {
          this.cursor = event.cursor;
        }
        this.state.taskStatus = event.status;
        this.state.status = 'complete';
        this.complete = true;
        break;
      case 'error':
        if (event.http_status === 401) {
          handleUnauthorized();
        }
        throw new ApiError(
          event.message,
          event.http_status,
          event.retryable ? event.code : 'log_stream_stopped',
          event.request_id
        );
    }
  }

  start() {
    this.stop();
    if (this.invalidated) {
      this.buffer.clear();
      this.replay = new LogReplayWindow();
      this.decoder = new TextDecoder();
      this.cursor = '';
      this.source = '';
      this.pendingCR = '';
      this.invalidated = false;
      Object.assign(this.state, {
        text: '',
        truncated: false,
        gap: false,
        follow: true,
        viewState: null,
        patch: null,
      });
      this.state.version++;
    }
    if (this.complete) {
      return;
    }
    const controller = new AbortController();
    this.controller = controller;
    this.state.status = 'connecting';
    let attempts = 0;
    let failureNotified = false;
    const connect = async () => {
      try {
        await readLogStream(this.resource, this.cursor, controller.signal, (event) => {
          if (controller.signal.aborted || this.controller !== controller) {
            return;
          }
          this.event(event);
          if (event.type === 'chunk' || event.type === 'waiting') {
            attempts = 0;
            failureNotified = false;
          }
        });
        if (this.complete || controller.signal.aborted) {
          return;
        }
        throw new ApiError(
          'Log connection interrupted',
          undefined,
          'network_error',
          undefined,
          'network'
        );
      } catch (error) {
        if (controller.signal.aborted || this.controller !== controller) {
          return;
        }
        this.flush();
        const apiError = error instanceof ApiError ? error : undefined;
        this.invalidated = apiError?.status === 409;
        if (!failureNotified) {
          this.toast.error(error instanceof Error ? error.message : 'Log read failed');
          failureNotified = true;
        }
        const retryable =
          apiError &&
          apiError.kind !== 'contract_mismatch' &&
          apiError.code !== 'log_stream_stopped' &&
          (!apiError.status || apiError.status === 429 || apiError.status >= 500);
        if (!retryable) {
          this.state.status = 'error';
          return;
        }
        this.state.status = 'reconnecting';
        const delay = Math.min(
          1000 * 2 ** attempts++ * (0.9 + Math.random() * 0.2),
          MAX_RETRY_DELAY
        );
        this.retryTimer = setTimeout(() => void connect(), delay);
      }
    };
    void connect();
  }
}

export class LogStreamCache {
  private sessions = new Map<string, LogSession>();
  get(resource: LogResource) {
    const key = logResourceKey(resource);
    const session = this.sessions.get(key) ?? new LogSession(resource);
    this.sessions.delete(key);
    this.sessions.set(key, session);
    return session;
  }
  release(session: LogSession) {
    session.stop();
    const closed = [...this.sessions.entries()].filter(([, value]) => !value.controller);
    for (const [key] of closed.slice(0, Math.max(0, closed.length - CLOSED_CACHE_LIMIT))) {
      this.sessions.delete(key);
    }
  }
  clear() {
    for (const session of this.sessions.values()) {
      session.stop();
    }
    this.sessions.clear();
  }
}

const cacheKey: InjectionKey<LogStreamCache> = Symbol('pageLogCache');

export function provideLogStreamCache(scope: MaybeRefOrGetter<string>) {
  const cache = new LogStreamCache();
  provide(cacheKey, cache);
  watch(
    () => toValue(scope),
    () => cache.clear(),
    { flush: 'sync' }
  );
  onScopeDispose(() => cache.clear());
  return cache;
}

export function useLogStream(
  resource: MaybeRefOrGetter<LogResource | undefined>,
  active: MaybeRefOrGetter<boolean>
) {
  const cache = inject(cacheKey, undefined) ?? new LogStreamCache();
  const state = shallowRef<LogState>();
  let session: LogSession | undefined;
  const resourceKey = computed(() => {
    const value = toValue(resource);
    return value ? logResourceKey(value) : '';
  });
  watch(
    [resourceKey, () => toValue(active)],
    ([key, enabled]) => {
      if (session) {
        cache.release(session);
      }
      const value = toValue(resource);
      session = key && value ? cache.get(value) : undefined;
      state.value = session?.state;
      if (session && enabled) {
        session.start();
      }
    },
    { immediate: true }
  );
  onScopeDispose(() => {
    if (session) {
      cache.release(session);
    }
  });
  return { state };
}
