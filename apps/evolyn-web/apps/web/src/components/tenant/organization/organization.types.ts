export type OrganizationMode = 'department' | 'role';

export interface OrganizationDepartment {
  id: string;
  name: string;
  /** 虚拟根节点对应当前租户，不写入 departments 表。 */
  isTenantRoot?: boolean;
  parentId?: string | null;
  order?: number;
  status?: 'active' | 'disabled';
  children?: OrganizationDepartment[];
}

export interface OrganizationRoleGroup {
  id: string;
  name: string;
}

export interface OrganizationRole {
  id: string;
  name: string;
  groupId: string;
}

export interface OrganizationMember {
  id: string;
  accountId: string;
  name: string;
  phone: string;
  email?: string;
  avatar?: string;
  department: string;
  departmentIds: string[];
  roleIds: string[];
  roleNames: string[];
  status: 'active' | 'disabled' | 'resigned';
  resignedAt?: string | null;
  employeeNo: string;
  alias: string;
  gender: string;
}

export interface OrganizationSelection {
  mode: OrganizationMode;
  id: string;
  name: string;
}

/** 交接抽屉确认后保留的资源选择快照，供后续交接接口提交。 */
export interface WorkHandoverSelection {
  categoryIds: string[];
  roleIds: string[];
}
