import path from 'node:path';
import { defineConfig } from 'tsdown';
import Vue from 'unplugin-vue/rolldown';

const srcDir = path.resolve(import.meta.dirname, 'src');

function resolveOutExtensions({ format }: { format: string }) {
  return {
    js: format === 'cjs' ? '.js' : '.mjs',
    dts: '.d.ts',
  };
}

function resolveEntryFileNames({ format }: { format: string }) {
  return `${format === 'cjs' ? 'cjs' : 'esm'}/[name]${format === 'cjs' ? '.js' : '.mjs'}`;
}

export default defineConfig({
  entry: ['src/index.ts'],
  clean: true,
  css: {
    fileName: 'style.css',
    minify: false,
    splitting: false,
  },
  deps: {
    neverBundle: ['@logicflow/core', 'element-plus', 'vue'],
  },
  dts: {
    vue: true,
    sourcemap: false,
    compilerOptions: {
      declarationMap: false,
    },
  },
  format: ['esm', 'cjs'],
  hash: false,
  minify: false,
  outDir: 'dist',
  outExtensions: resolveOutExtensions,
  platform: 'browser',
  plugins: [Vue()],
  root: srcDir,
  target: 'esnext',
  tsconfig: './tsconfig.json',
  unbundle: true,
  outputOptions: (_options, format, context) => ({
    dir: context.cjsDts ? 'dist/types' : 'dist',
    entryFileNames: context.cjsDts
      ? (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js')
      : resolveEntryFileNames({ format }),
    chunkFileNames: context.cjsDts
      ? (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js')
      : resolveEntryFileNames({ format }),
    exports: 'named',
    preserveModulesRoot: srcDir,
  }),
});
