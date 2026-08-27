<script setup lang="ts">
import {
  EvolynMemberDepartmentRolePicker,
  type EvolynMemberDepartmentRolePickerMember,
  type EvolynMemberDepartmentRolePickerSelection,
} from '@evolyn.do/ui';
import { ElCheckbox, ElMessage, ElMessageBox } from 'element-plus';
import { RiCloseFill } from '@remixicon/vue';
import { computed, defineComponent, h, onMounted, shallowRef, watch } from 'vue';
import { updateMyTenantProfile } from '~/api/tenant';
import OrganizationInviteMemberDialog from '~/components/tenant/organization/OrganizationInviteMemberDialog.vue';
import OrganizationMemberDrawer from '~/components/tenant/organization/OrganizationMemberDrawer.vue';
import OrganizationMembersTable from '~/components/tenant/organization/OrganizationMembersTable.vue';
import OrganizationTreeSidebar from '~/components/tenant/organization/OrganizationTreeSidebar.vue';
import WorkHandoverDialog from '~/components/tenant/organization/WorkHandoverDialog.vue';
import type {
  OrganizationDepartment,
  OrganizationMember,
  OrganizationRole,
  OrganizationRoleGroup,
  OrganizationSelection,
  WorkHandoverSelection,
} from '~/components/tenant/organization/organization.types';
import organizationInviteIllustration from '~/assets/images/organization-invite-illustration.png';
import { useOrganization } from '~/composables/tenant/useOrganization';
import { useAuth } from '~/composables/auth';

defineOptions({ name: 'TenantOrganizationPage' });

const DISABLE_REMINDER_STORAGE_KEY = 'evolyn.organization.disable-reminder-date';

const {
  mode,
  selection,
  departments,
  roleGroups,
  roles,
  members,
  availableMembers,
  memberTotal,
  memberPage,
  filters,
  filteredMembers,
  switchMode,
  selectItem,
  addMembers,
  removeMember,
  updateMember,
  loadMembers,
  loadAvailableMembers,
  loadRoles,
  changeMemberStatus,
  setMemberPage,
  createRoleGroup,
  createRole,
  renameRole,
  renameRoleGroup,
  deleteRoleGroup,
  reorderRoleGroups,
  moveRole,
  deleteRole,
  reorderRoles,
  loadDepartments,
  setTenantRootName,
  renameDepartment,
  createChildDepartment,
} = useOrganization();
const { userInfo, loadUserInfo } = useAuth();
const tenantOwnerAccountId = computed(() =>
  userInfo.value?.tenant.ownerAccountId === null ||
  userInfo.value?.tenant.ownerAccountId === undefined
    ? null
    : String(userInfo.value.tenant.ownerAccountId),
);

const inviteBannerVisible = shallowRef(true);
const inviteDialogVisible = shallowRef(false);
const memberPickerVisible = shallowRef(false);
const memberEditorVisible = shallowRef(false);
const editingMember = shallowRef<OrganizationMember | null>(null);
const workHandoverVisible = shallowRef(false);
const workHandoverMember = shallowRef<OrganizationMember | null>(null);
const workHandoverSelection = shallowRef<WorkHandoverSelection | null>(null);
const recipientPickerVisible = shallowRef(false);
const recipientSelections = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);
const disableReminderDate = shallowRef(
  window.localStorage.getItem(DISABLE_REMINDER_STORAGE_KEY) ?? '',
);
const roleDialogVisible = shallowRef(false);
const roleDialogMode = shallowRef<'group' | 'role' | 'rename' | 'group-rename'>('role');
const roleNameDraft = shallowRef('');
const roleGroupTarget = shallowRef<OrganizationRoleGroup | null>(null);
const groupAdjustVisible = shallowRef(false);
const targetGroupId = shallowRef('');
const departmentDialogVisible = shallowRef(false);
const departmentDialogMode = shallowRef<'rename' | 'create-child'>('rename');
const departmentNameDraft = shallowRef('');
const departmentTarget = shallowRef<OrganizationDepartment | null>(null);
const departmentSubmitting = shallowRef(false);
// 左侧组织树的初始宽度；由 Splitter 在拖动时维护当前像素值。
const organizationSidebarWidth = shallowRef(356);

