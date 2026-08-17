import packageJson from '../package.json';

export interface RuntimeConfig {
  // Public API URL prefix injected by the backend while serving index.html.
  // Empty means same-origin /api.
  publicUrl?: string;
  // Release version injected by the backend while serving index.html.
  appVersion?: string;
}

declare global {
  interface Window {
    __CONFIG__?: RuntimeConfig;
  }
}

const apiPathPrefixes = ['/api'];

function normalizePublicUrl(value: string | undefined): string {
  return (value ?? '').trim().replace(/\/+$/, '');
}

// Runtime config wins because Vite env values are baked into the bundle at build time.
export function getPublicUrl(): string {
  if (typeof window !== 'undefined' && window.__CONFIG__ !== undefined) {
    return normalizePublicUrl(window.__CONFIG__.publicUrl);
  }

  return normalizePublicUrl(import.meta.env.VITE_PUBLIC_URL);
}

export function getAppVersion(): string {
  if (typeof window !== 'undefined' && window.__CONFIG__?.appVersion !== undefined) {
    return window.__CONFIG__.appVersion.trim();
  }

  return packageJson.version;
}

export function buildApiUrl(path: string): string {
  assertApiPath(path, apiPathPrefixes);

  return `${getPublicUrl()}${path}`;
}

function assertApiPath(path: string, prefixes: string[]): void {
  const matched = prefixes.some((prefix) => path === prefix || path.startsWith(`${prefix}/`));

  if (!matched) {
    throw new Error(`API path must start with one of: ${prefixes.join(', ')}; got ${path}`);
  }
}

export const config = {
  get publicUrl() {
    return getPublicUrl();
  },
  get appVersion() {
    return getAppVersion();
  },
  envLabel: import.meta.env.VITE_ENV_LABEL,
};

export default config;
