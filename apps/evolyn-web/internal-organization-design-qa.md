# 内部组织界面设计核验

## 对比对象

- 设计真值：
  - `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-6c0e9d1c-5fc3-4b77-8613-b11b2973be6b.png`（部门成员页）。
  - `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-3e4ca205-79e4-4116-98de-b6331ad5bef1.png`（角色成员页）。
  - `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-c371284c-5b9b-479c-83c3-069e0be0e620.png`（添加成员）。
  - `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-b14a4957-06ec-4dce-b5c2-edb020ef1ff1.png`（调整分组）。
- 实现路由：`http://127.0.0.1:4173/tenant/organization`。
- 浏览器渲染证据：
  - `/private/tmp/evolyn-internal-organization-department.png`
  - `/private/tmp/evolyn-internal-organization-role.png`
  - `/private/tmp/evolyn-internal-organization-picker.png`
  - `/private/tmp/evolyn-internal-organization-group-dialog.png`
- 实现视口：2014 × 972 CSS px、device scale factor 1；参考图为 2014 × 1248 px 的缩放版本。对比保持相同桌面宽度，因本地浏览器可视高度较短，仅裁切底部留白与分页以下区域。

## 已验证交互

- 部门/角色页签切换、成员搜索、状态筛选、部门和角色树选中态。
- 角色添加成员选择器的搜索、选择、取消及确认闭环。
- 角色组调整弹窗的目标分组选择与确认。
- 新建角色/角色组、角色名称修改、成员资料抽屉、导出和邀请入口均有可见反馈。
- 浏览器控制台未发现由内部组织页面引起的错误。

## 比较结果

### 字体与排版

沿用应用中文系统字体回退，左侧双页签、目录树、成员表头和对话框标题层级与参考一致；长组织名称保持单行截断，未出现挤压或重叠。

### 间距与布局

组织页采用 356px 左侧工作栏、内容区固定工具栏和底部分页；邀请横幅、标题分隔、成员表格和角色树的纵向节奏与参考状态对齐。小于 1160px 时工具栏换行，不影响搜索和操作按钮使用。

### 色彩与视觉 token

选中态、按钮、危险操作和遮罩均使用 Element Plus 主题相关变量。管理后台切入时明确切换 VTable 至浅色根变量，修复了暗色偏好下画布表格误渲染为深色的 P1 问题。

### 图片、图标与文案

邀请横幅使用独立生成的成员协作 PNG；导航、树节点、搜索和操作图标均采用项目既有 Remix Fill 图标。截图中涉及的成员、角色、组织和操作文案已被覆盖。

### 焦点区域核验

- 部门成员页：横幅、左树、筛选工具栏、成员行和分页均可见，布局正确。
- 角色成员页：角色组树、头部文字操作、导入/导出和角色维度表格均可见。
- 添加成员：遮罩、已选标签、搜索、左组织树、右成员列表和固定底栏可用。
- 调整分组：居中对话框、列表选中态和确认区可用。

## 结论

无待修复的 P0/P1/P2 问题。生成插画与原截图的具体人物细节不同，但尺寸、裁切、色调和布局用途一致，作为可接受的 P3 视觉差异保留。

final result: passed
