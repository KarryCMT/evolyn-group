<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, shallowRef } from 'vue';
import PermissionAssetList from '~/components/application/permissions/PermissionAssetList.vue';
import PermissionGroupsPanel from '~/components/application/permissions/PermissionGroupsPanel.vue';
import PermissionMemberPickerDialog from '~/components/application/permissions/PermissionMemberPickerDialog.vue';
import type {
  AssetPermissionGroup,
  CreatePermissionGroupPayload,
  PermissionAsset,
  PermissionSubject,
} from '~/components/application/permissions/permission.types';

defineOptions({ name: 'ApplicationSettingPermissionsPage' });

/**
 * 权限 API 尚未落地，页面先用本地预览数据完成交互与视觉验收。
 * 数据源接入后仅替换此处状态装载与操作方法，子组件仍保持 props down / events up。
 */
const assets = shallowRef<PermissionAsset[]>([
  { id: 'form_order', name: '订单管理', type: 'workflow-form' },
  { id: 'form_purchase', name: '采购申请', type: 'workflow-form' },
  { id: 'form_office', name: '办公用品申请', type: 'workflow-form' },
  { id: 'form_employee', name: '员工档案', type: 'form' },
  { id: 'form_product', name: '产品管理', type: 'form' },
  { id: 'form_customer', name: '客户信息', type: 'form' },
  { id: 'dashboard_employee', name: '员工信息分析', type: 'dashboard' },
  { id: 'dashboard_order', name: '订单分析', type: 'dashboard' },
  { id: 'dashboard_customer', name: '客户信息分析', type: 'dashboard' },
]);

const selectableSubjects: PermissionSubject[] = [
  { id: 'department_sales', name: '销售部', type: 'department' },
  { id: 'department_operation', name: '运营部', type: 'department' },
  { id: 'role_sales_manager', name: '销售主管', type: 'role' },
  { id: 'member_zhangsan', name: '张三', type: 'member' },
  { id: 'member_lisi', name: '李四', type: 'member' },
  { id: 'member_wangwu', name: '王五', type: 'member' },
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

const selectedAsset = computed(() =>
  assets.value.find((asset) => asset.id === selectedAssetId.value),
);
const selectedGroups = computed(() => groupsByAssetId.value[selectedAssetId.value] ?? []);
const pickerTitle = computed(() => (targetGroupId.value ? '添加授权对象' : '添加成员'));

function updateGroups(assetId: string, groups: AssetPermissionGroup[]) {
  groupsByAssetId.value = { ...groupsByAssetId.value, [assetId]: groups };
}

function openGroupPicker() {
  targetGroupId.value = undefined;
  pickerVisible.value = true;
}

function openSubjectPicker(groupId: string) {
  targetGroupId.value = groupId;
  pickerVisible.value = true;
}

function createOrAppendSubjects(payload: CreatePermissionGroupPayload) {
  const asset = selectedAsset.value;
  if (!asset) return;

  const subjects = selectableSubjects.filter((subject) => payload.subjectIds.includes(subject.id));
  const groups = selectedGroups.value;
  if (targetGroupId.value) {
    updateGroups(
      asset.id,
      groups.map((group) =>
        group.id === targetGroupId.value
          ? {
              ...group,
              subjects: [
                ...group.subjects,
                ...subjects.filter(
                  (subject) => !group.subjects.some((item) => item.id === subject.id),
                ),
              ],
            }
          : group,
      ),
    );
    ElMessage.success('已添加授权对象');
    return;
  }

  const groupName = payload.groupName || defaultGroupName(asset.type);
  updateGroups(asset.id, [
    ...groups,
    {
      id: `group_preview_${Date.now()}`,
      name: groupName,
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
  <section class="application-setting-permissions" aria-label="表单和仪表盘权限">
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
    <PermissionMemberPickerDialog
      v-model="pickerVisible"
      :subjects="selectableSubjects"
      :title="pickerTitle"
      @confirm="createOrAppendSubjects"
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
}

@media (max-width: 920px) {
  .application-setting-permissions {
    min-width: 760px;
  }
}
</style>
