import path from 'node:path';
import Vue from '@vitejs/plugin-vue';

import { ElementPlusResolver } from 'unplugin-vue-components/resolvers';
import Components from 'unplugin-vue-components/vite';

import { defineConfig } from 'vite';

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '~/': `${path.resolve(__dirname, 'src')}/`,
    },
  },

  css: {
    preprocessorOptions: {
      scss: {
        additionalData: `@use "~/styles/element/index.scss" as *;`,
        // rolldown-vite 仅保留 modern Sass API，无需再声明 api: 'modern-compiler'
      },
    },
  },

  server: {
    // 前端开发服务器统一使用 11000 端口，避免占用 Vite 默认端口。
    port: 11000,
    proxy: {
      // 开发期将 /api 转发到本地 evolyn-core（默认 8080）；生产由网关同源托管
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },

  plugins: [
    Vue(),

    Components({
      // allow auto load markdown components under `./src/components/`
      extensions: ['vue', 'md'],
      // allow auto import and register components used in markdown
      include: [/\.vue$/, /\.vue\?vue/, /\.md$/],
      resolvers: [
        ElementPlusResolver({
          importStyle: 'sass',
        }),
      ],
      dts: 'src/components.d.ts',
    }),
  ],
});
