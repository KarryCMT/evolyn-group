/**
 * 字段无障碍 DOM id 约定：FormFieldHost 渲染 label/说明/错误，
 * 字段组件渲染输入控件，两侧按同一约定生成 id 保证 aria 关联（文档 §7.4）。
 */

function sanitizeKey(key: string): string {
  // id 不允许空白字符；异常 key 收敛为下划线，避免破坏 label for / aria-describedby 关联。
  return key.replace(/[^A-Za-z0-9_-]/g, '_');
}

export function fieldInputId(key: string): string {
  return `evf-field-${sanitizeKey(key)}`;
}

/**
 * 宿主标签 id：单控件字段经 label[for=inputId] 关联；
 * 单选/复选等分组字段在容器上以 aria-labelledby 引用该 id（label 的悬空 for 会被浏览器忽略）。
 */
export function fieldLabelId(key: string): string {
  return `evf-field-${sanitizeKey(key)}-label`;
}

export function fieldDescriptionId(key: string): string {
  return `evf-field-${sanitizeKey(key)}-desc`;
}

export function fieldErrorId(key: string): string {
  return `evf-field-${sanitizeKey(key)}-error`;
}

/** 组装 aria-describedby：仅拼接实际存在的说明/错误节点，避免悬空引用。 */
export function fieldAriaDescribedBy(
  key: string,
  hasDescription: boolean,
  hasErrors: boolean,
): string | undefined {
  const ids: string[] = [];
  if (hasDescription) ids.push(fieldDescriptionId(key));
  if (hasErrors) ids.push(fieldErrorId(key));
  return ids.length ? ids.join(' ') : undefined;
}
