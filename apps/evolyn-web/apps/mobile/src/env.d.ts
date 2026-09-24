/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_GLOB_APP_TITLE?: string;
  readonly VITE_GLOB_API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
