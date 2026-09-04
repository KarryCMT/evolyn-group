import type { FieldRegistry } from './types.js';

/**
 * 创建不可变字段注册表。注册表只提供领域定义的查询入口，不修改调用方对象，
 * 让设计器、运行时和校验器能够复用同一份字段描述而不共享 UI 实现。
 */
export function createFieldRegistry<FieldType extends string, Definition>(
  definitions: Readonly<Record<FieldType, Definition>>,
): FieldRegistry<FieldType, Definition> {
  const types = Object.freeze(Object.keys(definitions) as FieldType[]);
  return Object.freeze({
    types,
    has: (type: string): type is FieldType => Object.prototype.hasOwnProperty.call(definitions, type),
    get: (type: FieldType): Definition => definitions[type],
    find: (type: string): Definition | undefined =>
      (definitions as Readonly<Record<string, Definition | undefined>>)[type],
  });
}
