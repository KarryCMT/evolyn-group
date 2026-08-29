import path from 'node:path';
import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: [
      {
        find: /^@evolyn\.do\/form\/runtime\/style\.css$/,
        replacement: path.resolve(__dirname, '../../packages/form/src/runtime/styles/index.scss'),
      },
      {
        find: /^@evolyn\.do\/form\/runtime$/,
        replacement: path.resolve(__dirname, '../../packages/form/src/runtime/index.ts'),
      },
      {
        find: /^@evolyn\.do\/form\/schema$/,
        replacement: path.resolve(__dirname, '../../packages/form/src/schema/index.ts'),
      },
      { find: '~/', replacement: `${path.resolve(__dirname, 'src')}/` },
    ],
  },
  test: {
    environment: 'happy-dom',
    globals: true,
    include: ['src/**/__tests__/**/*.spec.ts'],
    clearMocks: true,
  },
});
