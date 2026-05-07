import { ref } from 'vue';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

interface Toast {
	id: number
	type: ToastType
	text: string
}

// Global singleton state shared across all composable instances
const toasts = ref<Toast[]>([]);

export function useToast() {
	function show(type: ToastType, text: string, duration = 3000) {
		const id = Date.now() + Math.random();
		toasts.value.push({ id, type, text });
		setTimeout(() => {
			toasts.value = toasts.value.filter((t) => t.id !== id);
		}, duration);
	}

	return {
		toasts,
		success: (text: string) => show('success', text),
		error: (text: string) => show('error', text),
		warning: (text: string) => show('warning', text),
		info: (text: string) => show('info', text),
	};
}
