// @vitest-environment happy-dom
/* eslint-disable vue/one-component-per-file -- The test uses child component doubles. */
import { createApp, h, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useProjectStore } from '@/stores/project';
import { provideLogStreamCache } from '@/composables/useLogStream';
import type { LogStreamEvent } from '@/gen/proto/orbit/v1/common/log_stream';

const { readLogStream } = vi.hoisted(() => ({ readLogStream: vi.fn() }));
vi.mock('@/api/log/log', async (original) => ({
  ...(await original<typeof import('@/api/log/log')>()),
  readLogStream,
}));
vi.mock('@/components/LogDrawer.vue', async () => {
  const { defineComponent, h: render } = await import('vue');
  return {
    default: defineComponent({
      props: { state: Object },
      setup(props) {
        return () => render('div', props.state?.text);
      },
    }),
  };
});
import RuntimeContainerLogsDrawer from './RuntimeContainerLogsDrawer.vue';

const callbacks: ((event: LogStreamEvent) => void)[] = [];
const signals: AbortSignal[] = [];
function event(type: string, extra: Partial<LogStreamEvent> = {}): LogStreamEvent {
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
    ...extra,
  };
}
async function flush() {
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

afterEach(() => {
  vi.useRealTimers();
  readLogStream.mockReset();
  callbacks.length = 0;
  signals.length = 0;
});

describe('RuntimeContainerLogsDrawer', () => {
  it('keeps streaming after deployment completion and restores content/cursor after unmount', async () => {
    vi.useFakeTimers();
    readLogStream.mockImplementation((_resource, _cursor, signal, callback) => {
      signals.push(signal);
      callbacks.push(callback);
      return new Promise<void>((resolve) =>
        signal.addEventListener('abort', () => resolve(), { once: true })
      );
    });
    const pinia = createPinia();
    setActivePinia(pinia);
    useProjectStore().setActiveProject('project-1');
    const visible = ref(true);
    const host = document.createElement('div');
    const app = createApp({
      setup() {
        provideLogStreamCache(() => 'project-1');
        return () =>
          visible.value
            ? h(RuntimeContainerLogsDrawer, {
                open: true,
                target: {
                  applicationId: 'app-1',
                  serviceId: 'service-1',
                  component: 'web',
                  title: 'Logs',
                  deploymentId: 'completed-deployment',
                },
              })
            : null;
      },
    });
    app.use(pinia);
    app.mount(host);
    callbacks[0]?.(event('ready'));
    callbacks[0]?.(event('chunk', { data_base64: btoa('started\n'), cursor: 'cursor-1' }));
    await vi.advanceTimersByTimeAsync(100);
    expect(host.textContent).toBe('started\n');
    expect(signals[0]?.aborted).toBe(false);
    visible.value = false;
    await flush();
    expect(signals[0]?.aborted).toBe(true);
    visible.value = true;
    await flush();
    expect(readLogStream).toHaveBeenLastCalledWith(
      expect.objectContaining({
        path: '/api/application/app-1/log/stream',
        params: { service_id: 'service-1', component: 'web' },
      }),
      'cursor-1',
      expect.any(AbortSignal),
      expect.any(Function)
    );
    expect(host.textContent).toBe('started\n');
    callbacks[1]?.(event('ready'));
    callbacks[1]?.(event('chunk', { data_base64: btoa('continued\n'), cursor: 'cursor-2' }));
    await vi.advanceTimersByTimeAsync(100);
    expect(host.textContent).toBe('started\ncontinued\n');
    app.unmount();
    expect(signals[1]?.aborted).toBe(true);
  });
});