const roleNamesForEditor = computed(() => editingMember.value?.roleNames ?? []);
const pickerDepartments = computed(() => [
  {
    id: departments.value.id,
    label: departments.value.name,
    children: departments.value.children?.map((department) => ({
      id: department.id,
      label: department.name,
    })),
  },
]);
const pickerMembers = computed(() =>
  availableMembers.value.map((member) => ({
    id: member.id,
    label: member.name,
    departmentIds: member.departmentIds,
    keywords: [member.phone, member.email].filter((value): value is string => Boolean(value)),
  })),
);
const selectedMemberIds = computed(() => filteredMembers.value.map((member) => member.id));
const handoverCandidates = computed<EvolynMemberDepartmentRolePickerMember[]>(() =>
  availableMembers.value.map((member) => ({
    id: member.id,
    label: member.name,
    departmentIds: member.departmentIds,
    avatarUrl: member.avatar,
    keywords: [member.phone, member.email].filter((value): value is string => Boolean(value)),
    // 接交人不能是工作原持有人，前端提前排除无效操作，服务端仍需同样校验。
    disabled: member.id === workHandoverMember.value?.id,
  })),
);

function updateMode(nextMode: 'department' | 'role') {
  switchMode(nextMode);
}

function chooseItem(nextSelection: OrganizationSelection) {
  if (
    nextSelection.mode === 'role' &&
    roleGroups.value.some((group) => group.id === nextSelection.id)
  )
    return;
  selectItem(nextSelection);
}

function openCreateDialog(kind: 'group' | 'role', group?: OrganizationRoleGroup) {
  roleDialogMode.value = kind;
  roleGroupTarget.value = group ?? null;
  roleNameDraft.value = '';
  roleDialogVisible.value = true;
}

function openDepartmentDialog(
  action: 'rename' | 'create-child',
  department: OrganizationDepartment,
) {
  departmentDialogMode.value = action;
  departmentTarget.value = department;
  departmentNameDraft.value = action === 'rename' ? department.name : '';
  departmentDialogVisible.value = true;
}

async function confirmDepartmentDialog() {
  const name = departmentNameDraft.value.trim();
  const target = departmentTarget.value;
  if (!name || !target) return;
  departmentSubmitting.value = true;
  try {
    if (departmentDialogMode.value === 'rename') {
      if (target.isTenantRoot) {
        await updateMyTenantProfile({ name });
        const info = await loadUserInfo();
        setTenantRootName(info?.tenant.name ?? name);
      } else {
        await renameDepartment(target, name);
      }
    } else {
      await createChildDepartment(target.id, name);
    }
    departmentDialogVisible.value = false;
    ElMessage.success(departmentDialogMode.value === 'rename' ? '名称已修改' : '子部门已添加');
  } finally {
    departmentSubmitting.value = false;
  }
}

function openRenameDialog() {
  roleDialogMode.value = 'rename';
  roleGroupTarget.value = null;
  roleNameDraft.value = selection.value.name;
  roleDialogVisible.value = true;
}

function handleRoleGroupAction(
  action: 'rename' | 'add-role' | 'delete',
  group: OrganizationRoleGroup,
) {
  if (action === 'add-role') {
    openCreateDialog('role', group);
    return;
  }
  if (action === 'rename') {
    roleDialogMode.value = 'group-rename';
    roleGroupTarget.value = group;
    roleNameDraft.value = group.name;
    roleDialogVisible.value = true;
    return;
  }
  void confirmDeleteRoleGroup(group).catch(() => undefined);
}

async function confirmDeleteRoleGroup(group: OrganizationRoleGroup) {
  await ElMessageBox.confirm(
    `删除「${group.name}」后，组内角色将移至默认角色组，角色权限和成员关系保持不变。`,
    '删除角色组',
    { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
  );
  await deleteRoleGroup(group);
  ElMessage.success('角色组已删除');
}

async function handleRoleGroupReorder(groupIds: string[]) {
  await reorderRoleGroups(groupIds);
  ElMessage.success('角色组排序已保存');
}

function handleRoleAction(action: 'rename' | 'adjust-group' | 'delete', role: OrganizationRole) {
  selectItem({ mode: 'role', id: role.id, name: role.name });
  if (action === 'rename') {
    openRenameDialog();
    return;
  }
  if (action === 'adjust-group') {
    openGroupAdjust();
    return;
  }
  void confirmDeleteRole(role).catch(() => undefined);
}

async function confirmDeleteRole(role: OrganizationRole) {
  await ElMessageBox.confirm(`确定删除角色「${role.name}」吗？`, '删除角色', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  });
  await deleteRole(role);
  await loadMembers();
  ElMessage.success('角色已删除');
}

