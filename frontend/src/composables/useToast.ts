import { ref } from 'vue';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  id: number;
  type: ToastType;
  text: string;
  duration: number;
}

// Global singleton state shared across all composable instances
const toasts = ref<Toast[]>([]);

export function useToast() {
  function show(type: ToastType, text: string, duration = 3000) {
    const id = Date.now() + Math.random();
    toasts.value.push({ id, type, text, duration });
  }

  function remove(id: number) {
    toasts.value = toasts.value.filter((toast) => toast.id !== id);
  }

  return {
    toasts,
    remove,
    success: (text: string, duration?: number) => show('success', text, duration),
    error: (text: string, duration?: number) => show('error', text, duration),
    warning: (text: string, duration?: number) => show('warning', text, duration),
    info: (text: string, duration?: number) => show('info', text, duration),
  };
}
