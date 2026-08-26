import {
  RiBookletFill,
  RiBriefcase4Fill,
  RiContactsBook3Fill,
  RiFileList3Fill,
  RiFolder3Fill,
  RiOrganizationChart,
  RiPieChart2Fill,
  RiShoppingBag3Fill,
  RiUserSettingsFill,
} from '@remixicon/vue';
import { computed, reactive, shallowRef } from 'vue';
import type {
  ManagementGroup,
  PermissionMember,
  PermissionNode,
  PermissionQueryView,
  PermissionSubjectType,
  PermissionWorkspace,
  SubjectNode,
} from '~/components/tenant/permissionQuery/permissionQuery.types';

const members: PermissionMember[] = [{ id: 'member-li', name: '李同学', department: '研发部' }];

const managementGroups = reactive<ManagementGroup[]>([
  {
    id: 'system-administrator',
    name: '系统管理组',
    type: '系统管理组',
    members: [],
    applicationScope: '全部应用',
  },
  { id: 'test', name: '测试', type: '普通管理组', members: [], applicationScope: '' },
]);

const subjectTrees: Record<PermissionSubjectType, SubjectNode[]> = {
  member: [{ id: 'member-li', label: '李同学', icon: RiUserSettingsFill }],
  department: [
    {
      id: 'dept-root',
      label: '重庆万柯互联网科技有限责任公司',
      icon: RiOrganizationChart,
      expanded: true,
      children: [
        { id: 'dept-rd', label: '研发部', icon: RiOrganizationChart },
        { id: 'dept-product', label: '产品部', icon: RiOrganizationChart },
      ],
    },
  ],
  role: [
    {
      id: 'role-default',
      label: '默认',
      icon: RiFolder3Fill,
      expanded: true,
      children: [{ id: 'role-test', label: '测试', icon: RiUserSettingsFill }],
    },
    {
      id: 'role-crm',
      label: 'CRM角色组',
      icon: RiFolder3Fill,
      expanded: true,
      children: [
        { id: 'role-sales-director', label: '销售总监', icon: RiUserSettingsFill },
        { id: 'role-sales-manager', label: '销售主管', icon: RiUserSettingsFill },
        { id: 'role-sales', label: '销售人员', icon: RiUserSettingsFill },
      ],
    },
  ],
  application: [
    {
      id: 'app-demo',
      label: '灵衍云示例应用',
      icon: RiBookletFill,
      expanded: true,
      children: [
        { id: 'app-orders', label: '订单管理', icon: RiBriefcase4Fill },
        { id: 'app-purchase', label: '采购申请', icon: RiShoppingBag3Fill },
        { id: 'app-employee', label: '员工档案', icon: RiFileList3Fill },
        { id: 'app-product', label: '产品管理', icon: RiContactsBook3Fill },
        { id: 'app-customer', label: '客户信息', icon: RiUserSettingsFill },
        { id: 'app-employee-analysis', label: '员工信息分析', icon: RiPieChart2Fill },
        { id: 'app-order-analysis', label: '订单分析', icon: RiPieChart2Fill },
      ],
    },
    { id: 'app-contract', label: '合同管理', icon: RiBookletFill },
    { id: 'app-it', label: 'IT项目管理', icon: RiBookletFill },
    { id: 'app-task', label: '任务管理', icon: RiBookletFill },
    { id: 'app-intro', label: '灵衍云高级功能介绍', icon: RiBookletFill },
    { id: 'app-crm', label: 'CRM_云星辰', icon: RiBookletFill },
  ],
};

const permissionTree: PermissionNode[] = [
  {
    id: 'intro',
    label: '灵衍云高级功能介绍',
    icon: RiBookletFill,
    expanded: true,
    children: [
      {
        id: 'print',
        label: '2. 自定义打印',
        icon: RiFolder3Fill,
        expanded: true,
        children: [
          {
            id: 'entry',
            label: '入职登记表',
            icon: RiFileList3Fill,
            expanded: true,
            children: [{ id: 'all-data', label: '管理全部数据', icon: RiFileList3Fill }],
          },
        ],
      },
      { id: 'submit', label: '3. 提交提示', icon: RiFolder3Fill },
    ],
  },
];

/**
 * 权限查询在权限接口上线前使用稳定的演示树与本地可编辑状态。
 * 页面只消费这里的状态和操作，后续接入接口时可在此统一替换数据源。
 */
export function usePermissionQuery() {
  const workspace = shallowRef<PermissionWorkspace>('system');
  const view = shallowRef<PermissionQueryView>('management-groups');
  const subjectType = shallowRef<PermissionSubjectType>('member');
  const selectedSubjectId = shallowRef('member-li');
  const selectedGroupId = shallowRef('system-administrator');
  const drawerVisible = shallowRef(false);
  const memberPickerVisible = shallowRef(false);

  const selectedGroup = computed(
    () =>
      managementGroups.find((group) => group.id === selectedGroupId.value) ?? managementGroups[0],
  );

  function selectGroup(id: string) {
    selectedGroupId.value = id;
    drawerVisible.value = true;
  }

  function saveMembers(nextMembers: PermissionMember[]) {
    selectedGroup.value.members = nextMembers;
    memberPickerVisible.value = false;
  }

  function selectSubject(type: PermissionSubjectType, id: string) {
    subjectType.value = type;
    selectedSubjectId.value = id;
  }

  return {
    drawerVisible,
    managementGroups,
    memberPickerVisible,
    members,
    permissionTree,
    saveMembers,
    selectedGroup,
    selectedGroupId,
    selectedSubjectId,
    selectGroup,
    selectSubject,
    subjectTrees,
    subjectType,
    view,
    workspace,
  };
}
