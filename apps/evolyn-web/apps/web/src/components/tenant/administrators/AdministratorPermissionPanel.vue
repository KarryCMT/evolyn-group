<script setup lang="ts">
import { RiAddFill, RiQuestionFill, RiTeamFill, RiUserSettingsFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, onMounted, shallowRef } from 'vue';
import { listApplications } from '~/api/applications';
import { getDepartmentTree, type DepartmentDto } from '~/api/department';
import { listMembers } from '~/api/member';
import { getOrganizationRoleTree } from '~/api/role';
import { useAuth } from '~/composables/auth';
import type { ApplicationIcon } from '~/types';
import AddressBookManagementDrawer from './AddressBookManagementDrawer.vue';
import AdministratorApplicationPickerDialog from './AdministratorApplicationPickerDialog.vue';
import AdministratorMemberPickerDialog from './AdministratorMemberPickerDialog.vue';
import AdministratorScopePickerDialog from './AdministratorScopePickerDialog.vue';
import type {
  AdministratorGroup,
  AdministratorPickerMember,
  AdministratorScope,
  AddressBookScope,
  ScopeMode,
} from './administrator.types';

defineOptions({ name: 'AdministratorPermissionPanel' });

const props = defineProps<{
  scope: AdministratorScope;
  group: AdministratorGroup | null;
  loading?: boolean;
  saving?: boolean;
  /** 区块保存入口（组合层乐观更新 + 失败回滚），返回是否保存成功。 */
  save: (patch: Partial<AdministratorGroup>) => Promise<boolean>;
}>();

const memberPickerVisible = shallowRef(false);
const applicationPickerVisible = shallowRef(false);
const addressBookVisible = shallowRef(false);
const departmentPickerVisible = shallowRef(false);
const rolePickerVisible = shallowRef(false);
const { userInfo } = useAuth();
/** 租户创建人由登录聚合信息提供；内置系统组须固定保留，自定义组不可加入。 */
const tenantOwnerAccountId = computed(() => userInfo.value?.tenant.ownerAccountId ?? null);
/** 管理组成员调整中禁止当前操作者移除自己，避免误操作失去管理入口。 */
const currentMemberId = computed(() => userInfo.value?.member.id ?? null);

// ---- 选择器数据源（挂载后并行加载；失败静默为空，弹窗展示暂无可选项）----
const members = shallowRef<AdministratorPickerMember[]>([]);
const departments = shallowRef<DepartmentDto[]>([]);
const roles = shallowRef<{ id: number; name: string }[]>([]);
const applications = shallowRef<{ id: number; name: string; icon: ApplicationIcon }[]>([]);

/** 部门/角色 ID → 名称（清单 chip 展示用）。 */
const departmentNameById = computed(() => {
  const names = new Map<number, string>();
  const walk = (list: DepartmentDto[]) => {
    for (const department of list) {
      names.set(department.id, department.name);
      walk(department.children ?? []);
    }
  };
  walk(departments.value);
  return names;
});
const roleNameById = computed(() => new Map(roles.value.map((role) => [role.id, role.name])));

const selectedDepartmentNames = computed(() =>
  (props.group?.departmentIds ?? [])
    .map((id) => departmentNameById.value.get(id))
    .filter((name): name is string => Boolean(name)),
);
const selectedRoleNames = computed(() =>
  (props.group?.roleIds ?? [])
    .map((id) => roleNameById.value.get(id))
    .filter((name): name is string => Boolean(name)),
);

/** 面板展示的可编辑应用：全量语义展开为全部应用，否则按清单过滤。 */
const selectedApplications = computed(() => {
  const group = props.group;
  if (!group) return [];
  if (group.allApplications) return applications.value;
  const ids = new Set(group.applicationIds ?? []);
  return applications.value.filter((app) => ids.has(app.id));
});

async function loadPickerSources() {
  const [memberPage, departmentTree, roleTree, applicationPage] = await Promise.allSettled([
    listMembers({ page: 1, pageSize: 500 }),
    getDepartmentTree(),
    getOrganizationRoleTree(),
    listApplications({ limit: 100 }),
  ]);
  if (memberPage.status === 'fulfilled') {
    members.value = memberPage.value.items.map((item) => ({
      id: item.id,
      accountId: item.accountId,
      name: item.name,
      department: item.departments[0]?.name ?? '',
      departmentIds: item.departments.map((department) => department.id),
    }));
  }
  if (departmentTree.status === 'fulfilled') departments.value = departmentTree.value;
  if (roleTree.status === 'fulfilled') {
    roles.value = roleTree.value.groups.flatMap((group) => group.roles);
  }
  if (applicationPage.status === 'fulfilled') {
    applications.value = applicationPage.value.items.map((app) => ({
      id: app.id,
      name: app.name,
      icon: app.icon,
    }));
  }
}

