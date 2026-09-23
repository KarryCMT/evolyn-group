import type { LabelSchema } from '../schema/label-schema.js';
import { describe, expect, it } from 'vitest';
import { renderLabelSvg } from '../renderer/svg-renderer.js';

const schema: LabelSchema = {
  schemaVersion: '1.0',
  name: '物料标签',
  page: { width: 80, height: 50, unit: 'mm', dpi: 300, background: '#ffffff' },
  settings: { snapToGrid: true, gridSize: 1, showGrid: false },
  elements: [
    {
      id: 'title',
      type: 'text',
      x: 4,
      y: 3,
      width: 72,
      height: 8,
      rotation: 0,
      zIndex: 1,
      visible: true,
      locked: false,
      value: { type: 'static', value: '物料<&>' },
      style: {
        fontFamily: 'Noto Sans CJK SC',
        fontSize: 5,
        fontWeight: 700,
        color: '#1b2129',
        lineHeight: 1.2,
        textAlign: 'left',
        verticalAlign: 'top',
        overflow: 'ellipsis',
      },
    },
    {
      id: 'owner',
      type: 'field',
      x: 4,
      y: 15,
      width: 42,
      height: 6,
      rotation: 0,
      zIndex: 2,
      visible: true,
      locked: false,
      label: '负责人',
      value: { type: 'field', fieldId: 'owner' },
      style: {
        fontFamily: 'Noto Sans CJK SC',
        fontSize: 4,
        fontWeight: 400,
        color: '#1b2129',
        lineHeight: 1.2,
        textAlign: 'left',
        verticalAlign: 'top',
        overflow: 'clip',
      },
    },
    {
      id: 'qr',
      type: 'qrcode',
      x: 52,
      y: 14,
      width: 24,
      height: 24,
      rotation: 0,
      zIndex: 3,
      visible: true,
      locked: false,
      value: { type: 'system', key: 'recordId' },
      options: {
        errorCorrection: 'M',
        quietZone: 1,
        foreground: '#000000',
        background: '#ffffff',
      },
    },
  ],
};

describe('renderLabelSvg', () => {
  it('以 LabelSchema 生成无脚本的矢量标签', async () => {
    const svg = await renderLabelSvg(
      schema,
      { fields: { owner: '张三' }, system: { recordId: 'record_001' } },
      {
        qrCodeGenerator: async () =>
          '<svg viewBox="0 0 10 10"><path fill="#000" d="M0 0h10v10H0z"/></svg>',
      },
    );

    expect(svg).toContain('width="80mm"');
    expect(svg).toContain('负责人:张三');
    expect(svg).toContain('物料&lt;&amp;&gt;');
    expect(svg).toContain('<svg x="52" y="14"');
    expect(svg).not.toContain('<script');
    expect(svg).not.toContain('foreignObject');
  });
});
