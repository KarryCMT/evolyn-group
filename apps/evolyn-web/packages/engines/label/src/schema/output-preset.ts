import type { LabelOutputPreset, LabelSchema } from './label-schema.js';

export function labelOutputPresets(schema: LabelSchema): LabelOutputPreset[] {
  return schema.settings.outputPresets?.length
    ? schema.settings.outputPresets.map((preset) => ({ ...preset }))
    : [{
        id: 'original',
        name: '原始尺寸',
        width: schema.page.width,
        height: schema.page.height,
        unit: schema.page.unit,
        dpi: schema.page.dpi,
      }];
}

export function labelPresetPixelSize(preset: LabelOutputPreset): { width: number; height: number } {
  if (preset.unit === 'px') return { width: Math.round(preset.width), height: Math.round(preset.height) };
  return {
    width: Math.round((preset.width / 25.4) * preset.dpi),
    height: Math.round((preset.height / 25.4) * preset.dpi),
  };
}

export function withLabelOutputPreset(schema: LabelSchema, presetId: string): LabelSchema {
  const preset = labelOutputPresets(schema).find((item) => item.id === presetId);
  if (!preset) throw new Error(`unknown label output preset: ${presetId}`);
  const scale = preset.width / schema.page.width;
  const clone = structuredClone(schema);
  clone.page = { ...clone.page, width: preset.width, height: preset.height, unit: preset.unit, dpi: preset.dpi };
  clone.elements = clone.elements.map((element) => {
    const scaled = {
      ...element,
      x: element.x * scale,
      y: element.y * scale,
      width: element.width * scale,
      height: element.height * scale,
    };
    if ('strokeWidth' in scaled && typeof scaled.strokeWidth === 'number') scaled.strokeWidth *= scale;
    if ('radius' in scaled && typeof scaled.radius === 'number') scaled.radius *= scale;
    if ('style' in scaled && scaled.style) scaled.style = { ...scaled.style, fontSize: scaled.style.fontSize * scale };
    return scaled;
  });
  return clone;
}
