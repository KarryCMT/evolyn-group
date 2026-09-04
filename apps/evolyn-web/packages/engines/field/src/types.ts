/**
 * 字段的声明式能力描述。它刻意不包含 Vue 组件、属性面板或具体存储协议；
 * 领域包以自己的字段 DSL 实例化该类型，再由查询、规则和校验能力消费。
 */
export interface FieldDefinition<
  Group extends string = string,
  ValueKind extends string = string,
  Property = unknown,
> {
  label: string;
  group: Group;
  valueKind: ValueKind;
  labelOptional?: boolean;
  props: Readonly<Record<string, Property>>;
}

export interface FieldRegistry<FieldType extends string, Definition> {
  readonly types: readonly FieldType[];
  has(type: string): type is FieldType;
  get(type: FieldType): Definition;
  find(type: string): Definition | undefined;
}