async function handleRoleReorder(groupId: string, roleIds: string[]) {
  await reorderRoles(groupId, roleIds);
  ElMessage.success('角色排序已保存');
}

function openGroupAdjust() {
  targetGroupId.value =
    roles.value.find((role) => role.id === selection.value.id)?.groupId ??
    roleGroups.value[0]?.id ??
    '';
  groupAdjustVisible.value = true;
}

async function confirmRoleDialog() {
  const name = roleNameDraft.value.trim();
  if (!name) return;
  if (roleDialogMode.value === 'group') await createRoleGroup(name);
  if (roleDialogMode.value === 'role') await createRole(name, roleGroupTarget.value?.id);
  if (roleDialogMode.value === 'rename') await renameRole(name);
  if (roleDialogMode.value === 'group-rename' && roleGroupTarget.value)
    await renameRoleGroup(roleGroupTarget.value, name);
  roleDialogVisible.value = false;
  roleGroupTarget.value = null;
  ElMessage.success('已保存');
}

async function confirmGroupAdjust() {
  await moveRole(targetGroupId.value);
  groupAdjustVisible.value = false;
  ElMessage.success('角色分组已调整');
}

function openMemberPicker() {
  memberPickerVisible.value = true;
}
async function confirmMemberPicker(values: { id: string | number }[]) {
  await addMembers(values.map((value) => String(value)));
  ElMessage.success('成员已添加到角色');
}
function openMemberEditor(member: OrganizationMember) {
  editingMember.value = member;
  memberEditorVisible.value = true;
}

function openWorkHandover(member: OrganizationMember) {
  workHandoverMember.value = member;
  workHandoverSelection.value = null;
  recipientSelections.value = [];
  workHandoverVisible.value = true;
}

function openRecipientPicker(selection: WorkHandoverSelection) {
  workHandoverSelection.value = selection;
  recipientSelections.value = [];
  recipientPickerVisible.value = true;
}

function confirmHandoverRecipient(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  const recipient = selections[0];
  if (!recipient) return;
  if (String(recipient.id) === workHandoverMember.value?.id) {
    ElMessage.warning('无法转交给本人');
    return;
  }
  // 当前后端尚未提供交接预览和提交接口；保留完整选择快照，避免伪造已完成的业务操作。
  const workCount = workHandoverSelection.value?.roleIds.length ?? 0;
  ElMessage.info(
    `已选择 ${recipient.label} 作为接交人${workCount ? `，包含 ${workCount} 个角色` : ''}；交接能力待后端接口接入后执行`,
  );
}

function currentDateKey() {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai' }).format(new Date());
}

async function disableMember(member: OrganizationMember) {
  const today = currentDateKey();
  const shouldSkipReminder = disableReminderDate.value === today;
  const doNotRemindToday = shallowRef(false);
  // MessageBox 接收的是 VNode；以一个轻量渲染组件承载复选框，才能保持其受控状态响应式更新。
  const DisableConfirmation = defineComponent({
    name: 'OrganizationMemberDisableConfirmation',
    setup() {
      return () =>
        h('div', { class: 'organization-member-disable-confirmation' }, [
          h('p', { class: 'organization-member-disable-confirmation__message' }, [
            '停用后，该成员无法访问本企业。',
          ]),
          h(
            ElCheckbox,
            {
              modelValue: doNotRemindToday.value,
              'onUpdate:modelValue': (value: unknown) => {
                doNotRemindToday.value = Boolean(value);
              },
            },
            { default: () => '今日不再提示' },
          ),
        ]);
    },
  });

  if (!shouldSkipReminder) {
    try {
      await ElMessageBox.confirm(h(DisableConfirmation), '停用成员', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        closeOnClickModal: false,
        customClass: 'organization-member-disable-message-box',
        type: 'warning',
      });
    } catch {
      return;
    }
  }

  if (doNotRemindToday.value) {
    disableReminderDate.value = today;
    window.localStorage.setItem(DISABLE_REMINDER_STORAGE_KEY, today);
  }
  await changeMemberStatus(member, 'disabled');
  ElMessage.success('成员已停用');
}

