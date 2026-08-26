import type {
  AdminAddressBookScopeDto,
  AdminGroupDetailDto,
  AdminGroupSummaryDto,
  AdminScopeMode,
} from '~/api/adminGroup';

/** 管理组类型：system=系统管理员页，application=灵衍云管理员页。 */
export type AdministratorScope = 'system' | 'application';

/** 范围模式：all=全部，partial=部分（ID 清单）。 */
export type ScopeMode = AdminScopeMode;

/** 管理组成员展示项（后端 AdminGroupMemberDto）。 */
export type AdministratorMember = AdminGroupDetailDto['members'][number];

/** 选择器成员：比管理组详情多携带账号和部门 ID，用于创建人拦截与左树过滤。 */
export type AdministratorPickerMember = AdministratorMember & {
  accountId: number;
  departmentIds?: number[];
};

/**
 * 管理组详情（权限面板本地状态）：与后端 AdminGroupDetailDto 字段一一对应；
 * application 专属字段（applicationIds/allApplications/applicationManage/
 * addressBook）在 system 组上为空态，不参与提交。
 */
export type AdministratorGroup = AdminGroupDetailDto;

/** 管理组概要（列表项）。 */
export type AdministratorGroupSummary = AdminGroupSummaryDto;

/** 通讯录管理子配置（application 组设置抽屉，与主行分发范围解耦）。 */
export type AddressBookScope = AdminAddressBookScopeDto;

/** 部分范围的选择目标：部门树或「角色组→角色」两级树。 */
export type AdministratorScopeTarget = 'department' | 'role';

/** 可编辑应用条目：管理组选择器消费的应用字段子集（icon 渲染映射同工作台）。 */
export interface AdministratorApplication {
  id: number;
  name: string;
  icon: string;
}
