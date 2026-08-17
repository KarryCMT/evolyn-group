import { defineConfig } from 'rolldown';
import { dts } from 'rolldown-plugin-dts';

export default defineConfig([
  {
    // Vue 是运行时 peer 依赖，外置可以避免把宿主项目的 Vue 打进 directives 包。
    input: 'src/index.ts',
    external: ['vue'],
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
      // 如果后续需要压缩版本，优先使用 Rolldown 内置 minify，避免重新引入 terser。
      // {
      //   file: 'dist/index.min.js',
      //   format: 'esm',
      //   codeSplitting: false,
      //   minify: true,
      // },
    ],
  },
  {
    // 类型声明单独构建，避免 CommonJS 构建重复生成声明文件。
    input: 'src/index.ts',
    external: ['vue'],
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
