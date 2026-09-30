// @vitest-environment happy-dom
/* eslint-disable vue/one-component-per-file -- The test uses log component doubles. */
import { createPinia, setActivePinia } from 'pinia';
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter, RouterView } from 'vue-router';
import { afterEach, assert, expect, it, vi } from 'vitest';
import { deploymentApi } from '@/api/deployment/deployment';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import type { LogStreamEvent } from '@/gen/proto/orbit/v1/common/log_stream';
import DeploymentDetail from './DeploymentDetail.vue';

const { readLogStream } = vi.hoisted(() => ({ readLogStream: vi.fn() }));
vi.mock('@/api/deployment/deployment', () => ({ deploymentApi: { get: vi.fn() } }));
vi.mock('@/api/log/log', async (original) => ({
  ...(await original<typeof import('@/api/log/log')>()),
  readLogStream,
}));
vi.mock('@/components/LogView.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: { state: Object },
      setup: (props) => () => h('div', { 'data-log-view': '' }, props.state?.text),
    }),
  };
});
vi.mock('@/components/LogDrawer.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: { open: Boolean, title: String, state: Object },
      emits: ['update:open'],
      setup:
        (props, { emit }) =>
        () =>
          props.open
            ? h('aside', { 'data-log-drawer': '', 'aria-label': props.title }, [
                props.state?.text,
                h('button', { onClick: () => emit('update:open', false) }, 'Close'),
              ])
            : null,
    }),
  };
});

let app: App | undefined;
let host: HTMLDivElement | undefined;

afterEach(() => {
  app?.unmount();
  host?.remove();
  app = undefined;
  host = undefined;
  vi.useRealTimers();
  vi.resetAllMocks();
});

it('keeps operation logs mounted while the container drawer streams, closes and resumes', async () => {
  vi.useFakeTimers();
  const streams = new Map<string, { signal: AbortSignal; emit: (event: LogStreamEvent) => void }>();
  readLogStream.mockImplementation((resource, _cursor, signal, emit) => {
    streams.set(resource.path, { signal, emit });
    return new Promise<void>((resolve) =>
      signal.addEventListener('abort', () => resolve(), { once: true })
    );
  });
  vi.mocked(deploymentApi.get).mockResolvedValue({
    id: 'deployment-1',
    application_name: 'Application',
    operation_type: 'deploy',
    trigger_type: 'manual',
    command_text: 'docker compose up -d',
    status: 'running',
    started_at: '2026-09-30T08:00:00Z',
    created_at: '2026-09-30T08:00:00Z',
    is_rollback: false,
  });
  const pinia = createPinia();
  setActivePinia(pinia);
  useProjectStore().setActiveProject('project-1');
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/deployment/:id', component: DeploymentDetail }],
  });
  await router.push('/deployment/deployment-1');
  host = document.createElement('div');
  document.body.append(host);
  app = createApp(RouterView);
  app.use(pinia).use(router).use(i18n).mount(host);
  await vi.waitFor(() => expect(readLogStream).toHaveBeenCalledTimes(1));

  const operationPath = '/api/deployment/deployment-1/log/stream';
  const containerPath = '/api/deployment/deployment-1/container-log/stream';
  const operationStream = streams.get(operationPath);
  assert(operationStream);
  const chunk = (text: string, cursor: string): LogStreamEvent => ({
    type: 'chunk',
    source_id: 'source',
    data_base64: btoa(text),
    cursor,
    status: '',
    message: '',
    code: '',
    request_id: '',
    http_status: 0,
    retryable: false,
  });
  operationStream.emit(chunk('operation output\n', 'operation-cursor'));
  await vi.advanceTimersByTimeAsync(100);
  const operationView = host.querySelector('[data-log-view]');
  assert(operationView);
  const openButton = host.querySelector<HTMLButtonElement>(
    `button[aria-label="${i18n.global.t('service.logs.title')}"]`
  );
  assert(openButton);
  expect(operationView.textContent).toBe('operation output\n');
  expect(operationView.closest('section')?.contains(openButton)).toBe(true);

  openButton.click();
  await nextTick();
  expect(readLogStream).toHaveBeenCalledTimes(2);
  const containerStream = streams.get(containerPath);
  assert(containerStream);
  containerStream.emit(chunk('container output\n', 'container-cursor'));
  operationStream.emit(chunk('operation continues\n', 'operation-cursor-2'));
  await vi.advanceTimersByTimeAsync(100);
  expect(host.querySelector('[data-log-view]')).toBe(operationView);
  expect(operationView.textContent).toBe('operation output\noperation continues\n');
  expect(host.querySelector('[data-log-drawer]')?.textContent).toContain('container output\n');

  const closeButton = host.querySelector<HTMLButtonElement>('[data-log-drawer] button');
  assert(closeButton);
  closeButton.click();
  await nextTick();
  expect(containerStream.signal.aborted).toBe(true);
  expect(operationStream.signal.aborted).toBe(false);
  expect(host.querySelector('[data-log-view]')).toBe(operationView);

  openButton.click();
  await nextTick();
  expect(readLogStream).toHaveBeenCalledTimes(3);
  expect(readLogStream).toHaveBeenLastCalledWith(
    expect.objectContaining({ path: containerPath, kind: 'container' }),
    'container-cursor',
    expect.any(AbortSignal),
    expect.any(Function)
  );
  expect(host.querySelector('[data-log-drawer]')?.textContent).toContain('container output\n');
  expect(host.querySelector('[data-log-view]')).toBe(operationView);
});