function saveMember(member: OrganizationMember) {
  updateMember(member);
  ElMessage.success('成员信息已保存');
}
async function handleRemoveMember(member: OrganizationMember) {
  if (mode.value === 'role') {
    await removeMember(member.id);
    ElMessage.success('成员已移出角色');
    return;
  }
  if (member.accountId === tenantOwnerAccountId.value) {
    ElMessage.warning('企业创建者无法转为离职');
    return;
  }
  await changeMemberStatus(member, 'resigned');
  ElMessage.success('成员已转为离职');
}
function exportMembers() {
  ElMessage.success('成员导出任务已创建');
}
function inviteMember() {
  inviteDialogVisible.value = true;
}

onMounted(async () => {
  try {
    const info = userInfo.value ?? (await loadUserInfo());
    await Promise.all([
      loadDepartments(info?.tenant.name ?? departments.value.name),
      loadRoles(),
      loadAvailableMembers(),
    ]);
  } catch {
    // 任一组织数据读取失败不阻断成员列表，页面仍保留已成功加载的部分数据。
  }
  await loadMembers();
});

// 部门、离职成员入口、状态筛选和翻页均由服务端完成，避免前端仅对当前页做伪筛选。
watch(
  () => [mode.value, selection.value.id, filters.status, memberPage.value],
  () => void loadMembers(),
);

// 关键词输入使用短暂延迟，减少连续输入时的无效请求；切换筛选条件会清理旧定时器。
watch(
  () => filters.memberKeyword,
  (_keyword, _previous, onCleanup) => {
    const timer = window.setTimeout(() => {
      memberPage.value = 1;
      void loadMembers();
    }, 250);
    onCleanup(() => window.clearTimeout(timer));
  },
);
</script>

