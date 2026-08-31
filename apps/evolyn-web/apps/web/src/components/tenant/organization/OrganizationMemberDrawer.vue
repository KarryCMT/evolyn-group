<script setup lang="ts">
import type { OrganizationMember } from './organization.types';
import type { MemberFieldSettingDto, MemberProfileAdminViewDto } from '~/api/memberField';
import {
  EvolynMemberDepartmentRolePicker,
  type EvolynMemberDepartmentRolePickerSelection,
  type EvolynMemberDepartmentRolePickerTreeNode,
} from '@evolyn.do/ui';
import { RiCloseLargeLine, RiGroup2Fill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, reactive, ref, shallowRef, watch } from 'vue';
import { updateMemberDepartments } from '~/api/member';
import { getMemberProfile, updateMemberProfile } from '~/api/memberField';
import { replaceMemberRoles } from '~/api/role';

const props = defineProps<{
  modelValue: boolean;
  member: OrganizationMember | null;
  roleNames: string[];
  departmentTree: EvolynMemberDepartmentRolePickerTreeNode[];
  roleTree: EvolynMemberDepartmentRolePickerTreeNode[];
}>();
const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  save: [member: OrganizationMember];
}>();

type MemberDrawerTab = 'basic' | 'more';

// 基础页签保留组织身份与常用档案字段；字段名称、类型和值始终由成员资料接口返回，
// 因此成员信息管理页调整字段配置后，此处不会维护第二套字段目录。
const BASIC_FIELD_KEYS = [
  'name',
  'alias',
  'code',
  'gender',
  'mobile',
  'email',
  'employeeId',
  'department',
  'role',
] as const;
const readOnlyKeys = new Set(['name', 'mobile', 'email', 'department', 'role']);
const relationKeys = new Set(['department', 'role']);

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
});
const activeTab = ref<MemberDrawerTab>('basic');
const fieldConfig = shallowRef<MemberFieldSettingDto[]>([]);
const profileValues = reactive<Record<string, string>>({});
const identifier = ref('');
const loading = ref(false);
const saving = ref(false);
const departmentPickerVisible = shallowRef(false);
const departmentSaving = shallowRef(false);
const selectedDepartments = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);
const rolePickerVisible = shallowRef(false);
const roleSaving = shallowRef(false);
const selectedRoles = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);
let requestVersion = 0;

const displayName = computed(() => profileValues.name || props.member?.name || '成员');
const displayInitial = computed(() => displayName.value.trim().slice(0, 1) || '成');
const statusLabel = computed(() => {
  const status = props.member?.status;
  if (status === 'disabled') return '已停用';
  if (status === 'resigned') return '已离职';
  return '已启用';
});
const basicFields = computed(() => {
  const fieldByKey = new Map(fieldConfig.value.map((field) => [field.key, field]));
  return BASIC_FIELD_KEYS.map((key) => fieldByKey.get(key)).filter(
    (field): field is MemberFieldSettingDto => Boolean(field),
  );
});
const moreFields = computed(() =>
  fieldConfig.value.filter((field) => !BASIC_FIELD_KEYS.includes(field.key as never)),
);
function isSelectableNode(node: EvolynMemberDepartmentRolePickerTreeNode) {
  return (node as { selectable?: boolean }).selectable !== false;
}
function selectableNodes(nodes: EvolynMemberDepartmentRolePickerTreeNode[]) {
  const result = new Map<string, EvolynMemberDepartmentRolePickerTreeNode>();
  const visit = (nodes: EvolynMemberDepartmentRolePickerTreeNode[]) => {
    for (const node of nodes) {
      if (isSelectableNode(node)) result.set(String(node.id), node);
      if (node.children?.length) visit(node.children);
    }
  };
  visit(nodes);
  return result;
}
const availableDepartments = computed(() => {
  return selectableNodes(props.departmentTree);
});
const availableRoles = computed(() => {
  return selectableNodes(props.roleTree);
});
const selectedDepartmentNames = computed(() =>
  selectedDepartments.value.map((department) => department.label),
);
const selectedRoleNames = computed(() => selectedRoles.value.map((role) => role.label));
// 角色树尚未就绪时，先使用成员列表返回的角色名称，避免抽屉中短暂显示为空。
const displayedRoleNames = computed(() =>
  selectedRoleNames.value.length ? selectedRoleNames.value : props.roleNames,
);

