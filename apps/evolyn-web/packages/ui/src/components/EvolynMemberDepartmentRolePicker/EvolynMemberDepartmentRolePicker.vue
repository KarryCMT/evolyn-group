<script setup lang="ts">
import { RiCloseLargeFill, RiSearchFill } from '@remixicon/vue';
import { computed, nextTick, shallowRef, useTemplateRef, watch } from 'vue';
import PickerMembers from './PickerMembers.vue';
import PickerSelectedItems from './PickerSelectedItems.vue';
import PickerTreeNode from './PickerTreeNode.vue';
import type {
  EvolynMemberDepartmentRolePickerEmits,
  EvolynMemberDepartmentRolePickerProps,
  EvolynMemberDepartmentRolePickerSelection,
} from './EvolynMemberDepartmentRolePicker.types';
import { useMemberDepartmentRolePicker } from './useMemberDepartmentRolePicker';

defineOptions({ name: 'EvolynMemberDepartmentRolePicker' });

const props = withDefaults(defineProps<EvolynMemberDepartmentRolePickerProps>(), {
  title: '选择成员、部门或角色',
  departments: () => [],
  roles: () => [],
  members: () => [],
  selectableTypes: () => ['department', 'role', 'member'],
  multiple: true,
  allowEmpty: false,
  searchPlaceholder: '搜索（多个关键词用空格隔开）',
  emptyText: '暂无可选择的数据',
});

const emit = defineEmits<EvolynMemberDepartmentRolePickerEmits>();
const modelValue = defineModel<EvolynMemberDepartmentRolePickerSelection[]>({ default: () => [] });
const open = defineModel<boolean>('open', { default: false });
const dialogRef = useTemplateRef<HTMLElement>('dialog');
const draftSelection = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);

const {
  activeDepartmentId,
  activeType,
  availableTypes,
  isDisabled,
  keyword,
  remove,
  resetView,
  selectedKeys,
  toggle,
  visibleDepartments,
  visibleMembers,
  visibleRoles,
} = useMemberDepartmentRolePicker({
  departments: () => props.departments,
  roles: () => props.roles,
  members: () => props.members,
  selectableTypes: () => props.selectableTypes,
  multiple: () => props.multiple,
  max: () => props.max,
  selection: draftSelection,
});

const canConfirm = computed(() => props.allowEmpty || draftSelection.value.length > 0);

function resetDraft() {
  draftSelection.value = [...modelValue.value];
  resetView();
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
          </div>

          <section class="evolyn-member-department-role-picker__panel" role="tabpanel">
            <template v-if="activeType === 'department' || activeType === 'role'">
              <ul
                v-if="(activeType === 'department' ? visibleDepartments : visibleRoles).length"
                class="evolyn-member-department-role-picker__tree"
              >
                <PickerTreeNode
                  v-for="node in activeType === 'department' ? visibleDepartments : visibleRoles"
                  :key="node.id"
                  :item-type="activeType === 'department' ? 'department' : 'role'"
                  mode="select"
                  :node="node"
                  :selected-keys="selectedKeys"
                  :is-disabled="
                    (item) => isDisabled(item, activeType === 'department' ? 'department' : 'role')
                  "
                  @select="toggle($event, activeType === 'department' ? 'department' : 'role')"
                />
              </ul>
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
                <ul class="evolyn-member-department-role-picker__tree">
                  <PickerTreeNode
                    v-for="node in visibleDepartments"
                    :key="node.id"
                    :active-id="activeDepartmentId"
                    item-type="department"
                    mode="filter"
                    :node="node"
                    :selected-keys="selectedKeys"
                    :is-disabled="() => false"
                    @select="activeDepartmentId = $event.id"
                  />
                </ul>
              </aside>
              <div class="evolyn-member-department-role-picker__member-list">
                <PickerMembers
                  v-if="visibleMembers.length"
                  :members="visibleMembers"
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
