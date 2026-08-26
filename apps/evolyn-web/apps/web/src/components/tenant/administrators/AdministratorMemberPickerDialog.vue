<script setup lang="ts">
import { RiCloseFill, RiSearchFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import type { DepartmentDto } from '~/api/department';
import type { AdministratorMember, AdministratorPickerMember } from './administrator.types';

defineOptions({ name: 'AdministratorMemberPickerDialog' });

const props = defineProps<{
  members: AdministratorPickerMember[];
  departments: DepartmentDto[];
  selectedMembers: AdministratorMember[];
  /** 租户创建人是固定所有者，不可通过管理组再次授予管理员身份。 */
  tenantOwnerAccountId: number | null;
}>();

const visible = defineModel<boolean>({ default: false });
const emit = defineEmits<{ confirm: [members: AdministratorMember[]] }>();
const keyword = shallowRef('');
const selectedIds = shallowRef<number[]>([]);
// 部门过滤：null = 全部成员；点击树节点切换，再点「全部成员」或同节点取消
const departmentFilterId = shallowRef<number | null>(null);

const keywords = computed(() => keyword.value.trim().split(/\s+/).filter(Boolean));

// 部门 → 整棵子树 ID 集：点父部门命中其下全部成员
const departmentSubtrees = computed(() => {
  const subtrees = new Map<number, Set<number>>();
  const walk = (department: DepartmentDto): Set<number> => {
    const ids = new Set<number>([department.id]);
    for (const child of department.children ?? []) {
      for (const id of walk(child)) ids.add(id);
    }
    subtrees.set(department.id, ids);
    return ids;
  };
  for (const department of props.departments) walk(department);
  return subtrees;
});

const visibleMembers = computed(() => {
  const subtree =
    departmentFilterId.value !== null
      ? departmentSubtrees.value.get(departmentFilterId.value)
      : null;
  return props.members.filter((member) => {
    if (subtree && !member.departmentIds?.some((id) => subtree.has(id))) {
      return false;
    }
    if (!keywords.value.length) return true;
    // 多关键词空格分隔：全部命中（姓名/部门名）才展示
    const haystack = `${member.name}${member.department}`;
    return keywords.value.every((word) => haystack.includes(word));
  });
});
const selectedItems = computed(() =>
  props.members.filter((member) => selectedIds.value.includes(member.id)),
);

function isTenantCreator(member: AdministratorPickerMember) {
  return props.tenantOwnerAccountId !== null && member.accountId === props.tenantOwnerAccountId;
}

function toggleMember(id: number) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((item) => item !== id)
    : [...selectedIds.value, id];
}

function chooseMember(member: AdministratorPickerMember) {
  if (isTenantCreator(member)) {
    ElMessage.warning('企业创建者不能加入任何管理组');
    return;
  }
  toggleMember(member.id);
}

function removeMember(member: AdministratorPickerMember) {
  if (isTenantCreator(member)) {
    ElMessage.warning('企业创建者不能加入任何管理组');
    return;
  }
  selectedIds.value = selectedIds.value.filter((item) => item !== member.id);
}

function onNodeClick(data: unknown) {
  departmentFilterId.value = (data as DepartmentDto).id;
}

function submit() {
  emit('confirm', selectedItems.value);
  visible.value = false;
}

