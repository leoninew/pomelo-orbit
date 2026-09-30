// @vitest-environment happy-dom
import { createApp, h, nextTick, reactive, type App } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { projectEnvironmentApi } from '@/api/project/environment';
import i18n from '@/i18n';
import EnvironmentTerminalDrawer from './EnvironmentTerminalDrawer.vue';

const terminalMock = vi.hoisted(() => ({
  input: undefined as ((data: string) => void) | undefined,
  open: vi.fn<(element: HTMLElement) => void>(),
  write: vi.fn(),
  dispose: vi.fn(),
  fit: vi.fn(),
  reset: vi.fn(),
  focus: vi.fn(),
  blur: vi.fn(),
}));

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 80;
    rows = 24;
    options = {};
    loadAddon() {}
    open = terminalMock.open;
    onData(callback: (data: string) => void) {
      terminalMock.input = callback;
    }
    write = terminalMock.write;
    dispose = terminalMock.dispose;
    reset = terminalMock.reset;
    focus = terminalMock.focus;
    blur = terminalMock.blur;
  },
}));
vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    fit = terminalMock.fit;
  },
}));
vi.mock('@/api/project/environment', () => ({
  projectEnvironmentApi: { terminalTicket: vi.fn() },
}));

class TestWebSocket {
  static OPEN = 1;
  static instances: TestWebSocket[] = [];
  readyState = 0;
  binaryType = '';
  onopen?: () => void;
  onmessage?: (event: { data: string | ArrayBuffer }) => void;
  onclose?: () => void;
  onerror?: () => void;
  send = vi.fn();
  close = vi.fn(() => {
    this.readyState = 3;
  });
  constructor(
    public url: URL,
    public protocols: string[]
  ) {
    TestWebSocket.instances.push(this);
  }
  open() {
    this.readyState = TestWebSocket.OPEN;
    this.onopen?.();
  }
  message(data: string | ArrayBuffer) {
    this.onmessage?.({ data });
  }
}

let app: App | undefined;
let container: HTMLDivElement | undefined;

beforeEach(() => {
  TestWebSocket.instances = [];
  vi.stubGlobal('WebSocket', TestWebSocket);
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      disconnect() {}
    }
  );
  window.__CONFIG__ = { publicUrl: 'https://api.orbit.example' };
  vi.mocked(projectEnvironmentApi.terminalTicket).mockResolvedValue({ ticket: 'test-ticket' });
});

afterEach(() => {
  app?.unmount();
  container?.remove();
  app = undefined;
  container = undefined;
  window.__CONFIG__ = undefined;
  vi.unstubAllGlobals();
  vi.useRealTimers();
  vi.clearAllMocks();
});

async function mountTerminal() {
  const props = reactive({
    open: true,
    projectId: 'project-1',
    host: 'ssh-host',
    username: 'managed-user',
  });
  container = document.createElement('div');
  document.body.append(container);
  app = createApp({ setup: () => () => h(EnvironmentTerminalDrawer, props) });
  app.use(i18n);
  app.mount(container);
  await nextTick();
  await nextTick();
  return props;
}

function socketAt(index: number) {
  const socket = TestWebSocket.instances[index];
  if (!socket) {
    throw new Error('Expected terminal connection');
  }
  return socket;
}

