import type { LabelDpi } from '../schema/label-schema.js';

/** 标签尺寸按物理单位持久化，只在视图或导出边界换算像素。 */
export function millimetersToPixels(value: number, dpi: LabelDpi): number {
  return (value / 25.4) * dpi;
}

export function pixelsToMillimeters(value: number, dpi: LabelDpi): number {
  return (value / dpi) * 25.4;
}
