<script setup lang="ts">
import { RiCloseLargeFill, RiSearchFill } from '@remixicon/vue';
import { computed, nextTick, shallowRef, useTemplateRef, watch } from 'vue';
import PickerSelectedItems from './PickerSelectedItems.vue';
import PickerVirtualMembers from './PickerVirtualMembers.vue';
import PickerVirtualTree from './PickerVirtualTree.vue';
import type {
  EvolynMemberDepartmentRolePickerEmits,
  EvolynMemberDepartmentRolePickerProps,
  EvolynMemberDepartmentRolePickerSelection,
} from './EvolynMemberDepartmentRolePicker.types';
import { useMemberDepartmentRolePicker } from './useMemberDepartmentRolePicker';
import { useVirtualTree } from './useVirtualTree';

defineOptions({ name: 'EvolynMemberDepartmentRolePicker' });

const props = withDefaults(defineProps<EvolynMemberDepartmentRolePickerProps>(), {
  title: '选择成员、部门或角色',
  departments: () => [],
  roles: () => [],
  members: () => [],
  currentMemberDepartmentIds: () => [],
  showCurrentMemberDepartmentTab: false,
  selectableTypes: () => ['department', 'role', 'member'],
  multiple: true,
  allowEmpty: false,
  searchPlaceholder: '搜索（多个关键词用空格隔开）',
  emptyText: '暂无可选择的数据',
  currentMemberDepartmentEmptyText: '当前用户暂未归属部门',
});

const emit = defineEmits<EvolynMemberDepartmentRolePickerEmits>();
const modelValue = defineModel<EvolynMemberDepartmentRolePickerSelection[]>({ default: () => [] });
const open = defineModel<boolean>('open', { default: false });
const dialogRef = useTemplateRef<HTMLElement>('dialog');
const draftSelection = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);
const departmentView = shallowRef<'organization' | 'current-member'>('organization');

const {
  activeDepartmentId,
  activeType,
  availableTypes,
  isDisabled,
  isMultiple,
  keyword,
  normalizeSelection,
  remove,
  resetView,
  selectedKeys,
  toggle,
  visibleDepartments,
  visibleCurrentMemberDepartments,
  visibleMembers,
  visibleRoles,
} = useMemberDepartmentRolePicker({
  departments: () => props.departments,
  roles: () => props.roles,
  members: () => props.members,
  currentMemberDepartmentIds: () => props.currentMemberDepartmentIds,
  selectableTypes: () => props.selectableTypes,
  multiple: () => props.multiple,
  departmentMultiple: () => props.departmentMultiple,
  memberMultiple: () => props.memberMultiple,
  max: () => props.max,
  selection: draftSelection,
});

// 树先拍平再交给虚拟列表，避免深层组织架构递归创建全部节点。
const { rows: departmentVirtualRows, toggleExpanded: toggleDepartmentExpanded } =
  useVirtualTree(visibleDepartments);
const { rows: currentMemberDepartmentVirtualRows } = useVirtualTree(
  visibleCurrentMemberDepartments,
);
const { rows: roleVirtualRows, toggleExpanded: toggleRoleExpanded } = useVirtualTree(visibleRoles);

const canConfirm = computed(() => props.allowEmpty || draftSelection.value.length > 0);
const showCurrentMemberDepartmentTab = computed(
  () =>
    props.showCurrentMemberDepartmentTab &&
    availableTypes.value.length === 1 &&
    availableTypes.value[0] === 'department',
);

function resetDraft() {
  draftSelection.value = normalizeSelection(modelValue.value);
  resetView();
  departmentView.value = 'organization';
}

function requestClose(reason: 'cancel' | 'close' | 'overlay') {
  resetDraft();
  open.value = false;
  if (reason === 'cancel') emit('cancel');
  emit('close', reason);
}

function confirm() {
  if (!canConfirm.value) return;
  // defineModel 在受控模式下等待父组件回传新值，确认事件直接使用本次快照避免读取旧 prop。
  const confirmedSelection = [...draftSelection.value];
  modelValue.value = confirmedSelection;
  emit('confirm', confirmedSelection);
  open.value = false;
}

watch(
  open,
  async (visible) => {
    if (!visible) return;
    resetDraft();
    await nextTick();
    dialogRef.value?.focus();
  },
  { immediate: true },
);

watch(modelValue, () => {
  if (!open.value) resetDraft();
});

