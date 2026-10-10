<template>
  <AppDrawer
    :open="open"
    :title="t('project.environment.terminal.title')"
    :width-class="fullscreen ? 'w-screen' : 'w-[min(960px,100vw)]'"
    header-class="flex shrink-0 items-center justify-between gap-3 border-b border-border px-6 py-3"
    body-class="min-h-0 flex-1 overflow-hidden p-0"
    @update:open="emit('update:open', $event)"
  >
    <template #title>
      <span class="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-1">
        <span>{{ t('project.environment.terminal.title') }}</span>
        <span class="min-w-0 break-all text-sm font-normal text-muted-foreground">
          {{ t(`project.environment.targetTypes.${targetType}`) }}
          <span v-if="identity">· {{ identity }}</span>
        </span>
      </span>
    </template>
    <template #actions>
      <button
        type="button"
        class="app-icon-button"
        :title="connectionControlLabel"
        :aria-label="
          t(`project.environment.terminal.${connectionActive ? 'disconnect' : 'reconnect'}`)
        "
        @click="connectionActive ? disconnect() : connect()"
      >
        <LoaderCircle
          v-if="status === 'connecting'"
          class="size-4 animate-spin text-primary"
          aria-hidden="true"
        />
        <Plug
          v-else-if="status === 'connected'"
          class="size-4 text-green-600 dark:text-green-400"
          aria-hidden="true"
        />
        <Unplug
          v-else-if="status === 'disconnected'"
          class="size-4 text-muted-foreground"
          aria-hidden="true"
        />
        <CircleAlert v-else class="size-4 text-destructive" aria-hidden="true" />
      </button>
      <span class="sr-only" role="status">
        {{ t(`project.environment.terminal.${status}`) }}
      </span>
      <button
        type="button"
        class="app-icon-button"
        :title="fullscreenControlLabel"
        :aria-label="fullscreenControlLabel"
        :aria-pressed="fullscreen"
        @click="fullscreen = !fullscreen"
      >
        <Minimize v-if="fullscreen" class="size-4" aria-hidden="true" />
        <Maximize v-else class="size-4" aria-hidden="true" />
      </button>
    </template>
    <div class="flex h-full min-h-0 flex-col gap-3 p-6">
      <p
        v-if="detail"
        class="shrink-0 break-words text-sm"
        :class="status === 'failed' ? 'text-destructive' : 'text-muted-foreground'"
        role="status"
      >
        {{ detail }}
      </p>
      <div
        ref="terminalElement"
        class="min-h-0 flex-1 overflow-hidden rounded-md border border-border/50 bg-white p-2 dark:bg-[#1e1e1e]"
        :aria-label="t('project.environment.terminal.title')"
      />
    </div>
  </AppDrawer>
</template>

