// 移动 Surface 自身样式入口。Vant 是宿主 peer，由应用统一加载其主题 CSS，避免组件包
// 重复打包第三方全量样式并阻断宿主的主题裁剪能力。
import '../runtime/style';

// 独立样式构建不会经过 runtime-mobile/index.ts，因此需要显式带入所有公开 Surface。
// Surface 会继续引用多标签渲染器与字段组件，构建器据此抽取完整的 scoped CSS。
import './FormMobileActionBar.vue';
import './FormMobileRuntimeSurface.vue';
