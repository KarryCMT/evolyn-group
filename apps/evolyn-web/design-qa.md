# 互联组织界面设计核验

## 对比对象

- 设计真值：`/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-511b9734-7e97-4999-a13c-97e2966942e9.png`（空态）与 `codex-clipboard-3b62747c-8d74-4319-8442-d7f2f6968105.png`（编辑链接态）。
- 实现：`http://127.0.0.1:5173/tenant/external-organization`，浏览器截图为 `/private/tmp/evolyn-external-organization-editor.png`。
- 实现视口：1280 × 720 CSS px、device scale factor 1；参考截图为 2936 × 1820 px（约 1468 × 910 CSS px 的高密度截图）。因密度与视口不同，按内容区比例、组件结构和同一交互状态核验，而非比较浏览器边框。
- 已核对状态：空态、新建连接/公开链接邀请、批量导入邀请、添加链接、编辑链接、链接开关、分享链接与删除确认。

## 比较历史

1. 初次空态：引导内容没有占满路由出口，纵向位置偏上，横幅尺寸偏大。
   - 修复：空态容器改为完整高度网格，横幅调整为参考密度下的 284px 宽，缩短横幅与说明之间的间距。
2. 初次编辑态：嵌套弹窗传送到 `body` 后跟随暗色偏好，和参考浅色弹窗不一致。
   - 修复：为两个传送弹窗补齐管理后台浅色表面、边框、文字与表单控件变量。
3. 修复后：浏览器截图显示空态、全屏邀请层及编辑链接层次、圆角、表单、遮罩和底部操作均与参考状态一致；控制台无错误。

## 必检表面

- 字体与排版：使用项目既有中文系统字体回退；标题、说明、表单标签和辅助文案层级与参考一致，无非预期截断。
- 间距与布局：空态在内容画布中垂直居中；邀请层保持顶栏下全屏面板；内层编辑弹窗采用 56px 标题栏、固定操作栏与参考相同的单列表单节奏。
- 色彩与视觉 token：全部操作态使用 Element Plus 当前主题变量；浅色弹窗明确使用管理后台表面色，避免受成员端暗色偏好影响。
- 图片质量：引导横幅使用生成的独立 PNG，不使用 CSS/HTML 绘制替代；裁切、圆角和比例与截图中的横幅位置相匹配。
- 文案：空态说明、公开邀请说明、批量导入提示、表单字段及按钮文案均按截图还原，并替换为项目约定的 `lingyanyun` 域名。

## 结果

无待修复的 P0/P1/P2 问题。剩余差异为浏览器视口密度、现有管理后台全局主题色及顶部公共导航的既有样式，均属于项目已存在的系统级呈现。

final result: passed

---

# Dashboard Designer Design QA

## Target

- Source visual 1: `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-dfafc2b9-3600-4a19-ab58-955b9cc27c79.png`
- Source visual 2: `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-bd1fa930-4676-4711-8096-ddbb81a00290.png`
- Previous implementation: `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-f4c5b34d-38a0-440b-a47d-5f2509e0c8c4.png`
- Scope: dashboard designer shell, navigation, command bar, component palette, canvas density, widget selection state, data panel, properties panel, and device preview.

## Implementation capture

- Preview URL used during QA: `http://127.0.0.1:4174/`
- Viewport: `1510 × 770` CSS pixels.
- State: desktop canvas, component palette expanded, first widget selected; data and properties drawers also verified independently.
- Capture: in-session browser screenshot taken from the temporary local visual-QA harness rendering the production `DashboardDesignerShell` and dashboard package components.

## Comparison

The target establishes a two-level horizontal header, a narrow left component palette, a large uninterrupted design canvas, dense widget spacing, a teal selection outline, and contextual tools. The updated implementation now follows that hierarchy. The former permanently visible dataset and inspector columns were moved to right-side drawers, returning most horizontal space to the canvas. Widget cards use flat borders, compact padding, and four-pixel grid gaps. The selected widget exposes edit, duplicate, and delete controls in its upper-right corner, matching the interaction pattern shown in source visual 2.

The product currently supports only the `统计图` and `明细表` widget types, so the palette intentionally does not reproduce unsupported reference items. QA fixture widgets were intentionally unbound and therefore display the existing empty-Dataset state; this verifies layout and interaction without introducing fake production data.

## Functional checks

