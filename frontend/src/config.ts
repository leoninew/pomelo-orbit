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
	envLabel: import.meta.env.VITE_ENV_LABEL,
	features: {
		sseDeploymentLog: import.meta.env.VITE_FEATURE_SSE_DEPLOYMENT_LOG === 'true',
	},
};

export default config;
