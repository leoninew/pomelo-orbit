// @vitest-environment happy-dom
import { createApp, nextTick, ref } from 'vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { provideLogStreamCache, useLogStream } from './useLogStream';
import type { LogState } from './useLogStream';
import type { LogResource } from '@/api/log/log';
import type { LogStreamEvent } from '@/gen/proto/orbit/v1/common/log_stream';
import { ApiError } from '@/utils/request';

const { readLogStream, toastError } = vi.hoisted(() => ({
  readLogStream: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock('@/api/log/log', async (original) => ({
  ...(await original<typeof import('@/api/log/log')>()),
  readLogStream,
}));
vi.mock('@/composables/useToast', () => ({ useToast: () => ({ error: toastError }) }));
const callbacks: ((event: LogStreamEvent) => void)[] = [];
function event(type: string, data: Partial<LogStreamEvent> = {}): LogStreamEvent {
  return {
    type,
    source_id: 'source',
    cursor: '',
    data_base64: '',
    status: '',
    message: '',
    code: '',
    request_id: '',
    http_status: 0,
    retryable: false,
    ...data,
  };
}
function setup() {
  vi.useFakeTimers();
  readLogStream.mockImplementation((_resource, _cursor, signal, callback) => {
    callbacks.push(callback);
    return new Promise<void>((resolve) =>
      signal.addEventListener('abort', () => resolve(), { once: true })
    );
  });
  const resource = ref<LogResource>({ projectId: 'project-1', path: '/log', kind: 'file' });
  const open = ref(true);
  let stream!: ReturnType<typeof useLogStream>;
  const app = createApp({
    setup() {
      provideLogStreamCache(() => resource.value.projectId);
      stream = useLogStream(resource, open);
      return () => null;
    },
  });
  app.mount(document.createElement('div'));
  return { app, stream, resource, open };
}
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  readLogStream.mockReset();
  toastError.mockReset();
  callbacks.length = 0;
});

describe('useLogStream', () => {
  it('preserves UTF-8 decoder, cursor and reading state across close/reopen', async () => {
    const { app, stream, open } = setup();
    const bytes = new TextEncoder().encode('中文');
    callbacks[0]?.(
      event('chunk', { data_base64: btoa(String.fromCharCode(...bytes.slice(0, 2))), cursor: '2' })
    );
    const state = stream.state.value as LogState;
    state.setFollow(false);
    open.value = false;
    await nextTick();
    open.value = true;
    await nextTick();
    expect(readLogStream).toHaveBeenLastCalledWith(
      expect.anything(),
      '2',
      expect.anything(),
      expect.anything()
    );
    expect(stream.state.value?.follow).toBe(false);
    callbacks[1]?.(
      event('chunk', { data_base64: btoa(String.fromCharCode(...bytes.slice(2))), cursor: '6' })
    );
    callbacks[1]?.(event('complete', { status: 'faulted' }));
    expect(stream.state.value?.text).toBe('中文');
    expect(stream.state.value?.status).toBe('complete');
    expect(stream.state.value?.taskStatus).toBe('faulted');
    app.unmount();
  });

  it('rejects late old resource output and stops retrying deterministic errors', async () => {
    const { app, stream, resource, open } = setup();
    callbacks[0]?.(event('chunk', { data_base64: btoa('old'), cursor: 'old-cursor' }));
    resource.value = { projectId: 'project-2', path: '/other', kind: 'file' };
    await nextTick();
    callbacks[0]?.(event('chunk', { data_base64: btoa('late') }));
    expect(stream.state.value?.text).toBe('');
    readLogStream.mockRejectedValue(new ApiError('Permission denied', 403, 'forbidden'));
    open.value = false;
    await nextTick();
    open.value = true;
    await nextTick();
    await Promise.resolve();
    await Promise.resolve();
    expect(stream.state.value?.status).toBe('error');
    expect(toastError).toHaveBeenCalledWith('Permission denied');
    const calls = readLogStream.mock.calls.length;
    await vi.advanceTimersByTimeAsync(60_000);
    expect(readLogStream).toHaveBeenCalledTimes(calls);
    app.unmount();
  });

  it('keeps received output and reports a continuous reconnect failure through one Toast', async () => {
    const { app, stream, open } = setup();
    callbacks[0]?.(event('chunk', { data_base64: btoa('received\n'), cursor: 'cursor-1' }));
    open.value = false;
    await nextTick();
    readLogStream.mockRejectedValue(new ApiError('Log source unavailable', 503, 'unavailable'));
    open.value = true;
    await nextTick();
    await Promise.resolve();
    expect(stream.state.value?.status).toBe('reconnecting');
    expect(stream.state.value?.text).toBe('received\n');
    await vi.advanceTimersByTimeAsync(10_000);
    expect(readLogStream.mock.calls.length).toBeGreaterThan(2);
    expect(toastError).toHaveBeenCalledOnce();
    expect(toastError).toHaveBeenCalledWith('Log source unavailable');
    app.unmount();
  });

  it('backs off when a ready handshake is followed by a source read failure', async () => {
    const { app, open } = setup();
    vi.spyOn(Math, 'random').mockReturnValue(0.5);
    open.value = false;
    await nextTick();
    readLogStream.mockImplementation((_resource, _cursor, _signal, callback) => {
      callback(event('ready'));
      return Promise.reject(new ApiError('Log source unavailable', 503, 'unavailable'));
    });
    open.value = true;
    await nextTick();
    await Promise.resolve();
    let calls = readLogStream.mock.calls.length;
    for (const delay of [1000, 2000, 4000, 8000, 15_000]) {
      await vi.advanceTimersByTimeAsync(delay - 1);
      expect(readLogStream).toHaveBeenCalledTimes(calls);
      await vi.advanceTimersByTimeAsync(1);
      expect(readLogStream).toHaveBeenCalledTimes(++calls);
    }
    expect(toastError).toHaveBeenCalledOnce();
    app.unmount();
  });
});