onMounted(() => {
  loadPickerSources().catch(() => undefined);
});

// ---- 区块保存：每次提交所属区块的全部字段（组合层映射为单区块 PATCH）----

/** 保存成功后使用全局消息提示，避免在面板内遗留固定位置的状态文案。 */
async function patch(patchValue: Partial<AdministratorGroup>) {
  if (!(await props.save(patchValue))) return;
  ElMessage.success('修改已保存');
}

/** 内置组配置固定全量：除成员与名称外不可变更，控件整体禁用。 */
const configDisabled = computed(() => props.group?.builtIn || props.saving === true);
/** 所有管理组复用同一成员选择区，确保系统、通讯录与应用管理组的交互一致。 */
const useMemberSelectionBox = computed(() => Boolean(props.group));

function patchDepartment(enabled?: boolean, mode?: ScopeMode, ids?: number[]) {
  const group = props.group;
  if (!group) return;
  void patch({
    departmentEnabled: enabled ?? group.departmentEnabled,
    departmentMode: mode ?? group.departmentMode,
    departmentIds: ids ?? group.departmentIds ?? [],
  });
}

function patchRole(visible?: boolean, manage?: boolean, mode?: ScopeMode, ids?: number[]) {
  const group = props.group;
  if (!group) return;
  // 可管理必然隐含可见：前端联动提交，服务端兜底校验
  const nextManage = manage ?? group.roleManage;
  void patch({
    roleVisible: visible ?? (group.roleVisible || nextManage),
    roleManage: nextManage,
    roleMode: mode ?? group.roleMode,
    roleIds: ids ?? group.roleIds ?? [],
  });
}

function patchApplication(manage?: boolean, ids?: number[], all?: boolean) {
  const group = props.group;
  if (!group) return;
  void patch({
    allApplications: all ?? group.allApplications,
    applicationIds: ids ?? group.applicationIds ?? [],
    applicationManage: manage ?? group.applicationManage,
  });
}

function onApplicationPickerConfirm(payload: { ids: number[]; all: boolean }) {
  patchApplication(undefined, payload.ids, payload.all);
}

async function saveAddressBook(scope: AddressBookScope) {
  return props.save({ addressBook: scope });
}
</script>

