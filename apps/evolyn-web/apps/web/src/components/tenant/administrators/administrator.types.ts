export type AdministratorScope = 'system' | 'application';
export type ScopeMode = 'all' | 'partial';

export interface AdministratorMember {
  id: string;
  name: string;
  department: string;
}

export interface AdministratorApplication {
  id: string;
  name: string;
  icon: string;
  tone: 'green' | 'coral' | 'blue' | 'cyan' | 'purple' | 'orange';
}

export interface AdministratorGroup {
  id: string;
  name: string;
  builtIn?: boolean;
  members: AdministratorMember[];
  departmentEnabled: boolean;
  departmentMode: ScopeMode;
  departmentIds: string[];
  roleVisible: boolean;
  roleManage: boolean;
  roleMode: ScopeMode;
  roleIds: string[];
  externalEnabled: boolean;
  applicationIds: string[];
  applicationManage: boolean;
}

export const administratorMembers: AdministratorMember[] = [
  { id: 'member-lisi', name: '李同学', department: '研发部' },
  { id: 'member-zhangsan', name: '张三', department: '产品部' },
  { id: 'member-wangwu', name: '王五', department: '研发部' },
];

export const administratorApplications: AdministratorApplication[] = [
  { id: 'app-demo', name: '灵衍云示例应用', icon: '▾', tone: 'green' },
  { id: 'app-contract', name: '合同管理', icon: '▰', tone: 'coral' },
  { id: 'app-it', name: 'IT项目管理', icon: '▣', tone: 'blue' },
  { id: 'app-task', name: '任务管理', icon: '▱', tone: 'cyan' },
  { id: 'app-intro', name: '灵衍云高级功能介绍', icon: '▤', tone: 'purple' },
  { id: 'app-crm-star', name: 'CRM_云星辰', icon: 'CRM', tone: 'coral' },
  { id: 'app-crm-suite', name: 'CRM_云星辰多账套', icon: 'CRM', tone: 'coral' },
  { id: 'app-crm', name: 'CRM_精斗云', icon: 'CRM', tone: 'coral' },
  { id: 'app-test', name: '测试应用', icon: '◆', tone: 'orange' },
];

export const departments = [
  { id: 'department-rd', name: '研发部' },
  { id: 'department-product', name: '产品部' },
];

export const roles = [{ id: 'role-test', name: '测试' }];