<template>
  <section class="tenant-organization-page" aria-label="内部组织">
    <el-splitter class="tenant-organization-page__splitter">
      <el-splitter-panel v-model:size="organizationSidebarWidth" :min="300" :max="560">
        <OrganizationTreeSidebar
          :mode="mode"
          :selection="selection"
          :departments="departments"
          :role-groups="roleGroups"
          :roles="roles"
          @update:mode="updateMode"
          @select="chooseItem"
          @department-action="openDepartmentDialog"
          @create="openCreateDialog"
          @role-group-action="handleRoleGroupAction"
          @role-group-reorder="handleRoleGroupReorder"
          @role-action="handleRoleAction"
          @role-reorder="handleRoleReorder"
        />
      </el-splitter-panel>

      <el-splitter-panel :min="640">
        <main class="tenant-organization-page__content">
          <template v-if="mode === 'department'">
            <section
              v-if="inviteBannerVisible"
              class="tenant-organization-page__invite-banner"
              aria-label="邀请成员"
            >
              <img :src="organizationInviteIllustration" alt="成员协作插画" />
              <strong>邀请成员加入 一起管理企业业务</strong>
              <button type="button" @click="inviteMember">邀请成员</button>
              <button
                class="tenant-organization-page__banner-close"
                type="button"
                aria-label="关闭邀请横幅"
                @click="inviteBannerVisible = false"
              >
                <RiCloseFill />
              </button>
            </section>
            <header class="tenant-organization-page__title-row">
              <h1>{{ selection.name }}</h1>
            </header>
          </template>
          <template v-else>
            <header class="tenant-organization-page__role-header">
              <h1>{{ selection.name }}</h1>
              <div>
                <button type="button" @click="openRenameDialog">修改名称</button
                ><span aria-hidden="true">|</span
                ><button type="button" @click="openGroupAdjust">调整分组</button>
              </div>
            </header>
          </template>
          <OrganizationMembersTable
            :mode="mode"
            :members="filteredMembers"
            :keyword="filters.memberKeyword"
            :status="filters.status"
            :total="memberTotal"
            :current-page="memberPage"
            :tenant-owner-account-id="tenantOwnerAccountId"
            @update:keyword="filters.memberKeyword = $event"
            @update:status="filters.status = $event"
            @update:page="setMemberPage"
            @invite="inviteMember"
            @add-member="openMemberPicker"
            @import="inviteMember"
            @export="exportMembers"
            @edit="openMemberEditor"
            @handover="openWorkHandover"
            @disable="disableMember"
            @remove="handleRemoveMember"
          />
        </main>
      </el-splitter-panel>
    </el-splitter>

    <EvolynMemberDepartmentRolePicker
      v-model:open="memberPickerVisible"
      title="添加成员"
      :departments="pickerDepartments"
      :members="pickerMembers"
      :selectable-types="['member']"
      :model-value="
        selectedMemberIds.map((id) => ({
          id,
          label: members.find((member) => member.id === id)?.name ?? '',
          type: 'member',
        }))
      "
      @confirm="confirmMemberPicker"
    />
    <OrganizationMemberDrawer
      v-model="memberEditorVisible"
      :member="editingMember"
      :role-names="roleNamesForEditor"
      @save="saveMember"
    />
    <OrganizationInviteMemberDialog
      v-model="inviteDialogVisible"
      :departments="departments"
      @completed="loadMembers"
    />
    <WorkHandoverDialog
      v-model="workHandoverVisible"
      :member="workHandoverMember"
      :roles="roles"
      @choose-recipient="openRecipientPicker"
    />
    <EvolynMemberDepartmentRolePicker
      v-model:open="recipientPickerVisible"
      v-model="recipientSelections"
      title="选择接交人"
      :departments="pickerDepartments"
      :members="handoverCandidates"
      :selectable-types="['member']"
      :member-multiple="false"
      :max="1"
      @confirm="confirmHandoverRecipient"
    />
    <el-dialog
      v-model="departmentDialogVisible"
      :title="departmentDialogMode === 'rename' ? '修改部门名称' : '添加子部门'"
      width="480px"
      align-center
      @closed="departmentTarget = null"
    >
      <p
        v-if="departmentDialogMode === 'create-child'"
        class="tenant-organization-page__dialog-prompt"
      >
        将在「{{ departmentTarget?.name }}」下创建子部门
      </p>
      <el-input
        v-model="departmentNameDraft"
        :placeholder="departmentDialogMode === 'rename' ? '请输入部门名称' : '请输入子部门名称'"
        maxlength="30"
        show-word-limit
        @keyup.enter="confirmDepartmentDialog"
      />
      <template #footer
        ><el-button @click="departmentDialogVisible = false">取消</el-button
        ><el-button type="primary" :loading="departmentSubmitting" @click="confirmDepartmentDialog"
          >确定</el-button
        ></template
      >
    </el-dialog>
    <el-dialog
      v-model="roleDialogVisible"
      :title="
        roleDialogMode === 'group'
          ? '创建角色组'
          : roleDialogMode === 'rename' || roleDialogMode === 'group-rename'
            ? '修改名称'
            : '创建角色'
      "
      width="480px"
      align-center
      @closed="roleGroupTarget = null"
    >
      <el-input
        v-model="roleNameDraft"
        :placeholder="
          roleDialogMode === 'group' || roleDialogMode === 'group-rename'
            ? '请输入角色组名称'
            : '请输入角色名称'
        "
        maxlength="30"
        show-word-limit
      />
      <template #footer
        ><el-button @click="roleDialogVisible = false">取消</el-button
        ><el-button type="primary" @click="confirmRoleDialog">确定</el-button></template
      >
    </el-dialog>
    <el-dialog v-model="groupAdjustVisible" title="调整分组" width="620px" align-center>
      <p class="tenant-organization-page__dialog-prompt">请选择目标分组</p>
      <div class="tenant-organization-page__group-list">
        <button
          v-for="group in roleGroups"
          :key="group.id"
          :class="{ 'tenant-organization-page__group-item--active': targetGroupId === group.id }"
          type="button"
          @click="targetGroupId = group.id"
        >
          <span>{{ group.name }}</span
          ><span v-if="targetGroupId === group.id">●</span>
        </button>
      </div>
      <template #footer
        ><el-button @click="groupAdjustVisible = false">取消</el-button
        ><el-button type="primary" @click="confirmGroupAdjust">确定</el-button></template
      >
    </el-dialog>
  </section>
</template>

