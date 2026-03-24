import { defineStore } from 'pinia';

export const useStorageStore = defineStore('storage', () => {
	function getItem<T>(key: string): T | null {
		const value = localStorage.getItem(key);
		return value ? JSON.parse(value) : null;
	}

	function setItem<T>(key: string, value: T): void {
		localStorage.setItem(key, JSON.stringify(value));
	}

	function removeItem(key: string): void {
		localStorage.removeItem(key);
	}

	return { getItem, setItem, removeItem };
});
