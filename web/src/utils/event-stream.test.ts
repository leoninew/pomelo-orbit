import { describe, expect, it } from 'vitest';
import { EventStreamParser } from './event-stream';

describe('EventStreamParser', () => {
  it('parses split CRLF, comments, multiline data and consecutive events', () => {
    const events: string[] = [];
    const parser = new EventStreamParser((data) => events.push(data));
    for (const text of [
      ': heartbeat\r',
      '\n\r',
      '\ndata: first\r',
      '\ndata: second\r\n\r',
      '\ndata: third\n\n',
    ]) {
      parser.feed(text);
    }
    parser.finish();
    expect(events).toEqual(['first\nsecond', 'third']);
  });

  it('does not commit an incomplete event', () => {
    const events: string[] = [];
    const parser = new EventStreamParser((data) => events.push(data));
    parser.feed('data: {"cursor":"partial"}\n');
    expect(() => parser.finish()).toThrow('Incomplete');
    expect(events).toEqual([]);
  });
});