<template>
  <section class="administrator-permission-panel" aria-label="管理组权限">
    <template v-if="group">
      <header class="administrator-permission-panel__header">
        <div>
          <h1>{{ group.name }}</h1>
          <p v-if="scope === 'system' && group.builtIn">
            系统管理员具备全产品/模块的全量管理及数据权限，建议配置不超过 5 人
          </p>
        </div>
      </header>
      <div v-loading="loading" class="administrator-permission-panel__body">
        <section
          class="administrator-permission-panel__row administrator-permission-panel__row--members"
          :class="{
            'administrator-permission-panel__row--member-selection': useMemberSelectionBox,
          }"
        >
          <h2>管理员</h2>
          <template v-if="useMemberSelectionBox">
            <!-- 所有管理组复用同一整块成员选择样式，并保留完整点击入口。 -->
            <button
              class="administrator-permission-panel__member-selection"
              type="button"
              :disabled="saving"
              aria-label="编辑管理员"
              @click="memberPickerVisible = true"
            >
              <span
                v-for="member in group.members"
                :key="member.id"
                class="administrator-permission-panel__member-selection-chip"
              >
                <i>{{ member.name.slice(0, 1) }}</i>{{ member.name }}
              </span>
              <span
                v-if="group.members.length === 0"
                class="administrator-permission-panel__member-selection-placeholder"
              >
                <RiAddFill />选择成员
              </span>
            </button>
          </template>
          <template v-else>
            <button
              class="administrator-permission-panel__text-action"
              type="button"
              :disabled="saving"
              @click="memberPickerVisible = true"
            >
              <RiAddFill />选择成员
            </button>
            <span
              v-for="member in group.members"
              :key="member.id"
              class="administrator-permission-panel__member"
              ><i>{{ member.name.slice(0, 1) }}</i
              >{{ member.name }}</span
            >
          </template>
        </section>

        <!-- 内置系统管理员的权限恒为全量，因此仅维护管理员成员；通讯录管理组
             才展示完整的通讯录权限配置。 -->
        <template v-if="scope === 'system' && !group.builtIn">
          <section class="administrator-permission-panel__row">
            <h2>内部部门</h2>
            <el-checkbox
              :model-value="group.departmentEnabled"
              :disabled="configDisabled"
              @update:model-value="patchDepartment(Boolean($event))"
              >可见/可管理</el-checkbox
            >
          </section>
          <section
            v-if="group.departmentEnabled"
            class="administrator-permission-panel__scope-detail"
          >
            <el-radio-group
              :model-value="group.departmentMode"
              :disabled="configDisabled"
              @update:model-value="patchDepartment(undefined, $event as ScopeMode)"
              ><el-radio value="all">全部部门</el-radio
              ><el-radio value="partial">部分部门</el-radio></el-radio-group
            >
            <div
              v-if="group.departmentMode === 'partial'"
              class="administrator-permission-panel__selection-box"
            >
              <span
                v-for="name in selectedDepartmentNames"
                :key="name"
                class="administrator-permission-panel__selection-item"
                ><RiTeamFill />{{ name }}</span
              >
              <button
                class="administrator-permission-panel__text-action"
                type="button"
                :disabled="configDisabled"
                @click="departmentPickerVisible = true"
              >
                <RiAddFill />选择部门
              </button>
            </div>
          </section>
          <section class="administrator-permission-panel__row">
            <h2>内部角色</h2>
            <div class="administrator-permission-panel__checkbox-pair">
              <el-checkbox
                :model-value="group.roleVisible || group.roleManage"
                :disabled="configDisabled"
                @update:model-value="patchRole(Boolean($event))"
                >可见</el-checkbox
              ><el-checkbox
                :model-value="group.roleManage"
                :disabled="configDisabled"
                @update:model-value="patchRole(undefined, Boolean($event))"
                >可管理</el-checkbox
              >
            </div>
          </section>
          <section
            v-if="group.roleVisible || group.roleManage"
            class="administrator-permission-panel__scope-detail"
          >
            <el-radio-group
              :model-value="group.roleMode"
              :disabled="configDisabled"
              @update:model-value="patchRole(undefined, undefined, $event as ScopeMode)"
              ><el-radio value="all">全部角色</el-radio
              ><el-radio value="partial">部分角色</el-radio></el-radio-group
            >
            <div
              v-if="group.roleMode === 'partial'"
              class="administrator-permission-panel__selection-box"
            >
              <span
                v-for="name in selectedRoleNames"
                :key="name"
                class="administrator-permission-panel__selection-item"
                ><RiUserSettingsFill />{{ name }}</span
              >
              <button
                class="administrator-permission-panel__text-action"
                type="button"
                :disabled="configDisabled"
                @click="rolePickerVisible = true"
              >
                <RiAddFill />选择角色
              </button>
            </div>
          </section>
          <section class="administrator-permission-panel__row">
            <h2>互联组织</h2>
            <el-checkbox
              :model-value="group.externalEnabled"
              :disabled="configDisabled"
              @update:model-value="patch({ externalEnabled: Boolean($event) })"
              >可见/可管理 <RiQuestionFill
            /></el-checkbox>
          </section>
        </template>

        <template v-else-if="scope === 'application'">
          <section class="administrator-permission-panel__row">
            <h2>应用管理</h2>
            <button
              class="administrator-permission-panel__text-action"
              type="button"
              :disabled="saving"
              @click="applicationPickerVisible = true"
            >
              <RiAddFill />选择可编辑的应用
            </button>
          </section>
          <section class="administrator-permission-panel__app-settings">
            <el-checkbox
              :model-value="group.applicationManage"
              :disabled="saving"
              @update:model-value="patchApplication(Boolean($event))"
              >可添加/删除应用</el-checkbox
            >
            <div class="administrator-permission-panel__app-tags">
              <span v-for="application in selectedApplications" :key="application.id">{{
                application.name
              }}</span>
            </div>
          </section>
          <section class="administrator-permission-panel__row">
            <h2>可选部门 <RiQuestionFill /></h2>
            <el-radio-group
              :model-value="group.departmentMode"
              :disabled="saving"
              @update:model-value="patchDepartment(undefined, $event as ScopeMode)"
              ><el-radio value="all">全部部门</el-radio
              ><el-radio value="partial">部分部门</el-radio></el-radio-group
            >
          </section>
          <section
            v-if="group.departmentMode === 'partial'"
            class="administrator-permission-panel__sub-action"
          >
            <div class="administrator-permission-panel__selection-box">
              <span
                v-for="name in selectedDepartmentNames"
                :key="name"
                class="administrator-permission-panel__selection-item"
                ><RiTeamFill />{{ name }}</span
              >
            </div>
            <button
              class="administrator-permission-panel__text-action"
              type="button"
              :disabled="saving"
              @click="departmentPickerVisible = true"
            >
              <RiAddFill />选择部门
            </button>
          </section>
          <section class="administrator-permission-panel__row">
            <h2>可选角色 <RiQuestionFill /></h2>
            <el-radio-group
              :model-value="group.roleMode"
              :disabled="saving"
              @update:model-value="patchRole(undefined, undefined, $event as ScopeMode)"
              ><el-radio value="all">全部角色</el-radio
              ><el-radio value="partial">部分角色</el-radio></el-radio-group
            >
          </section>
          <section
            v-if="group.roleMode === 'partial'"
            class="administrator-permission-panel__sub-action"
          >
            <div class="administrator-permission-panel__selection-box">
              <span
                v-for="name in selectedRoleNames"
                :key="name"
                class="administrator-permission-panel__selection-item"
                ><RiUserSettingsFill />{{ name }}</span
              >
            </div>
            <button
              class="administrator-permission-panel__text-action"
              type="button"
              :disabled="saving"
              @click="rolePickerVisible = true"
            >
              <RiAddFill />选择角色
            </button>
          </section>
          <section class="administrator-permission-panel__row">
            <h2>可选互联组织 <RiQuestionFill /></h2>
            <el-checkbox
              :model-value="group.externalEnabled"
              :disabled="saving"
              @update:model-value="patch({ externalEnabled: Boolean($event) })"
              >全部互联组织</el-checkbox
            >
          </section>
          <section
            class="administrator-permission-panel__row administrator-permission-panel__row--address-book"
          >
            <h2>通讯录管理</h2>
            <div>
              <p>
                可为成员设置通讯录管理权限，并可在通讯录管理组中查看。
                <button type="button">了解更多</button>
              </p>
              <el-button plain type="primary" :disabled="saving" @click="addressBookVisible = true"
                >设置</el-button
              >
            </div>
          </section>
        </template>
      </div>
      <AdministratorMemberPickerDialog
        v-model="memberPickerVisible"
        :members="members"
        :departments="departments"
        :selected-members="group.members"
        :tenant-owner-account-id="tenantOwnerAccountId"
        :is-builtin-system-group="scope === 'system' && group.builtIn"
        :current-member-id="currentMemberId"
        @confirm="patch({ members: $event })"
      />
      <AdministratorApplicationPickerDialog
        v-if="scope === 'application'"
        v-model="applicationPickerVisible"
        :applications="applications"
        :selected-ids="
          group.allApplications ? applications.map((app) => app.id) : (group.applicationIds ?? [])
        "
        @confirm="onApplicationPickerConfirm"
      />
      <AdministratorScopePickerDialog
        v-model="departmentPickerVisible"
        target="department"
        :selected-ids="group.departmentIds"
        @confirm="patchDepartment(undefined, undefined, $event)"
      />
      <AdministratorScopePickerDialog
        v-model="rolePickerVisible"
        target="role"
        :selected-ids="group.roleIds"
        @confirm="patchRole(undefined, undefined, undefined, $event)"
      />
      <AddressBookManagementDrawer
        v-if="scope === 'application'"
        v-model="addressBookVisible"
        :initial="group.addressBook ?? null"
        :save="saveAddressBook"
      />
    </template>
    <div
      v-else
      v-loading="loading"
      class="administrator-permission-panel__placeholder"
      aria-label="暂无选中的管理组"
    >
      <p v-if="!loading">暂无管理组</p>
    </div>
  </section>