function close() {
  visible.value = false;
}

function clearProfileValues() {
  for (const key of Object.keys(profileValues)) delete profileValues[key];
}

/** 管理员档案接口返回完整字段配置；个人可见性仅作用于成员自己的个人设置页。 */
function applyProfile(profile: MemberProfileAdminViewDto) {
  fieldConfig.value = profile.fieldConfig;
  clearProfileValues();
  for (const field of profile.fieldConfig) {
    profileValues[field.key] = profile.values[field.key] ?? '';
  }
  identifier.value = profile.values.code ?? '';
}

async function loadProfile(memberId: string) {
  const version = ++requestVersion;
  loading.value = true;
  try {
    const profile = await getMemberProfile(memberId);
    if (version === requestVersion) applyProfile(profile);
  } catch {
    if (version === requestVersion) ElMessage.error('成员资料加载失败，请稍后重试');
  } finally {
    if (version === requestVersion) loading.value = false;
  }
}

function isDateField(field: MemberFieldSettingDto) {
  return field.type === '日期时间';
}

function isReadOnly(field: MemberFieldSettingDto) {
  return readOnlyKeys.has(field.key);
}

function isRelation(field: MemberFieldSettingDto) {
  return relationKeys.has(field.key);
}

function inputValue(field: MemberFieldSettingDto) {
  if (field.key === 'mobile') return profileValues.mobile || props.member?.phone || '未绑定手机号';
  if (field.key === 'email') return profileValues.email || props.member?.email || '未绑定邮箱';
  return profileValues[field.key] ?? '';
}

function relationValue(field: MemberFieldSettingDto) {
  if (field.key === 'role')
    return selectedRoleNames.value.join('、') || props.roleNames.join('、') || '未分配角色';
  return selectedDepartmentNames.value.join('、') || inputValue(field) || '未分配部门';
}

/** 将关系 ID 映射为选择器项；根组织等非可选节点不会进入提交快照。 */
function toRelationSelections(
  ids: string[],
  labels: string[],
  available: Map<string, EvolynMemberDepartmentRolePickerTreeNode>,
  type: 'department' | 'role',
) {
  return ids.flatMap((id, index) => {
    const item = available.get(id);
    if (!item) return [];
    return [{ id: item.id, label: item.label || labels[index] || '', type }];
  });
}

function resetDepartmentSelection(member: OrganizationMember) {
  selectedDepartments.value = toRelationSelections(
    member.departmentIds,
    [],
    availableDepartments.value,
    'department',
  );
}

/** 按成员当前角色初始化选择器；角色组节点从不作为可绑定的角色提交。 */
function resetRoleSelection(member: OrganizationMember) {
  selectedRoles.value = toRelationSelections(
    member.roleIds,
    member.roleNames,
    availableRoles.value,
    'role',
  );
}

function openDepartmentPicker() {
  if (!loading.value && !departmentSaving.value) departmentPickerVisible.value = true;
}

/** 部门列表确认后整体替换部门关系；抽屉底部保存仍只负责成员档案字段。 */
async function confirmDepartmentSelection(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  const member = props.member;
  if (!member || departmentSaving.value) return;
  const departments = selections.filter(
    (selection) =>
      selection.type === 'department' && availableDepartments.value.has(String(selection.id)),
  );
  departmentSaving.value = true;
  try {
    await updateMemberDepartments(
      member.id,
      departments.map((department) => String(department.id)),
    );
    selectedDepartments.value = departments;
    emit('save', {
      ...member,
      departmentIds: departments.map((department) => String(department.id)),
      department: departments.map((department) => department.label).join('、') || '未分配部门',
    });
    ElMessage.success('成员部门已保存');
  } catch {
    ElMessage.error('成员部门保存失败，请稍后重试');
  } finally {
    departmentSaving.value = false;
  }
}

