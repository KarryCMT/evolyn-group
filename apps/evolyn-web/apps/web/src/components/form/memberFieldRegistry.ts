import { createWebFieldRegistry } from '@evolyn.do/form/runtime-web';
import DepartmentSelectionField from './DepartmentSelectionField.vue';
import MemberSelectionField from './MemberSelectionField.vue';

/**
 * 成员目录属于租户应用能力，不能反向写进通用 form 包；宿主在此为 Web 运行时补注册。
 * 三个 Web 入口复用同一注册表，确保设计器预览与正式填写的字段行为完全一致。
 */
const memberFieldRegistry = createWebFieldRegistry()
  .register('user', { component: MemberSelectionField })
  .register('usergroup', { component: MemberSelectionField })
  .register('dept', { component: DepartmentSelectionField })
  .register('deptgroup', { component: DepartmentSelectionField });

export function getMemberFieldRegistry() {
  return memberFieldRegistry;
}
