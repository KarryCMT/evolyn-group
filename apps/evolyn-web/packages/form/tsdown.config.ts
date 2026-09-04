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

// 构建策略（文档 §3.2 双入口的落地形态）：
// 1) 主构建：全部 TS 入口共用一张图（unbundle 按模块落文件）。rolldown unbundle 按
//    「图内实际引用」生成各模块文件的导出语句——若运行时入口与设计器入口分两组构建，
//    窄图（运行时组带入的 schema/{types,clone,codec} 模块）会覆盖宽图写入的模块文件，
//    丢失「仅被桶文件再导出」的符号（字典工厂、isCanonicalDateTime 等公共 API）。
//    因此 JS/DTS 必须单图一次构建；聚合 style.css（含 Element Plus 主题片）随主构建输出。
// 2) 样式构建：仅以 runtime/style.ts 为入口单独输出 runtime/style.css（不含设计器样式）；
//    该入口不引用任何 schema 模块，不会回写/收窄其他模块产物。
// clean 交给 build 脚本的 rimraf 统一处理，避免第二组构建清掉第一组产物。
const shared = {
  clean: false,
  deps: {
    neverBundle: ['vue'],
  },
  format: ['esm', 'cjs'],
  hash: false,
  // 将设计器空状态插图内联为 data URL：表单包被工作区应用消费时，不依赖 dist/assets
  // 的相对路径，从而避免插图请求落到宿主应用的错误目录。
  loader: {
    '.png': 'base64',
  },
  minify: false,
  outDir: 'dist',
  outExtensions: resolveOutExtensions,
  platform: 'browser' as const,
  plugins: [Vue()],
  root: srcDir,
  target: 'esnext' as const,
  tsconfig: './tsconfig.json',
  unbundle: true,
  outputOptions: (_options: unknown, format: string, context: { cjsDts: boolean }) => ({
    dir: context.cjsDts ? 'dist/types' : 'dist',
    entryFileNames: context.cjsDts
      ? (chunk: { name: string }) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js')
      : resolveEntryFileNames({ format }),
    chunkFileNames: context.cjsDts
      ? (chunk: { name: string }) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js')
      : resolveEntryFileNames({ format }),
    exports: 'named',
    preserveModulesRoot: srcDir,
  }),
};

export default defineConfig([
  {
    ...shared,
    entry: [
      'src/index.ts',
      'src/formula/index.ts',
      'src/schema/index.ts',
      'src/designer/index.ts',
      'src/runtime/index.ts',
      'src/runtime-core/index.ts',
      'src/runtime-web/index.ts',
      'src/runtime-mobile/index.ts',
    ],
    css: {
      fileName: 'style.css',
      minify: false,
      splitting: false,
    },
    dts: {
      vue: true,
      sourcemap: false,
      compilerOptions: {
        declarationMap: false,
      },
    },
  },
  {
    ...shared,
    entry: ['src/runtime/style.ts'],
    css: {
      fileName: 'runtime/style.css',
      minify: false,
      splitting: false,
    },
  },
  {
    ...shared,
    entry: ['src/runtime-core/style.ts'],
    css: {
      fileName: 'runtime-core/style.css',
      minify: false,
      splitting: false,
    },
  },
  {
    ...shared,
    entry: ['src/runtime-web/style.ts'],
    css: {
      fileName: 'runtime-web/style.css',
      minify: false,
      splitting: false,
    },
  },
  {
    ...shared,
    entry: ['src/runtime-mobile/style.ts'],
    css: {
      fileName: 'runtime-mobile/style.css',
      minify: false,
      splitting: false,
    },
  },
]);
