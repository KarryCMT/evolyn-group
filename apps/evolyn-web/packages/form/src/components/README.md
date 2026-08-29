# 表单包组件说明

本目录承载表单设计器的展示组件素材说明（目标保存协议，ADR-010）。设计器核心组件
（素材面板/画布/属性面板/预览控件）位于 `src/designer/`，随 `@evolyn.do/form/designer`
入口导出；运行时执行内核位于 `src/runtime/renderer/`，Web 组合表面位于
`src/runtime/surface/`。

## 设计器（src/designer/）

| 组件                          | 中文名        | 主要职责                                                                                                                                                             |
| ----------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `FormSchemaPalette.vue`       | 字段素材面板  | 按分组展示控件入口（P2 仅基础 9 类可添加，其余分组置灰），支持点击添加与拖拽克隆到画布。                                                                             |
| `FormSchemaCanvas.vue`        | 字段设计画布  | 目标协议 items 的拖拽排序、选择、复制、删除与落点插入；预览态按 lineWidth 收敛宽度。                                                                                 |
| `FormSchemaItemPreview.vue`   | 字段预览控件  | 按 widget.type 渲染画布内禁用态预览（Element Plus 控件）。                                                                                                           |
| `FormSchemaPropertyPanel.vue` | 属性配置面板  | 通过「字段属性 / 表单属性」切换：字段页编辑公共属性（label/description/allowBlank/enable/visible/labelHidden/lineWidth）与按类型分派的专属配置；表单页编辑资产名称。 |
| `useFormSchemaEditor.ts`      | 编辑状态 hook | 页面唯一持有 content.items；增删/复制/选中/重命名动作收口；持久化由页面接草稿接口。                                                                                  |
| `palette.ts`                  | 拖拽契约      | 素材面板与画布共享的拖拽分组名与临时对象标记。                                                                                                                       |

## 运行时（src/runtime/）

| 组件                       | 中文名       | 主要职责                                                                                     |
| -------------------------- | ------------ | -------------------------------------------------------------------------------------------- |
| `FormRenderer.vue`         | 渲染执行内核 | 校验输入协议、创建运行时会话、渲染字段、执行校验和提交/填写草稿命令，不渲染具体操作按钮。    |
| `FormRuntimeSurface.vue`   | 运行时表面   | 推荐 Web 组合入口；统一唯一滚动内容区、固定操作区、桌面/移动布局和内置动作分派。             |
| `FormRuntimeActionBar.vue` | 运行时操作栏 | 根据动作描述符渲染任意数量操作、错误摘要、移动折叠与更多菜单；只发事件，不直接调用业务 API。 |
| `FormSectionRenderer.vue`  | 区块渲染器   | 按预编译 render plan 渲染区块序列（当前平铺单区块）。                                        |
| `FormFieldHost.vue`        | 字段外壳     | 标签/必填/说明/错误/隐藏标签/跨列（lineWidth）/隐藏（visible）的通用展示。                   |
| `widgets/base/*`           | 基础字段组件 | 9 类基础字段的原生控件实现，按 widget.type 注册。                                            |

> 旧插件设计器遗留组件（PluginDesign\* / FormSubform\* / FormCodeEditor 等）已随
> FormDocument 协议一并退场，见 docs/低代码平台/表单设计器/迁移清单与验收矩阵.md；
> P4 子表单设计器将按目标协议重建。
