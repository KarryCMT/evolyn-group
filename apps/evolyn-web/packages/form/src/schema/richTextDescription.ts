/**
 * 将字段说明限制在富文本编辑器能够产生的安全子集内，再交给设计器与运行时渲染。
 * 协议内容可能来自已保存的表单，不能因设计器侧可编辑就直接信任原始 HTML。
 */
const allowedTags = new Set([
  'a',
  'b',
  'blockquote',
  'br',
  'code',
  'em',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'i',
  'img',
  'li',
  'ol',
  'p',
  's',
  'span',
  'strike',
  'strong',
  'u',
  'ul',
]);

/** 说明编辑器仅输出颜色、字号与对齐；仅保留这些受控样式。 */
function sanitizeStyle(element: HTMLElement, style: string): void {
  const declarations: string[] = [];
  // Tiptap 在不同浏览器中可能输出 #rrggbb 或 rgb()/rgba()，均为颜色选择器的正常结果。
  const color = style.match(
    /(?:^|;)\s*color\s*:\s*(#[\da-f]{3,8}|rgba?\(\s*(?:\d{1,3}\s*,\s*){2}\d{1,3}(?:\s*,\s*(?:0|0?\.\d+|1))?\))\s*(?:;|$)/i,
  )?.[1];
  if (color) declarations.push(`color: ${color}`);

  const fontSize = style.match(/(?:^|;)\s*font-size\s*:\s*(12|14|16|18|20|22)px\s*(?:;|$)/i)?.[1];
  if (fontSize) declarations.push(`font-size: ${fontSize}px`);

  const textAlign = style.match(
    /(?:^|;)\s*text-align\s*:\s*(left|center|right|justify)\s*(?:;|$)/i,
  )?.[1];
  if (textAlign) declarations.push(`text-align: ${textAlign}`);

  if (declarations.length) element.setAttribute('style', declarations.join('; '));
  else element.removeAttribute('style');
}

function isSafeUrl(value: string, allowRelative = false): boolean {
  if (allowRelative && value.startsWith('/') && !value.startsWith('//')) return true;
  try {
    return ['http:', 'https:', 'mailto:', 'tel:'].includes(new URL(value).protocol);
  } catch {
    return false;
  }
}

/** 输出可安全挂载到设计器画布和表单运行时的字段说明 HTML。 */
export function sanitizeRichTextDescription(html: string): string {
  if (!html) return '';
  // 服务端预渲染时没有 DOM，回退为纯文本，避免把未净化的 HTML 直接写入页面。
  if (typeof DOMParser === 'undefined') {
    return html.replace(
      /[&<>"']/g,
      (character) =>
        ({
          '&': '&amp;',
          '<': '&lt;',
          '>': '&gt;',
          '"': '&quot;',
          "'": '&#39;',
        })[character]!,
    );
  }

  const document = new DOMParser().parseFromString(html, 'text/html');
  for (const element of [...document.body.querySelectorAll<HTMLElement>('*')].reverse()) {
    const tag = element.tagName.toLowerCase();
    if (!allowedTags.has(tag)) {
      if (['script', 'style', 'iframe', 'object', 'svg'].includes(tag)) element.remove();
      else element.replaceWith(...Array.from(element.childNodes));
      continue;
    }

    const href = element.getAttribute('href');
    const src = element.getAttribute('src');
    const style = element.getAttribute('style') ?? '';
    for (const attribute of [...element.attributes]) element.removeAttribute(attribute.name);
    if (tag === 'a') {
      if (href && isSafeUrl(href, true)) {
        element.setAttribute('href', href);
        element.setAttribute('target', '_blank');
        element.setAttribute('rel', 'noopener noreferrer');
      }
    } else if (tag === 'img') {
      if (src && isSafeUrl(src, true)) element.setAttribute('src', src);
      else element.remove();
    }
    sanitizeStyle(element, style);
  }
  return document.body.innerHTML;
}
