import { defineConfig } from 'vitepress';
import { vitepressDemoPlugin } from 'vitepress-demo-plugin';
import path from 'node:path';
import { fileURLToPath } from 'node:url'; // 新增导入

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export const shared = defineConfig({
  // 设置基础路径,用于GitHub Pages部署
  base: '/evolyn/',
  vite: {
    ssr: {
      // gridstack@13 的 dist 内部是无扩展名的 ESM 相对导入，且未提供 exports 映射；
      // VTable 与其 @visactor/* 依赖同时声明了 CommonJS main 和 ESM module，Node 原生
      // SSR 加载会选用 CommonJS 入口，导致 ListTable 具名导入或无扩展名 ESM 相对导入失败。
      // 因此二者整条依赖链都需由 Vite 打入 server bundle 并转换后再执行预渲染。
      noExternal: ['gridstack', /^@visactor\//],
    },
  },
  // 启用最后更新时间
  lastUpdated: true,
  // 生成干净的 URL（去掉.html后缀）
  cleanUrls: true,
  // 将元数据拆分为单独的 chunk
  metaChunk: true,
  // URL重写规则,将zh/开头的路径重写为根路径
  rewrites: {
    'zh/:rest*': ':rest*',
  },
  // 配置HTML头部标签
  head: [['link', { rel: 'icon', href: '/vue3-turbo-component-lib-template/favicon.ico' }]],
  // Markdown配置
  markdown: {
    // 配置Markdown解析器
    config(md) {
      // 使用vitepress-demo-plugin插件,用于展示示例代码
      md.use(vitepressDemoPlugin, {
        demoDir: path.resolve(__dirname, '../../examples'),
      });
    },
  },
});
