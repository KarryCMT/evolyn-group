/**
 * `@evolyn.do/form/schema` 纯 TS Schema 层（ADR-010）：目标保存协议的类型、字段字典、
 * 严格校验器、深拷贝器、版本迁移器、基础字段值编解码、字段显隐规则编译/求值器（v5）
 * 与不可见字段赋值策略解析（v6）。本层不得依赖 Vue / Element Plus / 设计器与运行时；
 * 前端校验与后端 Go 校验器（internal/platform/form）对同一 JSON 的结论必须一致。
 */
export * from './types';
export * from './dictionary';
export * from './validate';
export * from './clone';
export * from './migrate';
export * from './codec';
export * from './rules';
export * from './invisible-value-policy';