<style scoped lang="scss">
.tenant-organization-page {
  position: relative;
  height: 100%;
  min-height: 720px;
  overflow: hidden;
  background: var(--el-bg-color);
}
.tenant-organization-page__splitter {
  height: 100%;
}
.tenant-organization-page__content {
  display: flex;
  height: 100%;
  min-width: 0;
  flex-direction: column;
}
.tenant-organization-page__invite-banner {
  position: relative;
  display: flex;
  height: 136px;
  margin: var(--el-space-xl) var(--el-space-4xl) 0;
  padding: 0 80px 0 300px;
  overflow: hidden;
  border-radius: var(--el-border-radius-base);
  align-items: center;
  background: var(--el-color-primary);
}
.tenant-organization-page__invite-banner img {
  position: absolute;
  bottom: -21px;
  left: -8px;
  width: 360px;
  height: 164px;
  object-fit: contain;
}
.tenant-organization-page__invite-banner strong {
  position: relative;
  color: var(--el-color-white);
  font-size: clamp(var(--el-font-size-extra-large), 1.65vw, 30.0006px);
  font-weight: 600;
  letter-spacing: 1px;
  white-space: nowrap;
}
.tenant-organization-page__invite-banner > button:not(.tenant-organization-page__banner-close) {
  min-width: 146px;
  height: 44px;
  margin-left: auto;
  border: 0;
  border-radius: 24px;
  color: var(--el-color-primary);
  background: var(--el-bg-color);
  cursor: pointer;
  font: inherit;
  font-size: var(--el-font-size-medium);
}
.tenant-organization-page__invite-banner
  > button:not(.tenant-organization-page__banner-close):hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.tenant-organization-page__banner-close {
  position: absolute;
  top: 10px;
  right: 10px;
  display: inline-flex;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: var(--el-border-radius-base);
  align-items: center;
  justify-content: center;
  color: var(--el-color-white);
  background: transparent;
  cursor: pointer;
  opacity: 0.8;
}
.tenant-organization-page__banner-close:hover {
  background: var(--el-color-primary-light-3);
  opacity: 1;
}
.tenant-organization-page__banner-close svg {
  width: 22px;
  height: 22px;
}
.tenant-organization-page__title-row,
.tenant-organization-page__role-header {
  display: flex;
  height: 66px;
  padding: 0 var(--el-space-4xl);
  border-bottom: 1px solid var(--el-border-color-lighter);
  align-items: center;
  justify-content: space-between;
}
.tenant-organization-page__title-row h1,
.tenant-organization-page__role-header h1 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-extra-large);
  font-weight: 650;
}
.tenant-organization-page__role-header > div {
  display: flex;
  align-items: center;
  gap: var(--el-space-lg);
  color: var(--el-border-color);
}
.tenant-organization-page__role-header button {
  padding: var(--el-space-xs) var(--el-space-sm);
  border: 0;
  border-radius: var(--el-border-radius-base);
  color: var(--el-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--el-font-size-medium);
}
.tenant-organization-page__role-header button:hover {
  background: var(--el-color-primary-light-9);
}
.tenant-organization-page__dialog-prompt {
  margin: 0 0 var(--el-space-lg);
  color: var(--el-text-color-secondary);
}
.tenant-organization-page__group-list {
  min-height: 260px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-medium);
  overflow: hidden;
}
.tenant-organization-page__group-list button {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  height: 46px;
  padding: 0 var(--el-space-xl);
  border: 0;
  align-items: center;
  justify-content: space-between;
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  text-align: left;
}
.tenant-organization-page__group-list button:hover {
  background: var(--el-fill-color-light);
}
.tenant-organization-page__group-item--active {
  color: var(--el-color-primary) !important;
  background: var(--el-color-primary-light-9) !important;
}

// 接交人选择器与交接抽屉都传送到 body。抽屉遮罩由 Element Plus 分配更高层级，
// 因此仅在二者并存时提升紧随其后的选择器，避免影响其他页面的通用选择器。
:global(.work-handover-dialog__modal ~ .evolyn-member-department-role-picker) {
  z-index: 3000;
}

@media (max-width: 1200px) {
  .tenant-organization-page__invite-banner {
    padding-left: 270px;
  }
  .tenant-organization-page__invite-banner img {
    width: 300px;
  }
  .tenant-organization-page__invite-banner strong {
    font-size: var(--el-font-size-extra-large);
  }
}
</style>
