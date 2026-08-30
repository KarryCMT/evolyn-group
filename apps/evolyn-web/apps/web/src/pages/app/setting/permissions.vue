<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus';
import {
  EvolynMemberDepartmentRolePicker,
  type EvolynMemberDepartmentRolePickerMember,
  type EvolynMemberDepartmentRolePickerSelection,
  type EvolynMemberDepartmentRolePickerTreeNode,
} from '@evolyn.do/ui';
import { computed, shallowRef } from 'vue';
import { useRoute } from 'vue-router';
import PermissionAssetList from '~/components/application/permissions/PermissionAssetList.vue';
import PermissionGroupsPanel from '~/components/application/permissions/PermissionGroupsPanel.vue';
import { useApplicationHome } from '~/composables/useApplicationHome';
import type {
  AssetPermissionGroup,
  PermissionAsset,
  PermissionSubject,
} from '~/components/application/permissions/permission.types';

defineOptions({ name: 'ApplicationSettingPermissionsPage' });

const route = useRoute();
const appCode = computed(() => String(route.params.appCode ?? ''));
const { application, errorMessage, reload, status } = useApplicationHome(appCode);

// 权限设置仅应用管理员可访问。后端应用级管理员体系尚未落地，先以应用详情
// 派生的 capabilities.edit（applications:patch）作为管理员口径；
// 应用级范围授权（AccessEvaluator 扩展）落地后替换判定即可。
const accessDenied = computed(
  () => status.value === 'ready' && !application.value?.capabilities.edit,
);

/**
 * 权限 API 尚未落地，页面先用本地预览数据完成交互与视觉验收。
 * 数据源接入后仅替换此处状态装载与操作方法，子组件仍保持 props down / events up。
 */
const assets = shallowRef<PermissionAsset[]>([
  { id: 'form_order', name: '订单管理', type: 'workflow-form' },
  { id: 'form_purchase', name: '采购申请', type: 'workflow-form' },
  { id: 'form_office', name: '办公用品申请', type: 'workflow-form' },
  {
    id: 'form_employee',
    name: '员工档案员工档案员工档案员工档案员工档案员工档案员工档案',
    type: 'form',
  },
  { id: 'form_product', name: '产品管理', type: 'form' },
  { id: 'form_customer', name: '客户信息', type: 'form' },
  { id: 'dashboard_employee', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_employee2', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_employee12', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_empl2oyee2', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_emplo3yee2', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_empl4oyee2', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_employee3', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_order', name: '订单分析', type: 'dashboard' },
  { id: 'dashboard_customer', name: '客户信息分析', type: 'dashboard' },
  {
    id: 'group_1',
    name: '分组1',
    type: 'group',
    children: [
      {
        id: 'group_2_order',
        name: '订单分析2',
        type: 'group',
        children: [{ id: 'form_product_group', name: '产品管理', type: 'form' }],
      },
      { id: 'form_employee1', name: '员工档案2', type: 'form' },
    ],
  },
]);

/** 选择器数据暂用本地演示结构；权限主体接口落地后替换为部门树、角色和成员接口响应。 */
const pickerDepartments: EvolynMemberDepartmentRolePickerTreeNode[] = [
  {
    id: 'department_company',
    label: '重庆万柯互联网科技有限责任公司',
    children: [
      { id: 'department_sales', label: '销售部' },
      { id: 'department_operation', label: '运营部' },
    ],
  },
];

const pickerRoles: EvolynMemberDepartmentRolePickerTreeNode[] = [
  { id: 'role_sales_manager', label: '销售主管' },
  { id: 'role_sales_director', label: '销售总监' },
];

const pickerMembers: EvolynMemberDepartmentRolePickerMember[] = [
  { id: 'member_zhangsan', label: '张三', departmentIds: ['department_sales'] },
  { id: 'member_lisi', label: '李四', departmentIds: ['department_sales'] },
  { id: 'member_wangwu', label: '王五', departmentIds: ['department_operation'] },
];

const groupsByAssetId = shallowRef<Record<string, AssetPermissionGroup[]>>({
  form_order: [
    {
      id: 'group_order_initiate',
      name: '发起流程',
      description: '此分组内的成员可以发起订单审批流程，并查看自己发起的流程。',
      enabled: true,
      subjects: [{ id: 'department_sales', name: '销售部', type: 'department' }],
    },
    {
      id: 'group_order_manage',
      name: '管理全部流程',
      description: '此分组内的成员可以查看、管理订单的全部流程数据。',
      enabled: true,
      subjects: [{ id: 'role_sales_manager', name: '销售主管', type: 'role' }],
    },
  ],
  form_employee: [
    {
      id: 'group_employee_manage',
      name: '管理全部数据',
      description: '此分组内的成员可以填报、查看和管理员工档案的全部数据。',
      enabled: true,
      subjects: [{ id: 'department_operation', name: '运营部', type: 'department' }],
    },
  ],
  dashboard_order: [
    {
      id: 'group_dashboard_order_view',
      name: '查看仪表盘',
      description: '此分组内的成员可以访问订单分析仪表盘；数据范围遵循图表配置。',
      enabled: true,
      subjects: [{ id: 'role_sales_manager', name: '销售主管', type: 'role' }],
    },
  ],
});

const selectedAssetId = shallowRef('form_order');
const keyword = shallowRef('');
const pickerVisible = shallowRef(false);
const targetGroupId = shallowRef<string>();
const pickerSelection = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);

