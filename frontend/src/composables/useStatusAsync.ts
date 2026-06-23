import { ref } from 'vue';

export type AsyncStatus = 'idle' | 'loading' | 'loaded' | 'error';

export function useStatusAsync() {
  const status = ref<AsyncStatus>('idle');
  const error = ref<string>();

  const loading = ref(false);

  async function execute<T>(fn: () => Promise<T>): Promise<T | undefined> {
    status.value = 'loading';
    loading.value = true;
    error.value = undefined;
    try {
      const result = await fn();
      status.value = 'loaded';
      return result;
    } catch (e) {
      status.value = 'error';
      error.value = e instanceof Error ? e.message : 'Request failed';
      console.error('Request failed:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { status, loading, error, execute };
}
