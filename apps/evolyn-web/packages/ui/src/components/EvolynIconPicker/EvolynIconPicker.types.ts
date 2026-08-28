import type { Component } from 'vue';

/** 系统图标选项：name 应为调用方可持久化的稳定键。 */
export interface EvolynIconPickerSystemIcon {
  name: string;
  label: string;
  icon: Component;
}

/** 应用图标背景选项。background 仅保存逗号分隔的两个纯色值。 */
export interface EvolynIconPickerColorOption {
  label: string;
  background: string;
}

/** 可直接写入 API `icon` 字段的可持久化图标值。 */
export type EvolynIconPickerValue =
  | { type: 'remix'; name: string; background: string }
  | { type: 'custom'; name: string };

export interface EvolynIconPickerProps {
  /** 可选的系统图标；默认提供常用的 Remix Fill 图标。 */
  systemIcons?: EvolynIconPickerSystemIcon[];
  /** 图标卡片可选背景；默认提供六组产品内置渐变。 */
  colors?: EvolynIconPickerColorOption[];
  /** 裁剪输出边长，默认 200px。 */
  outputSize?: number;
  /** 允许选择的自定义图片最大体积，默认 20MB。 */
  maxFileSize?: number;
}

export interface EvolynIconPickerEmits {
  change: [value: EvolynIconPickerValue];
  /** 裁剪完成后的文件；调用方上传成功后，将返回地址作为 custom.name 回填 v-model。 */
  upload: [file: File];
}
