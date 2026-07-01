export interface RuntimeConfig {
  // Public API prefix injected by deployment tooling.
  // Empty or "/" means same-origin /api; use a full origin only for cross-origin APIs.
  apiBaseUrl?: string;
}

declare global {
  interface Window {
    __CONFIG__?: RuntimeConfig;
  }
}

function normalizeApiBaseUrl(value: string | undefined): string {
  const trimmed = value?.trim() ?? '';
  if (trimmed === '' || trimmed === '/') {
    return '';
  }
  return trimmed.replace(/\/+$/, '');
}

// Runtime config wins because Vite env values are baked into the bundle at build time.
export function getApiBaseUrl(): string {
  return normalizeApiBaseUrl(window.__CONFIG__?.apiBaseUrl || import.meta.env.VITE_API_BASE_URL);
}

export function buildApiUrl(path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${getApiBaseUrl()}${normalizedPath}`;
}

export const config = {
  get apiBaseUrl() {
    return getApiBaseUrl();
  },
  envLabel: import.meta.env.VITE_ENV_LABEL,
};

export default config;