/** 资产为树形结构，选中查找需沿 children 递归。 */
function findAssetById(list: PermissionAsset[], id: string): PermissionAsset | undefined {
  for (const asset of list) {
    if (asset.id === id) return asset;
    const found = asset.children ? findAssetById(asset.children, id) : undefined;
    if (found) return found;
  }
  return undefined;
}

const selectedAsset = computed(() => findAssetById(assets.value, selectedAssetId.value));
const selectedGroups = computed(() => groupsByAssetId.value[selectedAssetId.value] ?? []);
const pickerTitle = computed(() => (targetGroupId.value ? '添加授权对象' : '添加成员'));

function updateGroups(assetId: string, groups: AssetPermissionGroup[]) {
  groupsByAssetId.value = { ...groupsByAssetId.value, [assetId]: groups };
}

function openGroupPicker() {
  targetGroupId.value = undefined;
  pickerSelection.value = [];
  pickerVisible.value = true;
}

function openSubjectPicker(groupId: string) {
  targetGroupId.value = groupId;
  const group = selectedGroups.value.find((item) => item.id === groupId);
  pickerSelection.value = group?.subjects.map(toPickerSelection) ?? [];
  pickerVisible.value = true;
}

function toPickerSelection(subject: PermissionSubject): EvolynMemberDepartmentRolePickerSelection {
  return { id: subject.id, label: subject.name, type: subject.type };
}

function toPermissionSubject(
  selection: EvolynMemberDepartmentRolePickerSelection,
): PermissionSubject {
  return { id: String(selection.id), name: selection.label, type: selection.type };
}

function createOrUpdateSubjects(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  const asset = selectedAsset.value;
  if (!asset) return;

  const subjects = selections.map(toPermissionSubject);
  const groups = selectedGroups.value;
  if (targetGroupId.value) {
    updateGroups(
      asset.id,
      groups.map((group) =>
        group.id === targetGroupId.value
          ? {
              ...group,
              // 选择器打开时已带入当前主体，确认结果即为该权限组的最终主体列表。
              subjects,
            }
          : group,
      ),
    );
    ElMessage.success('已添加授权对象');
    return;
  }

  updateGroups(asset.id, [
    ...groups,
    {
      id: `group_preview_${Date.now()}`,
      name: defaultGroupName(asset.type),
      description: defaultGroupDescription(asset.type),
      enabled: true,
      subjects,
    },
  ]);
  ElMessage.success('已创建权限组');
}

function defaultGroupName(type: PermissionAsset['type']) {
  if (type === 'dashboard') return '查看仪表盘';
  if (type === 'workflow-form') return '发起流程';
  return '管理全部数据';
}

function defaultGroupDescription(type: PermissionAsset['type']) {
  if (type === 'dashboard') return '此分组内的成员可以访问该仪表盘；数据范围遵循图表配置。';
  if (type === 'workflow-form') return '此分组内的成员可以发起流程，并查看自己发起的流程。';
  return '此分组内的成员可以填报、查看和管理全部数据。';
}

