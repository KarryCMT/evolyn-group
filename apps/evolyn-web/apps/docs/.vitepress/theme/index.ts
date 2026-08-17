import type { Theme } from 'vitepress';
import DefaultTheme from 'vitepress/theme';
// 引入UI库的样式
import '@evolyn.do/ui/style.css';
import { useGlobalComp } from '../utils/useGlobalComp';
// 自定义样式重载
import './styles/global.css';

// satisfies Theme 让 enhanceApp 的上下文参数（app/router/siteData）获得类型推导
export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    useGlobalComp(app);
  },
} satisfies Theme;
