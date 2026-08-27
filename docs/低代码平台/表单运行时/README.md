# 表单运行时

本目录承载“已发布表单”的最终渲染、填写、校验、提交与跨端适配设计；不承载表单设计器、拖拽编排、代码编辑或设计草稿。

## 文档导航

- [表单最终渲染器架构与实施设计.md](./表单最终渲染器架构与实施设计.md)：运行时边界、Schema 契约、状态与规则引擎、组件和字段注册、移动端性能、接口、安全、测试及分期实施方案（§4.1 的 FormDocument 模型已按 ADR-010 被目标保存协议取代，其余结论继续有效）。
- [表单设计器目标保存协议重构与分期实施方案](../表单设计器/目标保存协议重构与分期实施方案.md)：已确认的表单设计器保存协议重构决策和执行阶段（ADR-010）。
- [目标协议字段字典](../表单设计器/目标协议字段字典.md)：27 种 `widget.type` 的属性/值协议/发布白名单（运行时注册与提交校验的唯一依据）。

## 实现位置（P1+P2 基础字段闭环已落地，ADR-010）

运行时代码位于 `apps/evolyn-web/packages/form/src/`，入口按文档 §3.2 拆分；三入口均直接读写目标保存协议 `content.items[].widget`（不再存在 FormDocument 与画布转换器）：

| 入口 | 目录 | 说明 |
| --- | --- | --- |
| `@evolyn.do/form/schema` | `src/schema/` | 纯 TypeScript 协议层：27 种 `widget.type` 字典（`dictionary.ts`）、严格校验器（`validate.ts`，JSON Path 级错误）、深拷贝（`clone.ts`）、版本迁移器（`migrate.ts`）、基础字段值编解码（`codec.ts`，与后端逐字一致） |
| `@evolyn.do/form/runtime` | `src/runtime/` | 最终渲染器：会话 Store（`store/createFormRuntime.ts`）、按 `widget.type` 注册的字段注册表与基础 9 类组件（`widgets/`）、渲染组件（`renderer/`，执行 enable/visible/labelHidden/lineWidth）、Adapter 契约（`adapters/types.ts`）、独立样式（`runtime/style.css`）；提交载荷 `{formId, publishedVersion, schemaRevision, values}`（值按 `widgetName` 取键） |
| `@evolyn.do/form/designer` | `src/designer/` | 设计器入口（管理端按需引入）：素材面板/画布/属性面板组件与 `useFormSchemaEditor`（页面状态即协议文档）；发布白名单 `validatePublishableFormSchema` 前置拦截 |

- 主入口 `@evolyn.do/form` 为兼容期别名，等价于 designer 入口。
- 后端表单资产域：`apps/evolyn-core/internal/platform/form/`（草稿/发布 bootstrap `GET /applications/code/:appCode/forms/:formId/runtime`/提交 `POST /form-records`，见[表单资产域后端契约](../表单设计器/表单资产域后端契约.md)）。
- Web 宿主接入：`apps/web/src/pages/form/preview.vue`（已发布走 bootstrap + 真实提交；未发布回退 sessionStorage 草稿本地回放，`--evf-*` 主题变量映射到 `--el-*`）；设计器 `apps/web/src/pages/form/design.vue`（草稿保存/发布/预览）。
- 测试：`packages/form` 内 `pnpm test`（vitest + happy-dom：协议校验/值编解码/Store/渲染组件）。

尚未落地（随表单设计器方案 P3–P6 分期）：组织/文件/位置字段（P3）、子表单/关联/聚合/流水号（P4）、规则依赖图与富交互（P5，对应本文档 §6/§13 R2+ 的声明式规则方向）、迁移收口与可观测性（P6）。
