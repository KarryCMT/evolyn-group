import antfu from '@antfu/eslint-config';

// 格式统一交给 Prettier（apps/evolyn-web/prettier.config.mjs，见 AGENTS.md 约定）：
// 关闭 antfu 的 stylistic 规则（分号/引号/分隔符等，与 Prettier 输出冲突），
// eslint 只负责代码质量类规则；formatters 同理关闭，避免与 Prettier 的
// css/scss/html/md 格式化范围重叠
export default antfu({
  vue: true,
  stylistic: false,
});
