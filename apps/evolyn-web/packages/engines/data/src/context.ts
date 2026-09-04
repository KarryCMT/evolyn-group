import type { DataContext } from './types.js';

/**
 * 建立不可变的数据访问上下文，避免数据源适配器意外修改领域侧传入的元数据。
 */
export function createDataContext(
  resource: string,
  metadata?: Readonly<Record<string, unknown>>,
): DataContext {
  const normalizedResource = resource.trim();
  if (!normalizedResource) throw new Error('Data context resource must not be empty.');

  return Object.freeze({
    resource: normalizedResource,
    metadata: metadata ? Object.freeze({ ...metadata }) : undefined,
  });
}
