import { defineConfig } from 'rolldown';
import { dts } from 'rolldown-plugin-dts';

// 运行时依赖不打进产物（dependencies 已声明）；
// dts 构建同样 external，避免 rolldown-plugin-dts 解析 axios 的 CJS 类型入口（index.d.cts）报 MISSING_EXPORT
const external = ['axios', 'qs', 'lodash-es'];

export default defineConfig([
  {
    // 使用 Rolldown 内置的 TypeScript 与 JSON 支持，避免继续维护旧插件链。
    input: 'src/index.ts',
    external,
    tsconfig: './tsconfig.json',
    output: [
      {
        // 保持原有 CommonJS 产物路径，避免破坏 package.json 中的 main 字段。
        file: 'dist/cjs/index.js',
        format: 'cjs',
        codeSplitting: false,
        exports: 'named',
        minify: false,
      },
      {
        // 保持原有 ESM 产物路径，避免破坏 package.json 中的 module 字段。
        file: 'dist/esm/index.mjs',
        format: 'esm',
        codeSplitting: false,
        exports: 'named',
        minify: false,
      },
    ],
  },
  {
    // 类型声明单独构建，避免 CommonJS 构建重复生成声明文件。
    input: 'src/index.ts',
    external,
    tsconfig: './tsconfig.json',
    output: {
      dir: 'dist/types',
      format: 'esm',
      entryFileNames: (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js'),
      chunkFileNames: (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js'),
    },
    plugins: [
      dts({
        emitDtsOnly: true,
        tsconfig: './tsconfig.json',
      }),
    ],
  },
]);
