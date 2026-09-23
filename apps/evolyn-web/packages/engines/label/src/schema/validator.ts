import type { LabelElement, LabelSchema, TextStyle, ValueSource } from './label-schema.js';

export interface LabelValidationIssue {
  path: string;
  code: string;
  message: string;
}

const colorPattern = /^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$/;
const supportedDpi = new Set([96, 203, 300, 600]);
const supportedFonts = new Set(['Arial', 'Noto Sans', 'Noto Sans CJK SC', 'sans-serif']);
const systemKeys = new Set([
  'recordId',
  'createdAt',
  'updatedAt',
  'createdBy',
  'currentUser',
  'currentDate',
]);

function finite(value: number): boolean {
  return Number.isFinite(value);
}

function positiveFinite(value: number): boolean {
  return value > 0 && finite(value);
}

function validColor(value: string | undefined): boolean {
  return typeof value === 'string' && colorPattern.test(value);
}

function validateValue(
  value: ValueSource | undefined,
  path: string,
  add: (path: string, code: string, message: string) => void,
): void {
  if (!value) {
    add(path, 'VALUE_SOURCE_REQUIRED', '值来源不能为空');
    return;
  }
  switch (value.type) {
    case 'static':
      break;
    case 'field':
      if (!value.fieldId) add(`${path}.fieldId`, 'FIELD_ID_REQUIRED', '字段绑定不能为空');
      break;
    case 'system':
      if (!systemKeys.has(value.key)) add(`${path}.key`, 'SYSTEM_KEY_INVALID', '系统字段无效');
      break;
    case 'expression':
      if (!value.expression) add(`${path}.expression`, 'EXPRESSION_REQUIRED', '表达式不能为空');
      break;
    default:
      add(`${path}.type`, 'VALUE_SOURCE_TYPE_INVALID', '值来源类型无效');
  }
}

function validateTextStyle(
  style: TextStyle | undefined,
  path: string,
  add: (path: string, code: string, message: string) => void,
): void {
  if (!style) {
    add(path, 'TEXT_STYLE_REQUIRED', '文本样式不能为空');
    return;
  }
  if (
    !supportedFonts.has(style.fontFamily) ||
    !positiveFinite(style.fontSize) ||
    style.fontWeight < 100 ||
    style.fontWeight > 900 ||
    !positiveFinite(style.lineHeight)
  ) {
    add(path, 'TEXT_STYLE_INVALID', '字体、字号或字重无效');
  }
  if (!validColor(style.color)) add(`${path}.color`, 'COLOR_INVALID', '文本颜色必须为十六进制颜色');
  if (!['left', 'center', 'right'].includes(style.textAlign)) {
    add(`${path}.textAlign`, 'TEXT_ALIGN_INVALID', '文本水平对齐方式无效');
  }
  if (!['top', 'middle', 'bottom'].includes(style.verticalAlign)) {
    add(`${path}.verticalAlign`, 'TEXT_ALIGN_INVALID', '文本垂直对齐方式无效');
  }
  if (!['clip', 'ellipsis', 'wrap'].includes(style.overflow)) {
    add(`${path}.overflow`, 'TEXT_OVERFLOW_INVALID', '文本溢出策略无效');
  }
}