</template>

<style scoped lang="scss">
.administrator-permission-panel {
  display: flex;
  min-width: 0;
  height: 100%;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);
  &__header {
    display: flex;
    min-height: 68px;
    padding: 0 var(--el-space-4xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
  }
  &__header div {
    display: flex;
    align-items: baseline;
    gap: var(--el-space-lg);
  }
  &__header h1 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    line-height: 28px;
  }
  &__header p {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }
  &__body {
    position: relative;
    padding: var(--el-space-3xl) var(--el-space-4xl) 80px;
  }
  &__placeholder {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    min-height: 200px;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }
  &__row {
    display: grid;
    min-height: 64px;
    grid-template-columns: 246px 1fr;
    align-items: start;
  }
  &__row h2 {
    display: inline-flex;
    margin: 0;
    align-items: center;
    gap: var(--el-space-xs);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-large);
    line-height: 30px;
  }
  &__row h2 svg,
  &__row :deep(.el-checkbox svg) {
    width: 18px;
    height: 18px;
    color: var(--el-text-color-placeholder);
  }
  &__row :deep(.el-checkbox),
  &__row :deep(.el-radio) {
    min-height: 30px;
    margin-right: var(--el-space-3xl);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
  }
  &__row :deep(.el-checkbox__label),
  &__row :deep(.el-radio__label) {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-xs);
    font-size: var(--el-font-size-medium);
  }
  &__row--members {
    align-items: center;
  }
  &__row--member-selection {
    min-height: 100px;
  }
  &__text-action {
    display: inline-flex;
    min-height: 32px;
    border: 0;
    padding: 0;
    align-items: center;
    gap: var(--el-space-xs);
    color: var(--el-color-primary);
    background: transparent;
    font-size: var(--el-font-size-medium);
    cursor: pointer;
  }
  &__text-action svg {
    width: 22px;
    height: 22px;
  }
  &__text-action:hover {
    text-decoration: underline;
  }
  &__text-action:disabled {
    color: var(--el-text-color-placeholder);
    cursor: not-allowed;
    text-decoration: none;
  }
  &__member {
    display: inline-flex;
    height: 32px;
    margin-left: var(--el-space-lg);
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
  }
  &__member i {
    display: inline-flex;
    width: 24px;
    height: 24px;
    border-radius: var(--el-border-radius-half);
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    font-size: var(--el-font-size-small);
    font-style: normal;
  }
  &__member-selection {
    display: flex;
    min-height: 80px;
    padding: var(--el-space-lg);
    border: 1px dashed var(--el-border-color-light);
    border-radius: var(--el-border-radius-medium);
    align-content: flex-start;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: var(--el-space-md);
    background: transparent;
    cursor: pointer;
  }
  &__member-selection:hover:not(:disabled) {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
  }
  &__member-selection:disabled {
    cursor: not-allowed;
  }
  &__member-selection-chip {
    display: inline-flex;
    height: 42px;
    padding: 0 var(--el-space-lg) 0 var(--el-space-sm);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-medium);
  }
  &__member-selection-chip i {
    display: inline-flex;
    width: 28px;
    height: 28px;
    border-radius: var(--el-border-radius-half);
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    font-size: var(--el-font-size-small);
    font-style: normal;
  }
  &__member-selection-placeholder {
    display: inline-flex;
    min-height: 42px;
    align-items: center;
    gap: var(--el-space-xs);
    color: var(--el-color-primary);
    font-size: var(--el-font-size-medium);
  }
  &__member-selection-placeholder svg {
    width: 22px;
    height: 22px;
  }
  &__scope-detail {
    margin: -5px 0 var(--el-space-3xl) 246px;
  }
  &__scope-detail :deep(.el-radio) {
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
  }
  &__selection-box {
    display: flex;
    min-height: 92px;
    margin-top: var(--el-space-xl);
    padding: var(--el-space-lg);
    border: 1px dashed var(--el-border-color-light);
    border-radius: var(--el-border-radius-medium);
    align-content: flex-start;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: var(--el-space-md);
  }
  &__selection-item {
    display: inline-flex;
    height: 34px;
    padding: 0 var(--el-space-md);
    border-radius: var(--el-border-radius-base);
    align-items: center;
    gap: var(--el-space-xs);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-medium);
  }
  &__selection-item svg {
    color: var(--el-color-primary);
  }
  &__checkbox-pair {
    display: flex;
    align-items: center;
  }
  &__app-settings {
    display: flex;
    min-height: 90px;
    margin: -6px 0 var(--el-space-md) 246px;
    flex-direction: column;
    gap: var(--el-space-lg);
  }
  &__app-settings :deep(.el-checkbox) {
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
  }
  &__app-tags {
    display: flex;
    flex-wrap: wrap;
    gap: var(--el-space-md);
  }
  &__app-tags span {
    display: inline-flex;
    height: 34px;
    padding: 0 var(--el-space-md);
    border-radius: var(--el-border-radius-base);
    align-items: center;
    gap: var(--el-space-xs);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-base);
  }
  &__sub-action {
    margin: -18px 0 var(--el-space-3xl) 246px;
  }
  &__row--address-book {
    margin-top: var(--el-space-lg);
  }
  &__row--address-book p {
    margin: 0 0 var(--el-space-xl);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }
  &__row--address-book p button {
    border: 0;
    padding: 0;
    color: var(--el-color-primary);
    background: transparent;
    font: inherit;
    cursor: pointer;
  }
  &__row--address-book p button:hover {
    text-decoration: underline;
  }
  &__row--address-book .el-button {
    min-width: 72px;
    height: 42px;
    font-size: var(--el-font-size-medium);
  }
}
</style>