function openRolePicker() {
  if (!loading.value && !roleSaving.value) rolePickerVisible.value = true;
}

/** 角色列表确认即原子提交关系，抽屉底部保存仍仅负责成员档案字段。 */
async function confirmRoleSelection(selections: EvolynMemberDepartmentRolePickerSelection[]) {
  const member = props.member;
  if (!member || roleSaving.value) return;
  const roles = selections.filter(
    (selection) => selection.type === 'role' && availableRoles.value.has(String(selection.id)),
  );
  roleSaving.value = true;
  try {
    await replaceMemberRoles(
      member.id,
      roles.map((role) => String(role.id)),
    );
    selectedRoles.value = roles;
    emit('save', {
      ...member,
      roleIds: roles.map((role) => String(role.id)),
      roleNames: roles.map((role) => role.label),
    });
    ElMessage.success('成员角色已保存');
  } catch {
    ElMessage.error('成员角色保存失败，请稍后重试');
  } finally {
    roleSaving.value = false;
  }
}

/** 扩展字段随服务端注册表动态聚合，避免把手机号、部门等关系字段误提交到资料接口。 */
function profilePayloadValues() {
  return fieldConfig.value.reduce<Record<string, string>>((values, field) => {
    if (!isReadOnly(field) && field.key !== 'code')
      values[field.key] = profileValues[field.key] ?? '';
    return values;
  }, {});
}

async function save() {
  const member = props.member;
  if (!member || loading.value || saving.value) return;
  saving.value = true;
  try {
    const profile = await updateMemberProfile(member.id, {
      identifier: identifier.value.trim(),
      values: profilePayloadValues(),
    });
    applyProfile(profile);
    // 成员列表目前只展示姓名、联系方式等身份字段；仍同步本地可见的档案值，
    // 让再次打开抽屉前的列表状态与本次保存保持一致。
    emit('save', {
      ...member,
      employeeNo: profile.values.code ?? '',
      alias: profile.values.alias ?? '',
      gender: profile.values.gender ?? '',
    });
    ElMessage.success('成员资料已保存');
    close();
  } catch {
    ElMessage.error('成员资料保存失败，请检查填写内容后重试');
  } finally {
    saving.value = false;
  }
}

watch(
  () => [visible.value, props.member?.id] as const,
  ([open, memberId]) => {
    const member = props.member;
    if (!open || !memberId || !member) return;
    activeTab.value = 'basic';
    // 切换成员时先清空上一位成员的档案，避免网络请求完成前短暂展示错误资料。
    fieldConfig.value = [];
    clearProfileValues();
    identifier.value = '';
    resetDepartmentSelection(member);
    resetRoleSelection(member);
    void loadProfile(memberId);
  },
  { immediate: true },
);
</script>

