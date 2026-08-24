<script setup lang="ts">
import { EvolynMemberDepartmentRolePicker } from '@evolyn.do/ui';
import { ElMessage } from 'element-plus';
import { RiCloseFill } from '@remixicon/vue';
import { computed, onUnmounted, shallowRef } from 'vue';
import OrganizationMemberDrawer from '~/components/tenant/organization/OrganizationMemberDrawer.vue';
import OrganizationMembersTable from '~/components/tenant/organization/OrganizationMembersTable.vue';
import OrganizationTreeSidebar from '~/components/tenant/organization/OrganizationTreeSidebar.vue';
import type {
  OrganizationMember,
  OrganizationSelection,
} from '~/components/tenant/organization/organization.types';
import organizationInviteIllustration from '~/assets/images/organization-invite-illustration.png';
import { useOrganization } from '~/composables/tenant/useOrganization';

defineOptions({ name: 'TenantOrganizationPage' });

// 管理后台在现有设计中固定使用浅色表面；同时让 VTable 读取正确的根级主题变量。
const restoreDarkClass =
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark');
if (restoreDarkClass) document.documentElement.classList.remove('dark');
onUnmounted(() => {
  if (restoreDarkClass) document.documentElement.classList.add('dark');
});

const {
  mode,
  selection,
  departments,
  roleGroups,
  roles,
  members,
  filters,
  filteredMembers,
  switchMode,
  selectItem,
  roleName,
  addMembers,
  removeMember,
  updateMember,
  createRoleGroup,
  createRole,
  renameRole,
  moveRole,
} = useOrganization();

const inviteBannerVisible = shallowRef(true);
const memberPickerVisible = shallowRef(false);
const memberEditorVisible = shallowRef(false);
const editingMember = shallowRef<OrganizationMember | null>(null);
const roleDialogVisible = shallowRef(false);
const roleDialogMode = shallowRef<'group' | 'role' | 'rename'>('role');
const roleNameDraft = shallowRef('');
const groupAdjustVisible = shallowRef(false);
const targetGroupId = shallowRef('role-group-default');

const roleNamesForEditor = computed(
  () => editingMember.value?.roleIds.map(roleName).filter(Boolean) ?? [],
);
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
  members.value.map((member) => ({
    id: member.id,
    label: member.name,
    departmentIds: [
      member.department === departments.value.name ? departments.value.id : 'dept-rd',
    ],
    keywords: [member.phone, member.email].filter((value): value is string => Boolean(value)),
  })),
);
const selectedMemberIds = computed(() => filteredMembers.value.map((member) => member.id));

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

function openCreateDialog(kind: 'group' | 'role') {
  roleDialogMode.value = kind;
  roleNameDraft.value = '';
  roleDialogVisible.value = true;
}

function openRenameDialog() {
  roleDialogMode.value = 'rename';
  roleNameDraft.value = selection.value.name;
  roleDialogVisible.value = true;
}

function confirmRoleDialog() {
  const name = roleNameDraft.value.trim();
  if (!name) return;
  if (roleDialogMode.value === 'group') createRoleGroup(name);
  if (roleDialogMode.value === 'role') createRole(name);
  if (roleDialogMode.value === 'rename') renameRole(name);
  roleDialogVisible.value = false;
  ElMessage.success('已保存');
}

function confirmGroupAdjust() {
  moveRole(targetGroupId.value);
  groupAdjustVisible.value = false;
  ElMessage.success('角色分组已调整');
}

function openMemberPicker() {
  memberPickerVisible.value = true;
}
function confirmMemberPicker(values: { id: string | number }[]) {
  addMembers(values.map((value) => String(value)));
  ElMessage.success('成员已添加到角色');
}
function openMemberEditor(member: OrganizationMember) {
  editingMember.value = member;
  memberEditorVisible.value = true;
}
function saveMember(member: OrganizationMember) {
  updateMember(member);
  ElMessage.success('成员信息已保存');
}
function handleRemoveMember(member: OrganizationMember) {
  removeMember(member.id);
  ElMessage.success(mode.value === 'role' ? '成员已移出角色' : '成员已转为离职');
}
function exportMembers() {
  ElMessage.success('成员导出任务已创建');
}
function inviteMember() {
  ElMessage.success('邀请链接已生成');
}
</script>