function validateElement(
  element: LabelElement,
  path: string,
  add: (path: string, code: string, message: string) => void,
): void {
  switch (element.type) {
    case 'text':
    case 'field':
      validateValue(element.value, `${path}.value`, add);
      validateTextStyle(element.style, `${path}.style`, add);
      break;
    case 'qrcode':
      validateValue(element.value, `${path}.value`, add);
      if (!element.options) {
        add(`${path}.options`, 'QR_OPTIONS_REQUIRED', '二维码配置不能为空');
        return;
      }
      if (!['L', 'M', 'Q', 'H'].includes(element.options.errorCorrection)) {
        add(`${path}.options.errorCorrection`, 'QR_LEVEL_INVALID', '二维码纠错级别无效');
      }
      if (element.options.quietZone < 0 || element.options.quietZone > 16) {
        add(`${path}.options.quietZone`, 'QR_QUIET_ZONE_INVALID', '二维码静区必须在 0 到 16 之间');
      }
      if (!validColor(element.options.foreground) || !validColor(element.options.background)) {
        add(`${path}.options`, 'COLOR_INVALID', '二维码颜色必须为十六进制颜色');
      }
      break;
    case 'image':
      validateValue(element.value, `${path}.value`, add);
      if (!['contain', 'cover'].includes(element.objectFit)) {
        add(`${path}.objectFit`, 'IMAGE_FIT_INVALID', '图片缩放方式无效');
      }
      break;
    case 'rect':
      if (!validColor(element.fill) || (element.stroke !== undefined && !validColor(element.stroke))) {
        add(path, 'COLOR_INVALID', '矩形颜色必须为十六进制颜色');
      }
      break;
    case 'line':
      if (!validColor(element.color) || element.strokeWidth <= 0) {
        add(path, 'LINE_STYLE_INVALID', '线条颜色或宽度无效');
      }
      break;
    default:
      add(`${path}.type`, 'ELEMENT_TYPE_UNSUPPORTED', '不支持的标签元素类型');
  }
}

/** 与 Go 内核同语义校验 LabelSchema，并返回可定位到设计器元素的稳定问题。 */
export function validateLabelSchema(schema: LabelSchema | null | undefined): LabelValidationIssue[] {
  const issues: LabelValidationIssue[] = [];
  const add = (path: string, code: string, message: string) => issues.push({ path, code, message });
  if (!schema) {
    add('$', 'SCHEMA_REQUIRED', '标签 Schema 不能为空');
    return issues;
  }
  if (schema.schemaVersion !== '1.0') add('$.schemaVersion', 'VERSION_UNSUPPORTED', '仅支持 LabelSchema 1.0');
  if (!schema.name) add('$.name', 'NAME_REQUIRED', '标签名称不能为空');
  if (!positiveFinite(schema.page.width) || !positiveFinite(schema.page.height)) {
    add('$.page', 'PAGE_SIZE_INVALID', '标签宽高必须为正数');
  }
  if (!['mm', 'px'].includes(schema.page.unit)) add('$.page.unit', 'PAGE_UNIT_INVALID', '标签单位仅支持 mm 或 px');
  if (!supportedDpi.has(schema.page.dpi)) add('$.page.dpi', 'PAGE_DPI_INVALID', 'DPI 仅支持 96、203、300、600');
  if (!validColor(schema.page.background)) add('$.page.background', 'COLOR_INVALID', '页面背景色必须为十六进制颜色');
  if (!positiveFinite(schema.settings.gridSize)) add('$.settings.gridSize', 'GRID_SIZE_INVALID', '网格尺寸必须为正数');
  if (schema.elements.length > 200) add('$.elements', 'ELEMENT_LIMIT_EXCEEDED', '元素数量不能超过 200');

  const ids = new Set<string>();
  schema.elements.forEach((element, index) => {
    const path = `$.elements[${index}]`;
    if (!element.id) add(`${path}.id`, 'ELEMENT_ID_REQUIRED', '元素 ID 不能为空');
    else if (ids.has(element.id)) add(`${path}.id`, 'ELEMENT_ID_DUPLICATED', '元素 ID 不能重复');
    ids.add(element.id);
    if (
      !finite(element.x) ||
      !finite(element.y) ||
      !positiveFinite(element.width) ||
      !positiveFinite(element.height)
    ) {
      add(path, 'ELEMENT_BOUNDS_INVALID', '元素坐标必须有限且宽高必须为正数');
    }
    if (!finite(element.rotation)) add(`${path}.rotation`, 'ELEMENT_ROTATION_INVALID', '旋转角度必须是有限数值');
    if (element.opacity !== undefined && (element.opacity < 0 || element.opacity > 1 || !finite(element.opacity))) {
      add(`${path}.opacity`, 'ELEMENT_OPACITY_INVALID', '透明度必须在 0 到 1 之间');
    }
    validateElement(element, path, add);
  });
  return issues;
}
