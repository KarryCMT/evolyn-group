import { describe, expect, it } from 'vitest';
import { sanitizeRichTextDescription } from '../../schema/richTextDescription';

describe('sanitizeRichTextDescription', () => {
  it('保留编辑器生成的格式与安全链接', () => {
    expect(
      sanitizeRichTextDescription(
        '<p style="color: #f97316"><strong>重点</strong> <a href="https://lingyanyun.com">链接</a></p>',
      ),
    ).toBe(
      '<p style="color: #f97316"><strong>重点</strong> <a href="https://lingyanyun.com" target="_blank" rel="noopener noreferrer">链接</a></p>',
    );
  });

  it('保留编辑器输出的 RGB 颜色', () => {
    expect(
      sanitizeRichTextDescription('<p><span style="color: rgb(255, 0, 0)">红色</span></p>'),
    ).toBe('<p><span style="color: rgb(255, 0, 0)">红色</span></p>');
  });

  it('仅保留支持的字号', () => {
    expect(
      sanitizeRichTextDescription(
        '<p><span style="font-size: 22px; color: #1677ff">字号</span></p>',
      ),
    ).toBe('<p><span style="color: #1677ff; font-size: 22px">字号</span></p>');
    expect(sanitizeRichTextDescription('<p style="font-size: 24px">无效字号</p>')).toBe(
      '<p>无效字号</p>',
    );
  });

  it('移除不安全标签、事件和协议', () => {
    expect(
      sanitizeRichTextDescription(
        '<p onclick="alert(1)">说明<script>alert(1)</script><a href="javascript:alert(1)">链接</a></p>',
      ),
    ).toBe('<p>说明<a>链接</a></p>');
  });
});