<template>
  <section class="tenant-organization-page" aria-label="内部组织">
    <OrganizationTreeSidebar
      :mode="mode"
      :selection="selection"
      :departments="departments"
      :role-groups="roleGroups"
      :roles="roles"
      @update:mode="updateMode"
      @select="chooseItem"
      @create="openCreateDialog"
    />

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
            ><button type="button" @click="groupAdjustVisible = true">调整分组</button>
          </div>
        </header>
      </template>
      <OrganizationMembersTable
        :mode="mode"
        :role-name="selection.name"
        :members="filteredMembers"
        :role-name-of="roleName"
        :keyword="filters.memberKeyword"
        :status="filters.status"
        @update:keyword="filters.memberKeyword = $event"
        @update:status="filters.status = $event"
        @invite="inviteMember"
        @add-member="openMemberPicker"
        @import="ElMessage.success('导入功能即将接入')"
        @export="exportMembers"
        @edit="openMemberEditor"
        @remove="handleRemoveMember"
      />
    </main>

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
    <el-dialog
      v-model="roleDialogVisible"
      :title="
        roleDialogMode === 'group'
          ? '创建角色组'
          : roleDialogMode === 'rename'
            ? '修改名称'
            : '创建角色'
      "
      width="480px"
      align-center
    >
      <el-input
        v-model="roleNameDraft"
        :placeholder="roleDialogMode === 'group' ? '请输入角色组名称' : '请输入角色名称'"
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
  display: flex;
  height: 100%;
  min-height: 720px;
  overflow: hidden;
  background: #fff;
}
.tenant-organization-page__content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}
.tenant-organization-page__invite-banner {
  position: relative;
  display: flex;
  height: 136px;
  margin: 16px 30px 0;
  padding: 0 80px 0 300px;
  overflow: hidden;
  border-radius: 5px;
  align-items: center;
  background: #2f70e9;
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
  color: #fff;
  font-size: clamp(20px, 1.65vw, 30px);
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
  color: #2f70e9;
  background: #fff;
  cursor: pointer;
  font: inherit;
  font-size: 16px;
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
  border-radius: 5px;
  align-items: center;
  justify-content: center;
  color: rgb(255 255 255 / 80%);
  background: transparent;
  cursor: pointer;
}
.tenant-organization-page__banner-close:hover {
  color: #fff;
  background: rgb(255 255 255 / 14%);
}
.tenant-organization-page__banner-close svg {
  width: 22px;
  height: 22px;
}
.tenant-organization-page__title-row,
.tenant-organization-page__role-header {
  display: flex;
  height: 66px;
  padding: 0 30px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  align-items: center;
  justify-content: space-between;
}
.tenant-organization-page__title-row h1,
.tenant-organization-page__role-header h1 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 21px;
  font-weight: 650;
}
.tenant-organization-page__role-header > div {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--el-border-color);
}
.tenant-organization-page__role-header button {
  padding: 4px 6px;
  border: 0;
  border-radius: 4px;
  color: var(--el-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: 16px;
}
.tenant-organization-page__role-header button:hover {
  background: var(--el-color-primary-light-9);
}
.tenant-organization-page__dialog-prompt {
  margin: 0 0 14px;
  color: var(--el-text-color-secondary);
}
.tenant-organization-page__group-list {
  min-height: 260px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  overflow: hidden;
}
.tenant-organization-page__group-list button {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  height: 46px;
  padding: 0 18px;
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
@media (max-width: 1200px) {
  .tenant-organization-page__invite-banner {
    padding-left: 270px;
  }
  .tenant-organization-page__invite-banner img {
    width: 300px;
  }
  .tenant-organization-page__invite-banner strong {
    font-size: 21px;
  }
}
</style>
