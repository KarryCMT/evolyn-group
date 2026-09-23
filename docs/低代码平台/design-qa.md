# 成员信息管理界面还原 QA

## 对比目标

- Source visual truth: `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-20c98b95-1335-44fa-ab71-7d60b2ffb07d.png`、`/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-52b376cd-624b-4920-bdc2-8afaf0542fec.png`、`/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-17052635-6554-4b3e-aceb-1a252b20f299.png`、`/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-944c7860-ec51-424a-b1c1-9f343c5422b6.png`
- Implementation captures: `/private/tmp/member-fields-settings-final.png`、`/private/tmp/member-card-desktop-final.png`、`/private/tmp/member-card-mobile-final.png`
- Viewport: `1466 × 908` CSS px；参考图为 `2932 × 1816` 附近的 2x 图，按 2x 密度进行视觉对齐。
- State: 字段设置初始顶端、卡片展示桌面端、卡片展示移动端。

## 对比结论

**Findings**

- 未发现可操作的 P0、P1 或 P2 差异。
- 字体与层级：页签、说明、表头、字段行、预览卡片均遵循参考的层级和中文文案；沿用工程当前主题字体与字色。
- 布局与节奏：固定二级页签、三列表格、表内滚动、左侧字段选择及右侧桌面/移动预览的比例已与参考状态对齐。
- 色彩与组件状态：选中、禁用、悬停与焦点状态复用当前 Element Plus 主题色；系统字段以弱化禁用勾选表达固定配置。
- 图像质量：桌面与移动预览均使用项目内 `member-card-preview-desktop-v1.png` 背景素材，未以 CSS/SVG 伪造截图中的成员列表预览。
- 内容：成员示例信息、字段名称、字段类型和权限文案与参考一致。

**Open Questions**

- 无。字段配置目前为前端交互状态，等待后端接口可用时再持久化。

**Implementation Checklist**

- [x] 字段设置：系统字段禁用、普通字段可见性/可编辑联动、表内滚动。
- [x] 卡片展示：字段选择实时增减成员详情。
- [x] 卡片展示：桌面端与移动端预览切换。
- [x] 浏览器检查：核心交互已操作，页面控制台无错误。

**Follow-up Polish**

- 无 P3 项。

## 比较历史

1. 初次对比发现字段列表行高与参考的 2x 视觉密度不一致，已将表头和字段行调整为 56px，并收紧说明与表格的间距。
2. 初次卡片对比发现预览区横向分栏和设备切换器偏宽，已调整为 268px 字段栏、120px 设备切换器，并对齐桌面卡片宽度与位置。
3. 复查字段设置、桌面预览和移动预览；已验证隐藏“编号”会即时从预览卡片移除，且无浏览器控制台错误。

final result: passed