<template>
  <el-drawer
    v-model="visible"
    class="organization-member-drawer"
    direction="rtl"
    size="min(720px, 100vw)"
    :show-close="false"
    :close-on-click-modal="true"
    :destroy-on-close="false"
  >
    <template #header>
      <header class="organization-member-drawer__header">
        <span class="organization-member-drawer__avatar" aria-hidden="true">{{
          displayInitial
        }}</span>
        <div class="organization-member-drawer__identity">
          <h2>{{ displayName }}</h2>
          <div class="organization-member-drawer__tags">
            <el-tag effect="dark"> 已加入 </el-tag>
            <el-tag :type="member?.status === 'active' ? 'success' : 'info'" effect="dark">
              {{ statusLabel }}
            </el-tag>
          </div>
        </div>
        <button
          class="organization-member-drawer__close"
          type="button"
          aria-label="关闭编辑成员"
          @click="close"
        >
          <RiCloseLargeLine />
        </button>
      </header>
    </template>

    <section class="organization-member-drawer__layout" aria-label="编辑成员">
      <el-tabs v-model="activeTab" class="organization-member-drawer__tabs" stretch>
        <el-tab-pane label="基础字段" name="basic" />
        <el-tab-pane label="更多字段" name="more" />
      </el-tabs>

      <el-scrollbar v-loading="loading" class="organization-member-drawer__content">
        <form v-if="member" class="organization-member-drawer__form" @submit.prevent="save">
          <template v-if="activeTab === 'basic'">
            <div class="organization-member-drawer__field-grid">
              <div
                v-for="field in basicFields.slice(0, 4)"
                :key="field.key"
                class="organization-member-drawer__field"
              >
                <label :for="`member-${field.key}`">
                  {{ field.label }}<span v-if="field.key === 'name'" aria-hidden="true">*</span>
                </label>
                <el-input
                  v-if="field.key === 'code'"
                  :id="`member-${field.key}`"
                  v-model="identifier"
                  maxlength="50"
                  placeholder="请输入"
                />
                <el-input
                  v-else
                  :id="`member-${field.key}`"
                  :model-value="inputValue(field)"
                  :disabled="isReadOnly(field)"
                  maxlength="50"
                  placeholder="请输入"
                  @update:model-value="profileValues[field.key] = String($event)"
                />
                <p v-if="field.key === 'code'" class="organization-member-drawer__help">
                  企业内编号由管理员维护，需保持唯一。
                </p>
              </div>
            </div>

            <template v-for="field in basicFields.slice(4)" :key="field.key">
              <section
                v-if="isRelation(field)"
                class="organization-member-drawer__field-section organization-member-drawer__field-section--relation"
              >
                <label>{{ field.label }}</label>
                <button
                  v-if="field.key === 'role'"
                  class="organization-member-drawer__relation-value organization-member-drawer__relation-value--interactive organization-member-drawer__relation-value--roles"
                  type="button"
                  :disabled="roleSaving"
                  @click="openRolePicker"
                >
                  <span class="organization-member-drawer__role-list">
                    <span
                      v-for="roleName in displayedRoleNames"
                      :key="roleName"
                      class="organization-member-drawer__role-chip"
                    >
                      <RiGroup2Fill aria-hidden="true" />
                      <span>{{ roleName }}</span>
                    </span>
                    <span
                      v-if="!displayedRoleNames.length"
                      class="organization-member-drawer__role-empty"
                    >
                      未分配角色
                    </span>
                  </span>
                </button>
                <button
                  v-else
                  class="organization-member-drawer__relation-value organization-member-drawer__relation-value--interactive"
                  type="button"
                  :disabled="departmentSaving"
                  @click="openDepartmentPicker"
                >
                  <RiGroup2Fill aria-hidden="true" />
                  <span>{{ relationValue(field) }}</span>
                </button>
              </section>
              <section v-else class="organization-member-drawer__field-section">
                <label :for="`member-${field.key}`">{{ field.label }}</label>
                <div v-if="field.key === 'mobile'" class="organization-member-drawer__phone-value">
                  <span>+86</span>
                  <el-input :model-value="inputValue(field).replace(/^\+86-?/, '')" disabled />
                </div>
                <el-input
                  v-else
                  :id="`member-${field.key}`"
                  :model-value="inputValue(field)"
                  :disabled="isReadOnly(field)"
                  maxlength="50"
                  placeholder="请输入"
                  @update:model-value="profileValues[field.key] = String($event)"
                />
                <p v-if="field.key === 'mobile'" class="organization-member-drawer__help">
                  手机已验证，如需修改请由成员在个人设置中重新绑定。
                </p>
                <p v-else-if="field.key === 'email'" class="organization-member-drawer__help">
                  邮箱属于账号安全信息，请由成员在个人设置中维护。
                </p>
              </section>
            </template>
          </template>

          <template v-else>
            <section
              v-for="field in moreFields"
              :key="field.key"
              class="organization-member-drawer__field-section"
            >
              <label :for="`member-${field.key}`">{{ field.label }}</label>
              <el-date-picker
                v-if="isDateField(field)"
                :id="`member-${field.key}`"
                v-model="profileValues[field.key]"
                type="date"
                value-format="YYYY-MM-DD"
                format="YYYY-MM-DD"
                placeholder="请选择日期"
              />
              <el-input
                v-else
                :id="`member-${field.key}`"
                v-model="profileValues[field.key]"
                maxlength="50"
                placeholder="请输入"
              />
            </section>
            <el-empty v-if="!moreFields.length && !loading" description="暂未配置更多成员字段" />
          </template>
        </form>
      </el-scrollbar>

      <footer class="organization-member-drawer__footer">
        <el-button type="primary" :loading="saving" :disabled="loading" @click="save">
          保存
        </el-button>
        <el-button @click="close"> 取消 </el-button>
      </footer>
    </section>
  </el-drawer>

  <EvolynMemberDepartmentRolePicker
    v-model:open="departmentPickerVisible"
    v-model="selectedDepartments"
    title="部门列表"
    :departments="departmentTree"
    :selectable-types="['department']"
    :allow-empty="true"
    @confirm="confirmDepartmentSelection"
  />

  <EvolynMemberDepartmentRolePicker
    v-model:open="rolePickerVisible"
    v-model="selectedRoles"
    title="角色列表"
    :roles="roleTree"
    :selectable-types="['role']"
    :allow-empty="true"
    @confirm="confirmRoleSelection"
  />
