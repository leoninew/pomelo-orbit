import { describe, expect, it, vi } from 'vitest';
import { delayAsync } from './time';

describe('delayAsync', () => {
  it('resolves after the configured delay', async () => {
    vi.useFakeTimers();
    const delayed = delayAsync(1000);

    await vi.advanceTimersByTimeAsync(999);
    await expect(
      Promise.race([delayed.then(() => 'resolved'), Promise.resolve('pending')])
    ).resolves.toBe('pending');

    await vi.advanceTimersByTimeAsync(1);
    await expect(delayed).resolves.toBeUndefined();
    vi.useRealTimers();
  });

  it('resolves immediately when the signal is aborted', async () => {
    const controller = new AbortController();
    controller.abort();

    await expect(delayAsync(1000, controller.signal)).resolves.toBeUndefined();
  });

  it('resolves without waiting after abort', async () => {
    vi.useFakeTimers();
    const controller = new AbortController();
    const delayed = delayAsync(1000, controller.signal);

    controller.abort();
    await expect(delayed).resolves.toBeUndefined();
    await vi.advanceTimersByTimeAsync(1000);
    vi.useRealTimers();
  });
});
