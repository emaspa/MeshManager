/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_AMTD_URL?: string;
  readonly VITE_AMTD_TOKEN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
