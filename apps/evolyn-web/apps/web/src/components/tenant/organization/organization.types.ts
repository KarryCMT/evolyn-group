export type OrganizationMode = 'department' | 'role';

export interface OrganizationDepartment {
  id: string;
  name: string;
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
  name: string;
  phone: string;
  email?: string;
  department: string;
  roleIds: string[];
  status: '已启用' | '已停用' | '已离职';
  employeeNo: string;
  alias: string;
  gender: string;
}

export interface OrganizationSelection {
  mode: OrganizationMode;
  id: string;
  name: string;
}
