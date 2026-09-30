export const MAX_LOG_LINES = 50_000;
export const MAX_LOG_BYTES = 10 * 1024 * 1024;

export interface LogPatch {
  append: string;
  removedChars: number;
  removedLines: number;
}

export class LogBuffer {
  text = '';
  bytes = 0;
  lines = 1;
  removedLines = 0;

  constructor(
    private readonly maxBytes = MAX_LOG_BYTES,
    private readonly maxLines = MAX_LOG_LINES
  ) {}

  clear() {
    this.text = '';
    this.bytes = 0;
    this.lines = 1;
    this.removedLines = 0;
  }

  append(text: string): LogPatch {
    const encoder = new TextEncoder();
    this.text += text;
    this.bytes += encoder.encode(text).byteLength;
    this.lines += (text.match(/\n/g) ?? []).length;
    let removedChars = 0;
    let removedLines = 0;
    while (this.bytes > this.maxBytes || this.lines > this.maxLines) {
      const newline = this.text.indexOf('\n', removedChars);
      if (newline < 0) {
        const remaining = this.text.slice(removedChars);
        const bytes = encoder.encode(remaining);
        const decoder = new TextDecoder('utf-8', { fatal: true });
        let start = bytes.length - this.maxBytes;
        while (start < bytes.length && ((bytes[start] ?? 0) & 0xc0) === 0x80) {
          start++;
        }
        const suffix = decoder.decode(bytes.subarray(start));
        removedChars += remaining.length - suffix.length;
        this.bytes = bytes.length - start;
        break;
      }
      const line = this.text.slice(removedChars, newline + 1);
      this.bytes -= encoder.encode(line).byteLength;
      this.lines--;
      removedLines++;
      removedChars = newline + 1;
    }
    this.text = this.text.slice(removedChars);
    this.removedLines += removedLines;
    return { append: text, removedChars, removedLines };
  }
}

// Count matching timestamped records, so identical real records survive overlap replay.
export class LogReplayWindow {
  private counts = new Map<string, number>();
  private replay = new Map<string, number>();
  private baseline = new Map<string, number>();
  private bytes = 0;
  truncated = false;

  reconnect() {
    this.baseline = new Map(this.counts);
    this.replay.clear();
  }

  accept(source: string, line: string) {
    const key = `${source}:${line}`;
    const count = (this.replay.get(key) ?? 0) + 1;
    this.replay.set(key, count);
    if (count <= (this.baseline.get(key) ?? 0)) {
      return false;
    }
    if (!this.counts.has(key)) {
      this.bytes += key.length * 2;
    }
    this.counts.set(key, (this.counts.get(key) ?? 0) + 1);
    while (this.counts.size > 1024 || this.bytes > 1024 * 1024) {
      const first = this.counts.keys().next().value as string;
      this.counts.delete(first);
      this.baseline.delete(first);
      this.replay.delete(first);
      this.bytes -= first.length * 2;
      this.truncated = true;
    }
    return true;
  }
}
