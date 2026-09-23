import type {
  FieldElement,
  LabelElement,
  LabelRenderData,
  LabelSchema,
  QRCodeElement,
  TextElement,
  TextStyle,
} from '../schema/label-schema.js';
import QRCode from 'qrcode';
import { type ValueResolverOptions, resolveValueSource } from '../runtime/value-resolver.js';

export interface SvgRendererOptions extends ValueResolverOptions {
  qrCodeGenerator?: (content: string, element: QRCodeElement) => Promise<string>;
}

function escapeXml(value: string): string {
  return value
    .replace(/&/gu, '&amp;')
    .replace(/</gu, '&lt;')
    .replace(/>/gu, '&gt;')
    .replace(/"/gu, '&quot;')
    .replace(/'/gu, '&apos;');
}

function number(value: number): string {
  return Number.isFinite(value) ? String(Math.round(value * 1000) / 1000) : '0';
}

function rotation(element: LabelElement): string {
  if (!element.rotation) return '';
  const centerX = element.x + element.width / 2;
  const centerY = element.y + element.height / 2;
  return ` transform="rotate(${number(element.rotation)} ${number(centerX)} ${number(centerY)})"`;
}

function opacity(element: LabelElement): string {
  return element.opacity === undefined ? '' : ` opacity="${number(element.opacity)}"`;
}

function textAnchor(style: TextStyle): 'start' | 'middle' | 'end' {
  if (style.textAlign === 'center') return 'middle';
  if (style.textAlign === 'right') return 'end';
  return 'start';
}

function textX(element: TextElement | FieldElement): number {
  if (element.style.textAlign === 'center') return element.x + element.width / 2;
  if (element.style.textAlign === 'right') return element.x + element.width;
  return element.x;
}

function textY(element: TextElement | FieldElement): number {
  if (element.style.verticalAlign === 'middle') {
    return element.y + element.height / 2 + element.style.fontSize * 0.35;
  }
  if (element.style.verticalAlign === 'bottom') return element.y + element.height;
  return element.y + element.style.fontSize;
}

function renderTextNode(
  element: TextElement | FieldElement,
  value: string,
  clipId: string,
): string {
  const style = element.style;
  return `<g${rotation(element)}${opacity(element)}><clipPath id="${clipId}"><rect x="${number(element.x)}" y="${number(element.y)}" width="${number(element.width)}" height="${number(element.height)}"/></clipPath><text x="${number(textX(element))}" y="${number(textY(element))}" clip-path="url(#${clipId})" fill="${escapeXml(style.color)}" font-family="${escapeXml(style.fontFamily)}" font-size="${number(style.fontSize)}" font-weight="${number(style.fontWeight)}" text-anchor="${textAnchor(style)}">${escapeXml(value)}</text></g>`;
}

function safeImageHref(value: string): string {
  const normalized = value.trim();
  if (/^(data:image\/(?:png|jpeg|webp|svg\+xml);base64,|blob:|https?:\/\/)/iu.test(normalized)) {
    return escapeXml(normalized);
  }
  return '';
}

async function defaultQrCodeGenerator(content: string, element: QRCodeElement): Promise<string> {
  return QRCode.toString(content, {
    type: 'svg',
    errorCorrectionLevel: element.options.errorCorrection,
    margin: element.options.quietZone,
    color: { dark: element.options.foreground, light: element.options.background },
  });
}

function nestQrSvg(element: QRCodeElement, svg: string): string {
  const viewBox = svg.match(/viewBox="([^"]+)"/u)?.[1] ?? '0 0 100 100';
  const bodyStart = svg.indexOf('>');
  const bodyEnd = svg.lastIndexOf('</svg>');
  const body = bodyStart >= 0 && bodyEnd > bodyStart ? svg.slice(bodyStart + 1, bodyEnd) : '';
  return `<svg x="${number(element.x)}" y="${number(element.y)}" width="${number(element.width)}" height="${number(element.height)}" viewBox="${escapeXml(viewBox)}" preserveAspectRatio="xMidYMid meet"${rotation(element)}${opacity(element)}>${body}</svg>`;
}

async function renderElement(
  element: LabelElement,
  data: LabelRenderData,
  options: SvgRendererOptions,
): Promise<string> {
  if (!element.visible) return '';
  const clipId = `label-clip-${escapeXml(element.id.replace(/[^a-zA-Z0-9_-]/gu, '-'))}`;
  switch (element.type) {
    case 'text': {
      const value = await resolveValueSource(element.value, data, options);
      return renderTextNode(element, value, clipId);
    }
    case 'field': {
      const value = await resolveValueSource(element.value, data, options);
      const label = element.label ? `${element.label}${element.separator ?? ':'}` : '';
      return renderTextNode(element, `${label}${value}`, clipId);
    }
    case 'qrcode': {
      const value = await resolveValueSource(element.value, data, options);
      const generator = options.qrCodeGenerator ?? defaultQrCodeGenerator;
      return nestQrSvg(element, await generator(value, element));
    }
    case 'image': {
      const value = await resolveValueSource(element.value, data, options);
      const href = safeImageHref(value);
      if (!href) return '';
      return `<image x="${number(element.x)}" y="${number(element.y)}" width="${number(element.width)}" height="${number(element.height)}" href="${href}" preserveAspectRatio="xMidYMid ${element.objectFit === 'cover' ? 'slice' : 'meet'}"${rotation(element)}${opacity(element)}/>`;
    }
    case 'rect':
      return `<rect x="${number(element.x)}" y="${number(element.y)}" width="${number(element.width)}" height="${number(element.height)}" rx="${number(element.radius ?? 0)}" fill="${escapeXml(element.fill)}"${element.stroke ? ` stroke="${escapeXml(element.stroke)}"` : ''}${element.strokeWidth ? ` stroke-width="${number(element.strokeWidth)}"` : ''}${rotation(element)}${opacity(element)}/>`;
    case 'line':
      return `<line x1="${number(element.x)}" y1="${number(element.y)}" x2="${number(element.x + element.width)}" y2="${number(element.y + element.height)}" stroke="${escapeXml(element.color)}" stroke-width="${number(element.strokeWidth)}"${rotation(element)}${opacity(element)}/>`;
  }
}

function assertSchema(schema: LabelSchema): void {
  if (schema.schemaVersion !== '1.0') throw new Error('Unsupported LabelSchema version');
  if (!(schema.page.width > 0) || !(schema.page.height > 0)) {
    throw new Error('Label page size must be positive');
  }
}

/** 生成可直接预览或传给服务端对拍的纯 SVG，禁止 foreignObject/脚本与事件属性。 */
export async function renderLabelSvg(
  schema: LabelSchema,
  data: LabelRenderData,
  options: SvgRendererOptions = {},
): Promise<string> {
  assertSchema(schema);
  const unit = schema.page.unit;
  const elements = [...schema.elements].sort((left, right) => left.zIndex - right.zIndex);
  const body = await Promise.all(elements.map((element) => renderElement(element, data, options)));
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${number(schema.page.width)}${unit}" height="${number(schema.page.height)}${unit}" viewBox="0 0 ${number(schema.page.width)} ${number(schema.page.height)}" role="img" aria-label="${escapeXml(schema.name)}"><title>${escapeXml(schema.name)}</title><rect width="100%" height="100%" fill="${escapeXml(schema.page.background)}"/>${body.join('')}</svg>`;
}
