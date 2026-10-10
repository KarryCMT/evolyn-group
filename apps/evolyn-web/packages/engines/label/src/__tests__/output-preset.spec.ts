import { describe, expect, it } from 'vitest';

import type { LabelSchema } from '../schema/label-schema.js';
import { labelOutputPresets, labelPresetPixelSize, withLabelOutputPreset } from '../schema/output-preset.js';
import { validateLabelSchema } from '../schema/validator.js';

const schema: LabelSchema = {
  schemaVersion: '1.0',
  name: '标签',
  page: { width: 90, height: 60, unit: 'mm', dpi: 300, background: '#ffffff' },
  elements: [{
    id: 'title', type: 'text', x: 6, y: 3, width: 30, height: 6, rotation: 0, zIndex: 1,
    visible: true, locked: false, value: { type: 'static', value: '测试' },
    style: { fontFamily: 'sans-serif', fontSize: 3, fontWeight: 400, color: '#111111', lineHeight: 1.2, textAlign: 'left', verticalAlign: 'top', overflow: 'clip' },
  }],
  settings: { snapToGrid: true, gridSize: 1, showGrid: false },
};

describe('标签输出尺寸预设', () => {
  it('为存量协议投影 original', () => {
    expect(labelOutputPresets(schema)).toEqual([{
      id: 'original', name: '原始尺寸', width: 90, height: 60, unit: 'mm', dpi: 300,
    }]);
  });

  it('按统一舍入规则换算并等比缩放', () => {
    const configured: LabelSchema = {
      ...schema,
      settings: { ...schema.settings, outputPresets: [
        { id: 'large', name: '大尺寸', width: 150, height: 100, unit: 'mm', dpi: 300 },
      ] },
    };
    const preset = configured.settings.outputPresets?.[0];
    expect(preset).toBeDefined();
    expect(labelPresetPixelSize(preset!)).toEqual({ width: 1772, height: 1181 });
    const scaled = withLabelOutputPreset(configured, 'large');
    expect(scaled.page).toMatchObject({ width: 150, height: 100 });
    expect(scaled.elements[0]).toMatchObject({ x: 10, y: 5 });
    expect(schema.elements[0]).toMatchObject({ x: 6, y: 3 });
  });

  it('拒绝会拉伸画布的预设', () => {
    const invalid: LabelSchema = {
      ...schema,
      settings: { ...schema.settings, outputPresets: [
        { id: 'square', name: '方形', width: 100, height: 100, unit: 'mm', dpi: 300 },
      ] },
    };
    expect(validateLabelSchema(invalid).map((issue) => issue.code)).toContain('OUTPUT_PRESET_RATIO_INVALID');
  });
});
