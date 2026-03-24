import { ref } from 'vue';

export function useStatusAsync() {
	const loading = ref(false);
	const error = ref<string>();

	async function execute<T>(fn: () => Promise<T>): Promise<T | undefined> {
		loading.value = true;
		error.value = undefined;
		try {
			return await fn();
		} catch (e) {
			error.value = e instanceof Error ? e.message : 'Request failed';
			console.error('Request failed:', e);
			throw e;
		} finally {
			loading.value = false;
		}
	}

	return { loading, error, execute };
}