function updateGroupEnabled(payload: { groupId: string; enabled: boolean }) {
  const asset = selectedAsset.value;
  if (!asset) return;
  updateGroups(
    asset.id,
    selectedGroups.value.map((group) =>
      group.id === payload.groupId ? { ...group, enabled: payload.enabled } : group,
    ),
  );
}

function cloneGroup(groupId: string) {
  const asset = selectedAsset.value;
  const group = selectedGroups.value.find((item) => item.id === groupId);
  if (!asset || !group) return;
  updateGroups(asset.id, [
    ...selectedGroups.value,
    { ...group, id: `group_preview_${Date.now()}`, name: `${group.name}（副本）` },
  ]);
  ElMessage.success('已复制权限组');
}

async function removeGroup(groupId: string) {
  const asset = selectedAsset.value;
  if (!asset) return;
  try {
    await ElMessageBox.confirm('删除后该权限组中的成员将立即失去对应权限。', '删除权限组', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    });
    updateGroups(
      asset.id,
      selectedGroups.value.filter((group) => group.id !== groupId),
    );
    ElMessage.success('已删除权限组');
  } catch {
    // 用户取消确认不需要反馈；后续接入 API 时在此处理网络异常。
  }
}

async function disableAll() {
  const asset = selectedAsset.value;
  if (!asset) return;
  try {
    await ElMessageBox.confirm('停用后，当前资产下所有成员权限都会暂时失效。', '停用全部权限', {
      confirmButtonText: '确认停用',
      cancelButtonText: '取消',
      type: 'warning',
    });
    updateGroups(
      asset.id,
      selectedGroups.value.map((group) => ({ ...group, enabled: false })),
    );
    ElMessage.success('已停用全部权限组');
  } catch {
    // 用户取消确认不需要反馈；后续接入 API 时在此处理网络异常。
  }
}
</script>

<template>
  <!-- 应用信息装载中：整页占位等待，避免准入判定闪烁。 -->
  <section
    v-if="status === 'loading'"
    v-loading="true"
    class="application-setting-permissions__status"
  />

  <el-result
    v-else-if="status === 'not-found'"
    class="application-setting-permissions__result"
    icon="warning"
    title="应用不存在或已不可访问"
    sub-title="请返回工作台后重新选择应用。"
  />

  <el-result
    v-else-if="status === 'error'"
    class="application-setting-permissions__result"
    icon="error"
    title="加载应用设置失败"
    :sub-title="errorMessage"
  >
    <template #extra>
      <el-button type="primary" @click="reload()">重新加载</el-button>
    </template>
  </el-result>

  <!-- 仅应用管理员可进入权限设置，普通成员呈现无权限状态。 -->
  <el-result
    v-else-if="accessDenied"
    class="application-setting-permissions__result"
    icon="warning"
    title="无访问权限"
    sub-title="仅应用管理员可管理表单和仪表盘权限。"
  />

  <section v-else class="application-setting-permissions" aria-label="表单和仪表盘权限">
    <PermissionAssetList
      :assets="assets"
      :keyword="keyword"
      :selected-asset-id="selectedAssetId"
      @batch-select="ElMessage.info('批量选择将在权限接口接入后开放')"
      @update-keyword="keyword = $event"
      @select="selectedAssetId = $event"
    />
    <PermissionGroupsPanel
      :asset="selectedAsset"
      :groups="selectedGroups"
      @add-group="openGroupPicker"
      @add-subjects="openSubjectPicker"
      @clone-group="cloneGroup"
      @disable-all="disableAll"
      @edit-group="ElMessage.info('权限编辑面板将在下一步接入')"
      @remove-group="removeGroup"
      @update-group-enabled="updateGroupEnabled"
    />
    <EvolynMemberDepartmentRolePicker
      v-model="pickerSelection"
      v-model:open="pickerVisible"
      :departments="pickerDepartments"
      :roles="pickerRoles"
      :members="pickerMembers"
      :title="pickerTitle"
      @confirm="createOrUpdateSubjects"
    />
  </section>
</template>

<style scoped lang="scss">
.application-setting-permissions {
  display: flex;
  height: 100%;
  width: 100%;
  min-width: 920px;
  min-height: 0;
  overflow: hidden;

  &__status,
  &__result {
    display: grid;
    min-height: 100%;
    place-items: center;
  }
}

@media (max-width: 920px) {
  .application-setting-permissions {
    min-width: 760px;
  }
}
</style>
