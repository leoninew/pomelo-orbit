const MAX_EVENT_SIZE = 2 * 1024 * 1024;

export class EventStreamError extends Error {}

export class EventStreamParser {
  private buffer = '';
  private data: string[] = [];
  private eventSize = 0;

  constructor(private readonly onData: (data: string) => void) {}

  feed(text: string) {
    this.buffer += text;
    for (;;) {
      const end = this.buffer.search(/[\r\n]/);
      if (end < 0 || (this.buffer[end] === '\r' && end === this.buffer.length - 1)) {
        break;
      }
      const line = this.buffer.slice(0, end);
      const length = this.buffer[end] === '\r' && this.buffer[end + 1] === '\n' ? 2 : 1;
      this.buffer = this.buffer.slice(end + length);
      if (!line) {
        if (this.data.length) {
          this.onData(this.data.join('\n'));
        }
        this.data = [];
        this.eventSize = 0;
      } else if (line.startsWith('data:')) {
        const value = line.slice(5).replace(/^ /, '');
        this.eventSize += value.length;
        if (this.eventSize > MAX_EVENT_SIZE) {
          throw new EventStreamError('SSE event exceeds size limit');
        }
        this.data.push(value);
      }
    }
    if (this.buffer.length > MAX_EVENT_SIZE) {
      throw new EventStreamError('SSE line exceeds size limit');
    }
  }

  finish() {
    if (this.buffer || this.data.length) {
      throw new EventStreamError('Incomplete SSE event');
    }
  }
}
