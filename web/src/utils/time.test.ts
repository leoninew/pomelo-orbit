import { afterEach, describe, expect, it, vi } from 'vitest';
import { delayAsync } from './time';

afterEach(() => vi.useRealTimers());

describe('delayAsync', () => {
  it('resolves after the configured delay', async () => {
    vi.useFakeTimers();
    const resolved = vi.fn();
    const delayed = delayAsync(1000);
    void delayed.then(resolved);

    await vi.advanceTimersByTimeAsync(999);
    expect(resolved).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1);
    await expect(delayed).resolves.toBeUndefined();
    expect(resolved).toHaveBeenCalledOnce();
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
  });
});