watch(showCurrentMemberDepartmentTab, (visible) => {
  if (!visible) departmentView.value = 'organization';
});
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      ref="dialog"
      class="evolyn-member-department-role-picker"
      role="dialog"
      aria-modal="true"
      :aria-label="props.title"
      tabindex="-1"
      @keydown.esc="requestClose('close')"
    >
      <div class="evolyn-member-department-role-picker__overlay" @click="requestClose('overlay')" />
      <section class="evolyn-member-department-role-picker__dialog">
        <header class="evolyn-member-department-role-picker__header">
          <h2 class="evolyn-member-department-role-picker__title">{{ props.title }}</h2>
          <button
            class="evolyn-member-department-role-picker__close"
            type="button"
            aria-label="关闭"
            @click="requestClose('close')"
          >
            <RiCloseLargeFill aria-hidden="true" />
          </button>
        </header>

        <div class="evolyn-member-department-role-picker__content">
          <PickerSelectedItems :selections="draftSelection" @remove="remove" />

          <label class="evolyn-member-department-role-picker__search">
            <RiSearchFill aria-hidden="true" />
            <input v-model="keyword" type="search" :placeholder="props.searchPlaceholder" />
          </label>

          <div
            class="evolyn-member-department-role-picker__tabs"
            role="tablist"
            aria-label="选择类型"
          >
            <template v-if="showCurrentMemberDepartmentTab">
              <button
                class="evolyn-member-department-role-picker__tab"
                :class="{
                  'evolyn-member-department-role-picker__tab--active':
                    departmentView === 'organization',
                }"
                type="button"
                role="tab"
                :aria-selected="departmentView === 'organization'"
                @click="departmentView = 'organization'"
              >
                组织架构
              </button>
              <button
                class="evolyn-member-department-role-picker__tab"
                :class="{
                  'evolyn-member-department-role-picker__tab--active':
                    departmentView === 'current-member',
                }"
                type="button"
                role="tab"
                :aria-selected="departmentView === 'current-member'"
                @click="departmentView = 'current-member'"
              >
                当前用户所在部门
              </button>
            </template>
            <template v-else>
              <button
                v-for="type in availableTypes"
                :key="type"
                class="evolyn-member-department-role-picker__tab"
                :class="{
                  'evolyn-member-department-role-picker__tab--active': activeType === type,
                }"
                type="button"
                role="tab"
                :aria-selected="activeType === type"
                @click="activeType = type"
              >
                {{ type === 'department' ? '组织架构' : type === 'role' ? '角色' : '成员' }}
              </button>
            </template>
          </div>

          <section class="evolyn-member-department-role-picker__panel" role="tabpanel">
            <template v-if="activeType === 'department' && departmentView === 'current-member'">
              <PickerVirtualTree
                v-if="currentMemberDepartmentVirtualRows.length"
                item-type="department"
                mode="select"
                :multiple="isMultiple('department')"
                :rows="currentMemberDepartmentVirtualRows"
                :selected-keys="selectedKeys"
                :is-disabled="(item) => isDisabled(item, 'department')"
                @select="toggle($event, 'department')"
              />
              <p v-else class="evolyn-member-department-role-picker__empty">
                {{ props.currentMemberDepartmentEmptyText }}
              </p>
            </template>
            <template v-else-if="activeType === 'department' || activeType === 'role'">
              <PickerVirtualTree
                v-if="
                  activeType === 'department'
                    ? departmentVirtualRows.length
                    : roleVirtualRows.length
                "
                :item-type="activeType === 'department' ? 'department' : 'role'"
                mode="select"
                :multiple="isMultiple(activeType === 'department' ? 'department' : 'role')"
                :rows="activeType === 'department' ? departmentVirtualRows : roleVirtualRows"
                :selected-keys="selectedKeys"
                :is-disabled="
                  (item) => isDisabled(item, activeType === 'department' ? 'department' : 'role')
                "
                @select="toggle($event, activeType === 'department' ? 'department' : 'role')"
                @toggle-expand="
                  activeType === 'department'
                    ? toggleDepartmentExpanded($event)
                    : toggleRoleExpanded($event)
                "
              />
              <p v-else class="evolyn-member-department-role-picker__empty">
                {{ props.emptyText }}
              </p>
            </template>

            <div v-else class="evolyn-member-department-role-picker__member-panel">
              <aside class="evolyn-member-department-role-picker__member-tree">
                <button
                  class="evolyn-member-department-role-picker__all-members"
                  :class="{
                    'evolyn-member-department-role-picker__all-members--active':
                      activeDepartmentId === undefined,
                  }"
                  type="button"
                  @click="activeDepartmentId = undefined"
                >
                  全部成员
                </button>
                <PickerVirtualTree
                  v-if="departmentVirtualRows.length"
                  :active-id="activeDepartmentId"
                  item-type="department"
                  mode="filter"
                  :multiple="true"
                  :rows="departmentVirtualRows"
                  :selected-keys="selectedKeys"
                  :is-disabled="() => false"
                  @select="activeDepartmentId = $event.id"
                  @toggle-expand="toggleDepartmentExpanded"
                />
              </aside>
              <div class="evolyn-member-department-role-picker__member-list">
                <PickerVirtualMembers
                  v-if="visibleMembers.length"
                  :members="visibleMembers"
                  :multiple="isMultiple('member')"
                  :selected-keys="selectedKeys"
                  :is-disabled="(member) => isDisabled(member, 'member')"
                  @select="toggle($event, 'member')"
                />
                <p v-else class="evolyn-member-department-role-picker__empty">
                  {{ props.emptyText }}
                </p>
              </div>
            </div>
          </section>
        </div>

        <footer class="evolyn-member-department-role-picker__footer">
          <span class="evolyn-member-department-role-picker__count">
            已选择 {{ draftSelection.length }}{{ props.max ? `/${props.max}` : '' }}
          </span>
          <div class="evolyn-member-department-role-picker__actions">
            <button type="button" @click="requestClose('cancel')">取消</button>
            <button type="button" :disabled="!canConfirm" @click="confirm">确定</button>
          </div>
        </footer>
      </section>
    </div>
  </Teleport>
</template>

<style lang="scss">
@use './EvolynMemberDepartmentRolePicker.scss' as *;
</style>