- Component selection displays the teal outline and action tray.
- Component settings opens the contextual property drawer and closes cleanly.
- Data configuration opens the dataset drawer and closes cleanly.
- Component palette collapses to an icon rail and restores canvas width.
- Desktop and mobile canvas modes switch successfully.
- Dashboard style drawer changes the persisted grid row height and updates the canvas immediately.
- `统计图` and `明细表` both support click-to-add and drag-to-place; dropping preserves GridStack's resolved coordinates, persists a formal widget ID, and selects the new widget.
- The dashboard title switches to inline edit mode on click, supports Enter/blur to submit and Escape to cancel, and keeps unsaved canvas changes intact while the asset name is updated.
- No dashboard implementation runtime error was observed in the QA harness.

## Findings and resolution history

1. Initial capture showed the permanent `232 + 292 + 286px` side columns compressing the working canvas. Resolved by keeping only a narrow/collapsible palette and moving data/properties into drawers.
2. Initial cards used large spacing, rounded surfaces, and heavy empty-state panels. Resolved with a denser grid, flat borders, reduced padding, and neutral canvas background.
3. The initial implementation lacked reference-style selection actions. Resolved with contextual edit, duplicate, and delete controls plus a teal selected outline.
4. Visual QA initially missed shared UI package styles in the temporary harness, which prevented the grid layout from rendering correctly. The harness was corrected to load the same shared styles as the application, then the comparison was repeated.
5. The second pass found that the canvas ignored the document's persisted `rowHeight`, making the reference proportions impossible to reproduce consistently. The grid now consumes that setting, the default 80px density matches the reference card heights, and the new `仪表盘样式` drawer exposes compact, standard, and relaxed presets.
6. The second pass aligned the top navigation labels with the reference (`仪表盘设计 / 扩展功能 / 仪表盘发布`) and verified the final 1510 × 770 desktop state with zero browser warnings or errors.
7. Drag QA found two interaction blockers: native `button` palette items were ignored by GridStack's draggable guard, and an empty grid had no drop height. Palette items now use keyboard-accessible drag surfaces, and the designer reserves a 12-row empty grid. Both widget types were dragged into distinct positions with no duplicate click insertion or browser warnings.
8. The dashboard title now reuses the same shared inline title editor as the form workspace. The asset rename request updates only dashboard metadata and the browser title, so a successful rename cannot replace unsaved widget or layout edits.

## Final result

final result: passed

The dashboard designer's structure, spatial hierarchy, density, selection treatment, and contextual panel behavior now match the supplied references within the capabilities currently implemented by the product.

---

# Dashboard Widget Action Placement QA

## Target and evidence

- Source visual outside placement: `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-882674a5-0ae8-4e8c-ad63-858b9586e588.png`.
- Source visual inside placement: `/var/folders/s4/f6855yyx2j70w2r7crpmnv7h0000gn/T/codex-clipboard-a4459009-dde7-4ba3-a8c6-32e7cd1c89bb.png`.
- Implementation capture: Codex in-app browser tab 6 at `http://127.0.0.1:4174/`; the browser backend returned the screenshot pixels directly and did not expose a filesystem path.
- Viewport: `1280 × 720` CSS pixels, device scale factor `2`.
- States checked: a selected first-row widget and a selected lower-row widget.

## Focused comparison

- First row: the selected operation tray renders inside the card with a measured `6px` top and right inset, matching the reference fallback placement.
- Lower row: the selected operation tray renders above the card, right-aligned, with a measured `6px` gap between the tray bottom and card top, matching the reference external placement.
- The selected grid item rises above adjacent items and the card content remains clipped inside its own content layer, so the external tray is visible without allowing chart content to leak.
- Typography, colors, icons, copy, and card surfaces were not changed by this scoped iteration. Existing Remix icons remain sharp vector assets; no raster or generated image assets were required.
- Both placement states were exercised through the visible selector controls. The clean verification tab reported no console errors or warnings.

## Comparison history

1. Previous implementation always positioned the operation tray at `top: 4px; right: 6px`, covering card content regardless of row.
2. The canvas now derives placement from the persisted grid row: row zero uses `inside`, later rows use `outside`.
3. Post-fix browser measurements confirmed the intended `6px` inside inset and `6px` outside gap; no P0/P1/P2 mismatch remains for this interaction.

final result: passed
