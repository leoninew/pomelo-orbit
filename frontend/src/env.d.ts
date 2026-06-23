/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue';
  const component: DefineComponent<object, object, unknown>;
  export default component;
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string;
  readonly VITE_ENV_LABEL?: string;
  readonly VITE_FEATURE_SSE_DEPLOYMENT_LOG: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