watch(visible, (isVisible) => {
  if (isVisible) {
    keyword.value = '';
    departmentFilterId.value = null;
    selectedIds.value = props.selectedMembers.map((member) => member.id);
  }
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="administrator-member-picker"
    width="1034px"
    :show-close="false"
    append-to-body
  >
    <header class="administrator-member-picker__header">
      <h2>成员列表</h2>
      <button type="button" aria-label="关闭" @click="visible = false"><RiCloseFill /></button>
    </header>
    <section class="administrator-member-picker__selected" aria-label="已选成员">
      <span
        v-for="member in selectedItems"
        :key="member.id"
        class="administrator-member-picker__tag"
      >
        <i>{{ member.name.slice(0, 1) }}</i
        >{{ member.name }}<RiCloseFill @click="removeMember(member)" />
      </span>
    </section>
    <label class="administrator-member-picker__search">
      <RiSearchFill /><input v-model="keyword" placeholder="搜索（多个关键词用空格隔开）" />
    </label>
    <section class="administrator-member-picker__body">
      <div class="administrator-member-picker__tree">
        <el-scrollbar>
          <button
            class="administrator-member-picker__all"
            :class="{ 'administrator-member-picker__all--active': departmentFilterId === null }"
            type="button"
            @click="departmentFilterId = null"
          >
            全部成员
          </button>
          <el-tree
            :data="departments"
            node-key="id"
            default-expand-all
            highlight-current
            :expand-on-click-node="false"
            :props="{ label: 'name', children: 'children' }"
            @node-click="onNodeClick"
          />
        </el-scrollbar>
      </div>
      <div class="administrator-member-picker__results">
        <el-scrollbar>
          <p class="administrator-member-picker__result-title">
            已选 {{ selectedItems.length }}/{{ members.length }}
          </p>
          <label
            v-for="member in visibleMembers"
            :key="member.id"
            class="administrator-member-picker__member"
            :class="{ 'administrator-member-picker__member--creator': isTenantCreator(member) }"
            :title="isTenantCreator(member) ? '企业创建者不能加入任何管理组' : undefined"
            @click.prevent="chooseMember(member)"
          >
            <span class="administrator-member-picker__avatar">{{ member.name.slice(0, 1) }}</span>
            <span>{{ member.name }}</span>
            <span v-if="isTenantCreator(member)" class="administrator-member-picker__creator-tag"
              >创建者</span
            >
            <span class="administrator-member-picker__member-department">{{
              member.department
            }}</span>
            <el-checkbox
              :model-value="selectedIds.includes(member.id)"
              :disabled="isTenantCreator(member)"
            />
          </label>
          <p v-if="visibleMembers.length === 0" class="administrator-member-picker__empty">
            没有符合条件的成员
          </p>
        </el-scrollbar>
      </div>
    </section>
    <footer class="administrator-member-picker__footer">
      <el-button @click="visible = false">取消</el-button
      ><el-button type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.administrator-member-picker) {
  border-radius: 14px;
}
:global(.administrator-member-picker .el-dialog__header) {
  display: none;
}
:global(.administrator-member-picker .el-dialog__body) {
  padding: 0;
}
.administrator-member-picker {
  &__header {
    display: flex;
    height: 68px;
    padding: 0 28px;
    border-bottom: 1px solid #dde2ea;
    align-items: center;
    justify-content: space-between;
  }
  &__header h2 {
    margin: 0;
    color: #273142;
    font-size: 22px;
  }
  &__header button {
    display: inline-flex;
    border: 0;
    padding: 5px;
    color: #66707e;
    background: transparent;
    cursor: pointer;
  }
  &__header button:hover {
    border-radius: 6px;
    background: var(--el-fill-color-light);
  }
  &__header svg {
    width: 25px;
    height: 25px;
  }
  &__selected {
    display: flex;
    min-height: 132px;
    max-height: 176px;
    margin: 26px 28px 16px;
    padding: 12px;
    border: 1px dashed #dae0e9;
    border-radius: 9px;
    align-items: flex-start;
    gap: 8px;
    overflow-y: auto;
  }
  &__tag {
    display: inline-flex;
    height: 42px;
    gap: 7px;
    padding: 0 12px;
    border-radius: 7px;
    align-items: center;
    color: #4e5868;
    background: var(--el-fill-color-light);
    font-size: 17px;
  }
  &__tag i,
  &__avatar {
    display: inline-flex;
    width: 25px;
    height: 25px;
    border-radius: 50%;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    font-size: 14px;
    font-style: normal;
  }
  &__tag svg {
    margin-left: 48px;
    cursor: pointer;
  }
  &__tag svg:hover {
    color: var(--el-color-danger);
  }
  &__search {
    display: flex;
    height: 44px;
    margin: 0 28px;
    padding: 0 12px;
    border-radius: 7px;
    align-items: center;
    gap: 8px;
    background: #f5f6f8;
    color: #697384;
  }
  &__search svg {
    width: 20px;
    height: 20px;
  }
  &__search input {
    width: 100%;
    border: 0;
    outline: 0;
    color: #4e5868;
    background: transparent;
    font: inherit;
  }
  &__body {
    display: grid;
    min-height: 460px;
    margin: 6px 28px 0;
    grid-template-columns: 1fr 1fr;
    border-top: 1px solid #edf0f4;
  }
  &__tree,
  &__results {
    padding: 14px 12px;
  }
  &__tree {
    border-right: 1px solid #e2e6ec;
  }
  &__all {
    display: block;
    width: 100%;
    min-height: 40px;
    margin: 0 0 9px;
    padding: 0 16px;
    border: 0;
    border-radius: 7px;
    color: #4f5968;
    background: transparent;
    font-size: 17px;
    text-align: left;
    cursor: pointer;
  }
  &__all:hover {
    background: var(--el-fill-color-light);
  }
  &__all--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__tree :deep(.el-tree) {
    --el-tree-node-content-height: 38px;
    background: transparent;
    font-size: 17px;
  }
  &__result-title {
    margin: 0 4px 12px;
    color: var(--el-color-primary);
    font-size: 17px;
  }
  &__member {
    display: flex;
    height: 43px;
    padding: 0 4px;
    border-radius: 6px;
    align-items: center;
    gap: 8px;
    color: #4e5868;
    font-size: 17px;
    cursor: pointer;
  }
  &__member:hover {
    background: var(--el-fill-color-light);
  }
  &__member--creator {
    color: var(--el-text-color-secondary);
    cursor: not-allowed;
  }
  &__member--creator:hover {
    background: var(--el-fill-color-light);
  }
  &__member .el-checkbox {
    margin-left: auto;
  }
  &__creator-tag {
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--el-color-warning);
    background: var(--el-color-warning-light-9);
    font-size: 12px;
  }
  &__member-department {
    color: #98a1af;
    font-size: 15px;
  }
  &__empty {
    margin: 0;
    padding: 24px 0;
    color: #909aa8;
    font-size: 16px;
    text-align: center;
  }
  &__footer {
    display: flex;
    height: 76px;
    padding: 0 28px;
    border-top: 1px solid #dde2ea;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
  }
  &__footer .el-button {
    min-width: 74px;
    height: 42px;
    font-size: 17px;
  }
}
</style>
