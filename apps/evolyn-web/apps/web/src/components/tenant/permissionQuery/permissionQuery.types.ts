import type { Component } from 'vue';

export type PermissionWorkspace = 'system' | 'jiandaoyun';
export type PermissionQueryView = 'management-groups' | 'permission-groups';
export type PermissionSubjectType = 'member' | 'department' | 'role' | 'application';

export interface PermissionMember {
  id: string;
  name: string;
  department: string;
}

export interface ManagementGroup {
  id: string;
  name: string;
  type: string;
  members: PermissionMember[];
  applicationScope?: string;
}

export interface PermissionNode {
  id: string;
  label: string;
  icon: Component;
  children?: PermissionNode[];
  expanded?: boolean;
}

export interface SubjectNode {
  id: string;
  label: string;
  icon: Component;
  children?: SubjectNode[];
  expanded?: boolean;
}