</template>

<style scoped lang="scss">
.organization-member-drawer {
  --el-drawer-padding-primary: 0;

  :global(.el-drawer__header) {
    margin: 0;
    padding: 0;
  }

  &__layout {
    display: flex;
    height: 100%;
    min-width: 0;
    flex-direction: column;
    background: var(--el-bg-color);
  }

  &__header {
    display: flex;
    box-sizing: border-box;
    height: 74px;
    min-height: 74px;
    padding: 0 var(--el-space-2xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    gap: var(--el-space-xl);
  }

  &__avatar {
    display: grid;
    width: 44px;
    height: 44px;
    flex: 0 0 auto;
    border-radius: var(--el-border-radius-half);
    place-items: center;
    color: var(--el-color-white);
    background: linear-gradient(135deg, var(--el-color-primary-light-3), var(--el-color-primary));
    box-shadow: 0 4px 12px color-mix(in srgb, var(--el-color-primary) 18%, transparent);
    font-size: 20px;
    font-weight: 650;
  }

  &__identity {
    min-width: 0;
  }

  &__identity h2 {
    margin: 0 0 var(--el-space-sm);
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 650;
    line-height: 24px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__tags {
    display: flex;
    gap: var(--el-space-md);
  }

  &__tags :deep(.el-tag) {
    border: 0;
    font-weight: 500;
  }

  &__close {
    display: inline-grid;
    width: 32px;
    height: 32px;
    margin-left: auto;
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-medium);
    place-items: center;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;
  }

  &__close:hover {
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
  }

  &__close svg {
    width: 22px;
    height: 22px;
  }

  &__tabs {
    flex: 0 0 auto;
    padding: 0 var(--el-space-2xl);
  }

  &__tabs :deep(.el-tabs__header) {
    margin: 0;
  }

  &__tabs :deep(.el-tabs__nav-wrap::after) {
    height: 1px;
  }

  &__tabs :deep(.el-tabs__item) {
    height: 44px;
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-medium);
    line-height: 44px;
  }

  &__tabs :deep(.el-tabs__item.is-active) {
    color: var(--el-color-primary);
    font-weight: 650;
  }

  &__tabs :deep(.el-tabs__active-bar) {
    height: 3px;
  }

  &__tabs :deep(.el-tabs__content) {
    display: none;
  }

  &__content {
    min-height: 0;
    flex: 1;
  }

  &__content :deep(.el-scrollbar__view) {
    min-height: 100%;
  }

  &__form {
    display: grid;
    padding: var(--el-space-xl) var(--el-space-2xl) var(--el-space-2xl);
    gap: var(--el-space-xl);
  }

  &__field-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-xl);
  }

  &__field,
  &__field-section {
    display: grid;
    min-width: 0;
    gap: var(--el-space-sm);
  }

  &__field-section {
    padding-top: var(--el-space-xl);
    border-top: 1px dashed var(--el-border-color);
  }

  &__field label,
  &__field-section > label {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 560;
    line-height: 20px;
  }

  &__field label span {
    margin-left: 2px;
    color: var(--el-color-danger);
  }

  &__field :deep(.el-input__wrapper),
  &__field-section :deep(.el-input__wrapper),
  &__field-section :deep(.el-date-editor.el-input) {
    min-height: 34px;
    box-shadow: 0 0 0 1px var(--el-border-color) inset;
  }

  &__field :deep(.el-input__wrapper.is-focus),
  &__field-section :deep(.el-input__wrapper.is-focus) {
    box-shadow: 0 0 0 1px var(--el-color-primary) inset;
  }

  &__field :deep(.el-input.is-disabled .el-input__wrapper),
  &__field-section :deep(.el-input.is-disabled .el-input__wrapper) {
    background: var(--el-fill-color-light);
    box-shadow: 0 0 0 1px var(--el-border-color-lighter) inset;
  }

  &__help {
    margin: calc(-1 * var(--el-space-xs)) 0 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    line-height: 17px;
  }

  &__phone-value {
    display: flex;
  }

  &__phone-value > span {
    display: flex;
    min-height: 34px;
    padding: 0 var(--el-space-lg);
    border: 1px solid var(--el-border-color-lighter);
    border-right: 0;
    border-radius: var(--el-border-radius-medium) 0 0 var(--el-border-radius-medium);
    align-items: center;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
  }

  &__phone-value :deep(.el-input__wrapper) {
    border-radius: 0 var(--el-border-radius-medium) var(--el-border-radius-medium) 0;
  }

  &__relation-value {
    display: flex;
    min-height: 46px;
    padding: var(--el-space-sm) var(--el-space-md);
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-medium);
    align-items: flex-start;
    gap: var(--el-space-md);
    text-align: start;
  }

  &__relation-value--interactive {
    width: 100%;
    color: inherit;
    background: transparent;
    cursor: pointer;
  }

  &__relation-value--interactive:hover:not(:disabled) {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
  }

  &__relation-value--interactive:disabled {
    cursor: wait;
    opacity: 0.7;
  }

  &__relation-value--roles {
    min-height: 170px;
    align-items: flex-start;
  }

  &__relation-value svg {
    width: 20px;
    height: 20px;
    flex: 0 0 auto;
    color: var(--el-color-primary);
  }

  &__relation-value span {
    color: var(--el-text-color-primary);
    line-height: 20px;
  }

  &__role-list {
    display: flex;
    flex-wrap: wrap;
    gap: var(--el-space-md);
  }

  &__role-chip {
    display: inline-flex;
    min-height: 36px;
    padding: 0 var(--el-space-md);
    align-items: center;
    gap: var(--el-space-sm);
    border-radius: var(--el-border-radius-medium);
    background: var(--el-fill-color-light);
  }

  &__role-chip svg {
    width: 20px;
    height: 20px;
    color: var(--el-color-primary);
  }

  &__role-empty {
    color: var(--el-text-color-secondary) !important;
  }

  &__footer {
    display: flex;
    min-height: 56px;
    padding: 0 var(--el-space-2xl);
    border-top: 1px solid var(--el-border-color-lighter);
    align-items: center;
    gap: var(--el-space-lg);
    box-shadow: 0 -4px 14px rgb(31 45 61 / 4%);
  }

  &__footer :deep(.el-button) {
    min-width: 72px;
    height: 32px;
  }
}

@media (max-width: 600px) {
  .organization-member-drawer {
    &__header,
    &__tabs,
    &__form,
    &__footer {
      padding-right: var(--el-space-xl);
      padding-left: var(--el-space-xl);
    }

    &__field-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>
