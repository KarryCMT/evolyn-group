import antfu from '@antfu/eslint-config';

// 代码质量交给 ESLint，排版继续复用工作区根部的 Prettier 配置。
export default antfu({
  vue: true,
  stylistic: false,
});
