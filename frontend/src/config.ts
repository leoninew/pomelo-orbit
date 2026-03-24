declare global {
	interface Window {
		__CONFIG__?: {
			apiBaseUrl?: string
		}
	}
}

export function getApiBaseUrl(): string {
	return window.__CONFIG__?.apiBaseUrl || import.meta.env.VITE_API_BASE_URL;
}

export const config = {
	get apiBaseUrl() {
		return getApiBaseUrl();
	},
	isDev: import.meta.env.DEV,
	isProd: import.meta.env.PROD,
};

export default config;
