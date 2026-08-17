import path from 'node:path';
import { defineConfig } from 'tsdown';
import Vue from 'unplugin-vue/rolldown';

const srcDir = path.resolve(import.meta.dirname, 'src');

/**
 * 生成与 package.json 发布字段保持一致的输出扩展名。
 * 这里不依赖 tsdown 的默认扩展名，是为了继续兼容原有 dist/esm/*.mjs 与 dist/cjs/*.js 结构。
 */
function resolveOutExtensions({ format }: { format: string }) {
  return {
    js: format === 'cjs' ? '.js' : '.mjs',
    dts: '.d.ts',
  };
}

/**
 * 生成与原 Vite library build 相同的分目录文件名。
 * preserveModules 产物需要按格式落到 esm/cjs 目录，避免破坏现有 exports 指向。
 */
function resolveEntryFileNames({ format }: { format: string }) {
  return `${format === 'cjs' ? 'cjs' : 'esm'}/[name]${format === 'cjs' ? '.js' : '.mjs'}`;
}

export default defineConfig({
  entry: ['src/index.ts'],
  clean: true,
  copy: [{ from: 'src/types/global.d.ts', to: 'dist', flatten: true }],
  css: {
    fileName: 'style.css',
    minify: false,
    splitting: false,
  },
  deps: {
    neverBundle: ['vue'],
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
