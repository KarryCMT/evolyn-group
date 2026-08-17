import { rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { type OutputOptions, type RolldownOptions, rolldown } from 'rolldown';
import { dts } from 'rolldown-plugin-dts';
import Vue from 'unplugin-vue/rolldown';

type PackageName = 'hooks' | 'directives' | 'utils';
type BundleFormat = 'esm' | 'cjs';

const buildDir = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(buildDir, '..');
const aggregateDistDir = path.resolve(rootDir, 'dist');

/**
 * 解析项目内路径，避免 Gulp 的工作目录变化时把聚合产物写到错误位置。
 */
function resolveRoot(...segments: string[]) {
  return path.resolve(rootDir, ...segments);
}

/**
 * 为普通工具包生成统一的 Rolldown 配置，保持与独立包构建相同的产物结构。
 * 聚合构建只改变输出目录，避免维护两套不同的源码打包规则。
 */
function createPackageConfig(pkg: PackageName): RolldownOptions[] {
  const packageDir = resolveRoot('packages', pkg);
  const input = path.join(packageDir, 'src/index.ts');
  const external = pkg === 'utils' ? [] : ['vue'];

  return [
    {
      input,
      cwd: packageDir,
      external,
      tsconfig: './tsconfig.json',
      output: [createJsOutput(pkg, 'cjs'), createJsOutput(pkg, 'esm')],
    },
    {
      input,
      cwd: packageDir,
      external,
      tsconfig: './tsconfig.json',
      output: {
        dir: resolveRoot('dist', pkg, 'types'),
        format: 'esm',
        entryFileNames: (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js'),
        chunkFileNames: (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js'),
      },
      plugins: [
        dts({
          cwd: packageDir,
          emitDtsOnly: true,
          tsconfig: path.join(packageDir, 'tsconfig.json'),
        }),
      ],
    },
  ];
}

/**
 * 生成 JS 输出配置时固定 cjs/esm 目录，是为了兼容旧 build:gulp 的聚合产物路径。
 */
function createJsOutput(pkg: PackageName, format: BundleFormat): OutputOptions {
  return {
    file: resolveRoot('dist', pkg, format, format === 'cjs' ? 'index.js' : 'index.mjs'),
    format,
    codeSplitting: false,
    exports: 'named',
    minify: false,
  };
}

/**
 * 执行单个 Rolldown 配置，并主动关闭 bundle，避免聚合任务长时间占用文件句柄。
 */
async function writeRolldownConfig(config: RolldownOptions) {
  const outputOptions = Array.isArray(config.output) ? config.output : [config.output];
  const bundle = await rolldown(config);

  try {
    for (const output of outputOptions) {
      if (output) {
        await bundle.write(output);
      }
    }
  } finally {
    await bundle.close();
  }
}

/**
 * 构建普通工具包，复用 Rolldown 的 JS API 可以让 Gulp 继续只负责聚合编排。
 */
async function bundlePackage(pkg: PackageName) {
  const configs = createPackageConfig(pkg);

  for (const config of configs) {
    await writeRolldownConfig(config);
  }
}

/**
 * 使用 tsdown 构建 UI 聚合产物，保持用户可以同时选择独立包构建或根目录聚合构建。
 */
export async function ui() {
  const { build: buildWithTsdown } = await import('tsdown');
  const uiDir = resolveRoot('packages', 'ui');
  const uiSrcDir = path.resolve(uiDir, 'src');

  await buildWithTsdown({
    config: false,
    name: 'ui',
    cwd: uiDir,
    entry: ['src/index.ts'],
    clean: false,
    copy: [{ from: 'src/types/global.d.ts', to: resolveRoot('dist', 'ui'), flatten: true }],
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
    outDir: resolveRoot('dist', 'ui'),
    outExtensions: ({ format }) => ({
      js: format === 'cjs' ? '.js' : '.mjs',
      dts: '.d.ts',
    }),
    platform: 'browser',
    plugins: [Vue()],
    root: uiSrcDir,
    target: 'esnext',
    tsconfig: './tsconfig.json',
    unbundle: true,
    outputOptions: (_options, format, context) => ({
      dir: context.cjsDts ? resolveRoot('dist', 'ui', 'types') : resolveRoot('dist', 'ui'),
      entryFileNames: context.cjsDts
        ? (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js')
        : `${format === 'cjs' ? 'cjs' : 'esm'}/[name]${format === 'cjs' ? '.js' : '.mjs'}`,
      chunkFileNames: context.cjsDts
        ? (chunk) => (chunk.name.endsWith('.d') ? '[name].ts' : '[name].js')
        : `${format === 'cjs' ? 'cjs' : 'esm'}/[name]${format === 'cjs' ? '.js' : '.mjs'}`,
      exports: 'named',
      preserveModulesRoot: uiSrcDir,
    }),
  });
}

/**
 * 清理根目录聚合产物，避免旧文件残留影响使用者判断实际构建结果。
 */
export async function clean() {
  await rm(aggregateDistDir, { recursive: true, force: true });
}

/**
 * 构建 hooks 聚合产物，单独导出任务是为了仍然支持 gulp hooks 这种局部调试方式。
 */
export async function hooks() {
  await bundlePackage('hooks');
}

/**
 * 构建 directives 聚合产物，单独导出任务是为了仍然支持 gulp directives 这种局部调试方式。
 */
export async function directives() {
  await bundlePackage('directives');
}

/**
 * 构建 utils 聚合产物，单独导出任务是为了仍然支持 gulp utils 这种局部调试方式。
 */
export async function utils() {
  await bundlePackage('utils');
}

/**
 * 默认任务保持旧 build:gulp 的完整聚合语义：先清理根 dist，再并行构建全部库包。
 */
export default async function buildAll() {
  await clean();
  await Promise.all([hooks(), directives(), utils(), ui()]);
}
