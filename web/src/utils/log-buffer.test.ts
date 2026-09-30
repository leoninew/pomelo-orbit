import { describe, expect, it } from 'vitest';
import { LogBuffer, LogReplayWindow } from './log-buffer';

describe('LogBuffer', () => {
  it('trims old lines while preserving the bounded suffix', () => {
    const buffer = new LogBuffer(100, 3);
    buffer.append('one\ntwo\n');
    const patch = buffer.append('three\nfour\n');
    expect(buffer.text).toBe('three\nfour\n');
    expect(patch.removedLines).toBe(2);
    expect(buffer.lines).toBe(3);
  });

  it('bounds UTF-8 bytes even for one very long line', () => {
    const buffer = new LogBuffer(10, 100);
    buffer.append('中文中文中文');
    expect(buffer.text).toBe('文中文');
    expect(buffer.bytes).toBe(9);
    expect(buffer.text).not.toContain('\uFFFD');
  });
});

describe('LogReplayWindow', () => {
  it('skips overlap occurrences and preserves additional identical records', () => {
    const replay = new LogReplayWindow();
    expect(replay.accept('source', '2026-09-30T00:00:00Z same\n')).toBe(true);
    expect(replay.accept('source', '2026-09-30T00:00:00Z same\n')).toBe(true);
    replay.reconnect();
    expect(replay.accept('source', '2026-09-30T00:00:00Z same\n')).toBe(false);
    expect(replay.accept('source', '2026-09-30T00:00:00Z same\n')).toBe(false);
    expect(replay.accept('source', '2026-09-30T00:00:00Z same\n')).toBe(true);
    expect(replay.accept('rebuilt', '2026-09-30T00:00:00Z same\n')).toBe(true);
  });
});
