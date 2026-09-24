// 移动 Surface 自身样式入口。Vant 是宿主 peer，由应用统一加载其主题 CSS，避免组件包
// 重复打包第三方全量样式并阻断宿主的主题裁剪能力。
import '../runtime/style';
