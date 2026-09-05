import type { Component } from 'vue';
import {
  RiArrowDownBoxFill,
  RiAttachmentFill,
  RiBuildingFill,
  RiCalendarScheduleFill,
  RiCheckDoubleFill,
  RiCheckboxMultipleFill,
  RiCommunityFill,
  RiFileListFill,
  RiFileTextFill,
  RiGroupFill,
  RiHashtag,
  RiMenuFill,
  RiRadioButtonFill,
  RiText,
  RiUser3Fill,
} from '@remixicon/vue';

/**
 * 表单 widget 类型 → 图标组件的统一映射：设计器素材面板与数据管理
 * 「列设置」面板共用，保证同一控件类型在两处图标一致。未收录的类型
 * 回落到通用字段图标，新增控件类型不至空白。
 * 注：text/number 无 Fill 版图标（remixicon 仅提供 Line 版），按语义取用。
 */
const WIDGET_ICONS: Readonly<Record<string, Component>> = {
  text: RiText,
  textarea: RiFileTextFill,
  number: RiHashtag,
  datetime: RiCalendarScheduleFill,
  radiogroup: RiRadioButtonFill,
  checkboxgroup: RiCheckboxMultipleFill,
  combo: RiArrowDownBoxFill,
  combocheck: RiCheckDoubleFill,
  separator: RiMenuFill,
  user: RiUser3Fill,
  usergroup: RiGroupFill,
  dept: RiBuildingFill,
  deptgroup: RiCommunityFill,
  attachment: RiAttachmentFill,
  subform: RiFileListFill,
};

/** 按 widget 类型取图标；缺省回落通用字段图标。 */
export function widgetIconOfType(type: string): Component {
  return WIDGET_ICONS[type] ?? RiMenuFill;
}
