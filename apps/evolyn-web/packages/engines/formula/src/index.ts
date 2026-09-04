/**
 * Formula Engine 的公开入口。
 *
 * 本包只承载与框架无关的公式 DSL：函数目录、AST、解析与静态诊断。
 * CodeMirror 编辑器、函数库面板等交互 UI 由 @evolyn.do/form/designer
 * 等领域包负责，避免基础引擎反向依赖 Vue 或具体业务。
 */
export * from './analyzer.js';
export * from './catalog.js';
export * from './parser.js';
export * from './types.js';
