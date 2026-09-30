// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { readLogStream } from './log';
import type { LogStreamEvent } from '@/gen/proto/orbit/v1/common/log_stream';

vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ token: 'test-token' }) }));
const { unauthorized } = vi.hoisted(() => ({ unauthorized: vi.fn() }));
vi.mock('@/utils/handle-unauthorized', () => ({ handleUnauthorized: unauthorized }));
afterEach(() => {
  vi.unstubAllGlobals();
  unauthorized.mockReset();
});

describe('readLogStream', () => {
  it('uses Bearer auth and decodes arbitrary network byte boundaries', async () => {
    const raw = new TextEncoder().encode(
      ': heartbeat\r\n\r\ndata: {"type":"ready","source_id":"source","message":"中文"}\r\n\r\ndata: {"type":"complete"}\n\n'
    );
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        for (const byte of raw) {
          controller.enqueue(new Uint8Array([byte]));
        }
        controller.close();
      },
    });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(body, { headers: { 'Content-Type': 'text/event-stream' } }));
    vi.stubGlobal('fetch', fetchMock);
    const events: LogStreamEvent[] = [];
    await readLogStream(
      { projectId: 'project-1', path: '/log', kind: 'file' },
      'cursor',
      new AbortController().signal,
      (event) => events.push(event)
    );
    expect(events.map((event) => event.type)).toEqual(['ready', 'complete']);
    expect(events[0]?.message).toBe('中文');
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('cursor=cursor'),
      expect.objectContaining({
        headers: { Accept: 'text/event-stream', Authorization: 'Bearer test-token' },
      })
    );
  });

  it('normalizes HTTP authorization errors before streaming', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ code: 'unauthorized', error: 'Expired', requestId: 'request-1' }),
            { status: 401, headers: { 'Content-Type': 'application/json' } }
          )
        )
    );
    await expect(
      readLogStream(
        { projectId: 'project-1', path: '/log', kind: 'file' },
        '',
        new AbortController().signal,
        vi.fn()
      )
    ).rejects.toMatchObject({ status: 401, code: 'unauthorized', requestId: 'request-1' });
    expect(unauthorized).toHaveBeenCalledOnce();
  });
});
