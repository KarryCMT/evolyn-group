# 表单包组件说明

本目录承载表单设计器的展示组件素材说明（目标保存协议，ADR-010）。设计器核心组件
（素材面板/画布/属性面板/预览控件）位于 `src/designer/`，随 `@evolyn.do/form/designer`
入口导出；运行时执行内核位于 `src/runtime/renderer/`，Web 组合表面位于
`src/runtime/surface/`。

## 设计器（src/designer/）

| 组件                                  | 中文名        | 主要职责                                                                                              |
| ------------------------------------- | ------------- | ----------------------------------------------------------------------------------------------------- |
| `FormSchemaPalette.vue`               | 字段素材面板  | 按分组展示控件入口（P2 仅基础 9 类可添加，其余分组置灰），支持点击添加与拖拽克隆到画布。              |
| `FormSchemaLayoutCanvas.vue`          | 布局设计画布  | 按 v2 引用序列编排顶层字段与 `multitab`；负责布局选中、预览及字段在顶层/标签页间拖拽。                |
| `FormSchemaFieldCard.vue`             | 字段设计卡片  | 复用字段标题、预览、复制与删除交互；字段定义仍唯一存放在 `content.items`。                            |
| `FormSchemaSubformCard.vue`           | 子表单设计卡  | 以桌面表格形态展示嵌套字段，承接素材拖入、子字段排序、选择、复制与删除。                              |
| `FormSchemaSubformPropertyPanel.vue`  | 子表单属性区  | 配置基础信息、子字段、行权限、快速填报、冻结列及移动端纵向/横向展示。                                 |
| `FormSchemaMultitabPropertyPanel.vue` | 标签页属性区  | 在右侧属性面板内维护标签样式、标题、增删、复制与拖拽排序。                                            |
| `FormSchemaItemPreview.vue`           | 字段预览控件  | 按 widget.type 渲染画布内禁用态预览（Element Plus 控件）。                                            |
| `FormSchemaPropertyPanel.vue`         | 属性配置面板  | 通过「字段属性 / 表单属性」切换；字段页根据画布选中节点分派字段属性或标签页属性，表单页编辑资产名称。 |
| `useFormSchemaEditor.ts`              | 编辑状态 hook | 页面唯一持有字段定义与布局引用；增删、复制、重命名、跨容器移动和解散动作集中维护引用完整性。          |
| `palette.ts`                          | 拖拽契约      | 素材面板与画布共享的拖拽分组名与临时对象标记。                                                        |

## 运行时（src/runtime/）

| 组件                       | 中文名       | 主要职责                                                                                     |
| -------------------------- | ------------ | -------------------------------------------------------------------------------------------- |
| `FormRenderer.vue`         | 渲染执行内核 | 校验输入协议、创建运行时会话、渲染字段、执行校验和提交/填写草稿命令，不渲染具体操作按钮。    |
| `FormRuntimeSurface.vue`   | 运行时表面   | 推荐 Web 组合入口；统一唯一滚动内容区、固定操作区、桌面/移动布局和内置动作分派。             |
| `FormRuntimeActionBar.vue` | 运行时操作栏 | 根据动作描述符渲染任意数量操作、错误摘要、移动折叠与更多菜单；只发事件，不直接调用业务 API。 |
| `FormSectionRenderer.vue`  | 区块渲染器   | 按预编译 render plan 分派顶层字段与标签页布局。                                              |
| `FormMultitabRenderer.vue` | 标签页渲染器 | 渲染 `multitab`，切页保留会话状态，错误定位时先激活字段所属标签页。                          |
| `FormFieldHost.vue`        | 字段外壳     | 标签/必填/说明/错误/隐藏标签/跨列（lineWidth）/隐藏（visible）的通用展示。                   |
| `widgets/base/*`           | 基础字段组件 | 9 类基础字段的原生控件实现，按 widget.type 注册。                                            |

> 旧插件设计器遗留组件（PluginDesign\* / FormSubform\* / FormCodeEditor 等）已随
> FormDocument 协议一并退场，见 docs/低代码平台/表单设计器/迁移清单与验收矩阵.md；
> P4 子表单设计态已按 v4 目标协议重建；运行时行编辑、记录提交与服务端行级终审仍按 P4
> 后续纵切推进。