<script setup lang="ts">
  import '@xterm/xterm/css/xterm.css';
  import { FitAddon } from '@xterm/addon-fit';
  import { Terminal } from '@xterm/xterm';
  import { CircleAlert, LoaderCircle, Maximize, Minimize, Plug, Unplug } from '@lucide/vue';
  import { computed, onBeforeUnmount, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { projectEnvironmentApi } from '@/api/project/environment';
  import AppDrawer from '@/components/AppDrawer.vue';
  import { useTheme } from '@/composables/useTheme';
  import { buildApiUrl } from '@/config';

  const props = defineProps<{
    open: boolean;
    projectId: string;
    targetType: string;
    host: string;
    username: string;
  }>();
  const emit = defineEmits<{ 'update:open': [value: boolean] }>();
  const { t } = useI18n();
  const fullscreen = ref(false);
  const fullscreenControlLabel = computed(() =>
    t(`project.environment.terminal.${fullscreen.value ? 'exitFullscreen' : 'fullscreen'}`)
  );
  const identity = computed(() => [props.username, props.host].filter(Boolean).join('@'));
  const { effectiveTheme } = useTheme();
  const terminalTheme = computed(() =>
    effectiveTheme.value === 'dark'
      ? {
          background: '#1e1e1e',
          foreground: '#d4d4d4',
          cursor: '#d4d4d4',
          selectionBackground: '#264f78',
        }
      : {
          background: '#ffffff',
          foreground: '#333333',
          cursor: '#333333',
          selectionBackground: '#add6ff',
          black: '#000000',
          red: '#cd3131',
          green: '#008000',
          yellow: '#795e26',
          blue: '#0451a5',
          magenta: '#af00db',
          cyan: '#267f99',
          white: '#555555',
          brightBlack: '#666666',
          brightRed: '#e51400',
          brightGreen: '#198a19',
          brightYellow: '#b89500',
          brightBlue: '#0067c0',
          brightMagenta: '#af00db',
          brightCyan: '#168aad',
          brightWhite: '#333333',
        }
  );
  const terminalElement = ref<HTMLElement>();
  const status = ref<'connecting' | 'connected' | 'disconnected' | 'failed'>('connecting');
  const connectionActive = computed(
    () => status.value === 'connected' || status.value === 'connecting'
  );
  const connectionControlLabel = computed(
    () =>
      `${t(`project.environment.terminal.${status.value}`)} - ${t(
        `project.environment.terminal.${connectionActive.value ? 'disconnect' : 'reconnect'}`
      )}`
  );
  const detail = ref('');
  let socket: WebSocket | undefined;
  let terminal: Terminal | undefined;
  let terminalContainer: HTMLDivElement | undefined;
  let fitAddon: FitAddon | undefined;
  let observer: ResizeObserver | undefined;
  let connectionTimer: ReturnType<typeof setTimeout> | undefined;
  let generation = 0;

  function clearConnectionTimer() {
    clearTimeout(connectionTimer);
    connectionTimer = undefined;
  }

  function disconnect() {
    clearConnectionTimer();
    generation++;
    socket?.close();
    socket = undefined;
    status.value = 'disconnected';
    detail.value = '';
  }

  function dispose() {
    disconnect();
    observer?.disconnect();
    observer = undefined;
    terminal?.dispose();
    terminal = undefined;
    fitAddon = undefined;
    terminalContainer?.remove();
    terminalContainer = undefined;
  }

  function fit() {
    if (!props.open || !terminal || !fitAddon || !terminalElement.value) {
      return;
    }
    fitAddon.fit();
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }));
    }
  }

  async function connect() {
    if (!props.open || !props.projectId || !terminalElement.value) {
      return;
    }
    disconnect();
    const current = generation;
    status.value = 'connecting';
    detail.value = '';
    if (!terminal) {
      terminal = new Terminal({
        cursorBlink: true,
        fontFamily: 'Consolas, "Cascadia Code", monospace',
        fontSize: 12,
        theme: terminalTheme.value,
      });
      fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      // Reka removes drawer content on close; retain xterm's DOM with its session.
      terminalContainer = document.createElement('div');
      terminalContainer.className = 'h-full w-full';
      terminalElement.value.append(terminalContainer);
      terminal.open(terminalContainer);
      terminal.onData((value) => {
        if (!props.open || status.value !== 'connected' || socket?.readyState !== WebSocket.OPEN) {
          return;
        }
        const data = new TextEncoder().encode(value);
        for (let offset = 0; offset < data.length; offset += 16 * 1024) {
          socket.send(data.subarray(offset, offset + 16 * 1024));
        }
      });
      observer = new ResizeObserver(fit);
      observer.observe(terminalElement.value);
    }
    terminal.reset();
    fit();
    try {
      const { ticket } = await projectEnvironmentApi.terminalTicket(props.projectId);
      if (current !== generation) {
        return;
      }
      const url = new URL(buildApiUrl('/api/environment/terminal'), window.location.href);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      url.searchParams.set('project_id', props.projectId);
      const connection = new WebSocket(url, ['orbit-terminal-v1', `ticket.${ticket}`]);
      socket = connection;
      connection.binaryType = 'arraybuffer';
      connectionTimer = setTimeout(() => {
        if (current !== generation) {
          return;
        }
        disconnect();
        status.value = 'failed';
        detail.value = t('project.environment.terminal.connectionTimedOut');
      }, 20_000);
      connection.onopen = () => {
        if (current !== generation) {
          return;
        }
        fit();
      };
      connection.onmessage = (event: MessageEvent<string | ArrayBuffer>) => {
        if (current !== generation) {
          return;
        }
        if (event.data instanceof ArrayBuffer) {
          terminal?.write(new Uint8Array(event.data));
          return;
        }
        try {
          const control = JSON.parse(event.data) as { type: string; code?: number };
          if (control.type === 'ready') {
            clearConnectionTimer();
            status.value = 'connected';
            if (props.open) {
              terminal?.focus();
            }
          } else if (control.type === 'exit') {
            detail.value = t('project.environment.terminal.exited', { code: control.code ?? 0 });
          } else if (control.type === 'connection_failed') {
            status.value = 'failed';
            detail.value = t('project.environment.terminal.connectionFailed');
          } else if (
            control.type === 'idle_timeout' ||
            control.type === 'access_or_target_changed' ||
            control.type === 'session_limit'
          ) {
            detail.value = t(`project.environment.terminal.${control.type}`);
          }
        } catch {
          status.value = 'failed';
          detail.value = t('project.environment.terminal.connectionFailed');
        }
      };
      connection.onerror = () => {
        if (current === generation) {
          clearConnectionTimer();
        }
        if (current === generation && !detail.value) {
          status.value = 'failed';
          detail.value = t('project.environment.terminal.connectionFailed');
        }
      };
      connection.onclose = () => {
        if (current === generation) {
          clearConnectionTimer();
          socket = undefined;
          if (status.value !== 'failed') {
            status.value = 'disconnected';
          }
        }
      };
    } catch (error) {
      if (current === generation) {
        status.value = 'failed';
        detail.value =
          error instanceof Error
            ? error.message
            : t('project.environment.terminal.connectionFailed');
      }
    }
  }

  watch(terminalTheme, (theme) => {
    if (terminal) {
      terminal.options.theme = theme;
    }
  });

  watch(
    () => [props.projectId, props.open, terminalElement.value] as const,
    ([projectId, open, element], previous) => {
      if (previous && projectId !== previous[0]) {
        dispose();
      }
      observer?.disconnect();
      if (!open || !element) {
        terminal?.blur();
        if (!element) {
          terminalContainer?.remove();
        }
        return;
      }
      if (!terminal) {
        void connect();
        return;
      }
      if (terminalContainer) {
        element.append(terminalContainer);
      }
      observer?.observe(element);
      fit();
      terminal.focus();
    },
    { immediate: true, flush: 'post' }
  );
  onBeforeUnmount(dispose);
</script>