describe('Environment terminal', () => {
  it('uses one connection control to disconnect and reconnect', async () => {
    await mountTerminal();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(1));
    const first = socketAt(0);
    first.open();
    first.message('{"type":"ready"}');
    await nextTick();

    const buttons = document.querySelectorAll<HTMLButtonElement>('[role="dialog"] button');
    expect(buttons).toHaveLength(2);
    const connectionButton = buttons[0];
    if (!connectionButton) {
      throw new Error('Expected terminal connection control');
    }
    expect(connectionButton.getAttribute('aria-label')).toContain(
      i18n.global.t('project.environment.terminal.disconnect')
    );
    expect(connectionButton.querySelectorAll('svg')).toHaveLength(1);

    connectionButton.click();
    await nextTick();
    expect(first.close).toHaveBeenCalledOnce();
    expect(connectionButton.getAttribute('aria-label')).toContain(
      i18n.global.t('project.environment.terminal.reconnect')
    );

    connectionButton.click();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(2));
    socketAt(1).open();
    socketAt(1).message('{"type":"ready"}');
    await nextTick();
    expect(connectionButton.getAttribute('title')).toContain(
      i18n.global.t('project.environment.terminal.connected')
    );
  });

  it('reports a timeout when the WebSocket never becomes ready', async () => {
    vi.useFakeTimers();
    await mountTerminal();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(1));
    const socket = socketAt(0);
    await vi.advanceTimersByTimeAsync(20_000);
    expect(socket.close).toHaveBeenCalledOnce();
    expect(document.body.textContent).toContain(
      i18n.global.t('project.environment.terminal.connectionTimedOut')
    );
  });

  it('connects through the API URL using a ticket subprotocol and transfers terminal data', async () => {
    await mountTerminal();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(1));
    const socket = socketAt(0);
    expect(socket.url.toString()).toBe(
      'wss://api.orbit.example/api/environment/terminal?project_id=project-1'
    );
    expect(socket.protocols).toEqual(['orbit-terminal-v1', 'ticket.test-ticket']);
    socket.open();
    expect(socket.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'resize', cols: 80, rows: 24 })
    );
    terminalMock.input?.('before-ready');
    expect(socket.send).toHaveBeenCalledTimes(1);
    socket.message('{"type":"ready"}');
    terminalMock.input?.('pwd\r');
    expect(socket.send).toHaveBeenLastCalledWith(new TextEncoder().encode('pwd\r'));
    const output = new TextEncoder().encode('remote-output').buffer;
    socket.message(output);
    expect(terminalMock.write).toHaveBeenCalledWith(new Uint8Array(output));
    socket.message('{"type":"exit","code":7}');
    socket.onclose?.();
    await nextTick();
    expect(document.body.textContent).toContain('7');
  });

  it('preserves the connection, output and terminal across drawer close and reopen', async () => {
    const props = await mountTerminal();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(1));
    const socket = socketAt(0);
    socket.open();
    socket.message('{"type":"ready"}');
    const terminalContainer = terminalMock.open.mock.calls[0]?.[0];
    expect(terminalContainer?.isConnected).toBe(true);
    props.open = false;
    await nextTick();
    await vi.waitFor(() => expect(document.querySelector('[role="dialog"]')).toBeNull());
    expect(socket.close).not.toHaveBeenCalled();
    expect(terminalMock.dispose).not.toHaveBeenCalled();
    expect(terminalContainer?.isConnected).toBe(false);
    const fitsBeforeHiddenOutput = terminalMock.fit.mock.calls.length;
    const output = new TextEncoder().encode('output-while-hidden').buffer;
    socket.message(output);
    expect(terminalMock.write).toHaveBeenCalledWith(new Uint8Array(output));
    expect(terminalMock.fit).toHaveBeenCalledTimes(fitsBeforeHiddenOutput);
    props.open = true;
    await nextTick();
    await vi.waitFor(() => expect(terminalContainer?.isConnected).toBe(true));
    expect(TestWebSocket.instances).toHaveLength(1);
    expect(projectEnvironmentApi.terminalTicket).toHaveBeenCalledOnce();
    expect(terminalMock.open).toHaveBeenCalledOnce();
    expect(terminalMock.reset).toHaveBeenCalledOnce();
    terminalMock.input?.('pwd\r');
    expect(socket.send).toHaveBeenLastCalledWith(new TextEncoder().encode('pwd\r'));
    expect(document.body.textContent).toContain(
      i18n.global.t('project.environment.terminal.connected')
    );
    app?.unmount();
    app = undefined;
    expect(socket.close).toHaveBeenCalledOnce();
    expect(terminalMock.dispose).toHaveBeenCalledOnce();
  });

  it.each(['projectId', 'host', 'username'] as const)(
    'ends a hidden session when %s changes',
    async (field) => {
      const props = await mountTerminal();
      await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(1));
      const socket = socketAt(0);
      props.open = false;
      await nextTick();
      props[field] = `${props[field]}-changed`;
      await nextTick();
      expect(socket.close).toHaveBeenCalledOnce();
      expect(terminalMock.dispose).toHaveBeenCalledOnce();
      expect(TestWebSocket.instances).toHaveLength(1);
    }
  );

  it('preserves an explicit disconnect when the drawer reopens and reconnects only on request', async () => {
    const props = await mountTerminal();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(1));
    const socket = socketAt(0);
    socket.open();
    socket.message('{"type":"ready"}');
    await nextTick();
    document
      .querySelector<HTMLButtonElement>(
        `button[aria-label="${i18n.global.t('project.environment.terminal.disconnect')}"]`
      )
      ?.click();
    expect(socket.close).toHaveBeenCalledOnce();
    props.open = false;
    await nextTick();
    props.open = true;
    await nextTick();
    await vi.waitFor(() => expect(document.querySelector('[role="dialog"]')).not.toBeNull());
    expect(projectEnvironmentApi.terminalTicket).toHaveBeenCalledOnce();
    expect(document.body.textContent).toContain(
      i18n.global.t('project.environment.terminal.disconnected')
    );
    document
      .querySelector<HTMLButtonElement>(
        `button[aria-label="${i18n.global.t('project.environment.terminal.reconnect')}"]`
      )
      ?.click();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(2));
    expect(terminalMock.open).toHaveBeenCalledOnce();
  });

  it('continues a pending connection while hidden without focusing the terminal', async () => {
    let resolveTicket: (value: { ticket: string }) => void = () => {};
    vi.mocked(projectEnvironmentApi.terminalTicket).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveTicket = resolve;
        })
    );
    const props = await mountTerminal();
    await vi.waitFor(() => expect(projectEnvironmentApi.terminalTicket).toHaveBeenCalledOnce());
    props.open = false;
    await nextTick();
    resolveTicket({ ticket: 'late-ticket' });
    await nextTick();
    await vi.waitFor(() => expect(TestWebSocket.instances).toHaveLength(1));
    const socket = socketAt(0);
    socket.open();
    const fitsBeforeReady = terminalMock.fit.mock.calls.length;
    const focusBeforeReady = terminalMock.focus.mock.calls.length;
    socket.message('{"type":"ready"}');
    expect(terminalMock.fit).toHaveBeenCalledTimes(fitsBeforeReady);
    expect(terminalMock.focus).toHaveBeenCalledTimes(focusBeforeReady);
    props.open = true;
    await nextTick();
    await vi.waitFor(() => expect(document.querySelector('[role="dialog"]')).not.toBeNull());
    expect(TestWebSocket.instances).toHaveLength(1);
    expect(document.body.textContent).toContain(
      i18n.global.t('project.environment.terminal.connected')
    );
  });

  it('cancels a pending ticket when the component is unmounted', async () => {
    let resolveTicket: (value: { ticket: string }) => void = () => {};
    vi.mocked(projectEnvironmentApi.terminalTicket).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveTicket = resolve;
        })
    );
    await mountTerminal();
    await vi.waitFor(() => expect(projectEnvironmentApi.terminalTicket).toHaveBeenCalledOnce());
    app?.unmount();
    app = undefined;
    resolveTicket({ ticket: 'late-ticket' });
    await nextTick();
    expect(TestWebSocket.instances).toHaveLength(0);
    expect(terminalMock.dispose).toHaveBeenCalledOnce();
  });
});
