/** 可在选择器中出现的主体类型。 */
export type EvolynMemberDepartmentRolePickerItemType = 'department' | 'role' | 'member';

/** 业务侧主键可以直接保留为接口返回的数值或字符串。 */
export type EvolynMemberDepartmentRolePickerItemId = string | number;

/** 部门和角色共用的层级节点。角色没有层级时，直接传入一级节点即可。 */
export interface EvolynMemberDepartmentRolePickerTreeNode {
  id: EvolynMemberDepartmentRolePickerItemId;
  label: string;
  children?: EvolynMemberDepartmentRolePickerTreeNode[];
  /** 供搜索使用的额外文本，例如拼音、工号或别名。 */
  keywords?: string[];
  /** 禁用节点会保留展示，但不可新增或移除选择。 */
  disabled?: boolean;
}

/** 成员数据由业务侧聚合；组件不依赖具体账号、成员或租户接口模型。 */
export interface EvolynMemberDepartmentRolePickerMember {
  id: EvolynMemberDepartmentRolePickerItemId;
  label: string;
  /** 成员所属部门，用于“成员”页签的左树筛选。 */
  departmentIds?: EvolynMemberDepartmentRolePickerItemId[];
  avatarUrl?: string;
  keywords?: string[];
  disabled?: boolean;
}

/** 确认后的统一选择结果，父组件可直接用于提交权限主体等业务数据。 */
export interface EvolynMemberDepartmentRolePickerSelection {
  id: EvolynMemberDepartmentRolePickerItemId;
  label: string;
  type: EvolynMemberDepartmentRolePickerItemType;
  avatarUrl?: string;
}

export interface EvolynMemberDepartmentRolePickerProps {
  /** 弹窗标题。 */
  title?: string;
  departments?: EvolynMemberDepartmentRolePickerTreeNode[];
  roles?: EvolynMemberDepartmentRolePickerTreeNode[];
  members?: EvolynMemberDepartmentRolePickerMember[];
  /** 控制哪些主体页签可用，默认全部可选。 */
  selectableTypes?: EvolynMemberDepartmentRolePickerItemType[];
  /** 是否允许多选；关闭后每次选择会替换暂存结果。 */
  multiple?: boolean;
  /**
   * 部门是否允许多选。未传时继承 multiple；传 false 时仅限制部门之间互斥，
   * 不会移除已选择的成员或角色。
   */
  departmentMultiple?: boolean;
  /**
   * 成员是否允许多选。未传时继承 multiple；传 false 时仅限制成员之间互斥，
   * 不会移除已选择的部门或角色。
   */
  memberMultiple?: boolean;
  /** 最多可选数量；不传表示不限制。 */
  max?: number;
  /** 是否允许确认空选择。 */
  allowEmpty?: boolean;
  searchPlaceholder?: string;
  emptyText?: string;
}

export interface EvolynMemberDepartmentRolePickerEmits {
  (event: 'confirm', value: EvolynMemberDepartmentRolePickerSelection[]): void;
  (event: 'cancel'): void;
  (event: 'close', reason: 'cancel' | 'close' | 'overlay'): void;
}
