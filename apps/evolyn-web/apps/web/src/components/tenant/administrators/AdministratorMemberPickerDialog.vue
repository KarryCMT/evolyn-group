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
            highlight-current:
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
  border-radius: var(--el-border-radius-large);
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
    padding: 0 var(--el-space-3xl);
    border-bottom: 1px solid var(--el-border-color);
    align-items: center;
    justify-content: space-between;
  }
  &__header h2 {
    margin: 0;
    color: #273142;
    font-size: var(--el-font-size-medium);
  }
  &__header button {
    display: inline-flex;
    border: 0;
    padding: var(--el-space-xs);
    color: #66707e;
    background: transparent;
    cursor: pointer;
  }
  &__header button:hover {
    border-radius: var(--el-border-radius-medium);
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
    margin: var(--el-space-3xl) var(--el-space-3xl) var(--el-space-xl);
    padding: var(--el-space-lg);
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-large);
    align-items: flex-start;
    gap: var(--el-space-md);
    overflow-y: auto;
  }
  &__tag {
    display: inline-flex;
    height: 42px;
    gap: var(--el-space-sm);
    padding: 0 var(--el-space-lg);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    color: #4e5868;
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-medium);
  }
  &__tag i,
  &__avatar {
    display: inline-flex;
    width: 25px;
    height: 25px;
    border-radius: var(--el-border-radius-half);
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    font-size: var(--el-font-size-base);
    font-style: normal;
  }
  &__tag svg {
    margin-left: var(--el-space-6xl);
    cursor: pointer;
  }
  &__tag svg:hover {
    color: var(--el-color-danger);
  }
  &__search {
    display: flex;
    height: 44px;
    margin: 0 var(--el-space-3xl);
    padding: 0 var(--el-space-lg);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-md);
    background: var(--el-bg-color-page);
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
    margin: var(--el-space-sm) var(--el-space-3xl) 0;
    grid-template-columns: 1fr 1fr;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  &__tree,
  &__results {
    padding: var(--el-space-lg) var(--el-space-lg);
  }
  &__tree {
    border-right: 1px solid var(--el-border-color-light);
  }
  &__all {
    display: block;
    width: 100%;
    min-height: 40px;
    margin: 0 0 var(--el-space-md);
    padding: 0 var(--el-space-xl);
    border: 0;
    border-radius: var(--el-border-radius-medium);
    color: #4f5968;
    background: transparent;
    font-size: var(--el-font-size-medium);
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
    font-size: var(--el-font-size-medium);
  }
  &__result-title {
    margin: 0 var(--el-space-xs) var(--el-space-lg);
    color: var(--el-color-primary);
    font-size: var(--el-font-size-medium);
  }
  &__member {
    display: flex;
    height: 43px;
    padding: 0 var(--el-space-xs);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-md);
    color: #4e5868;
    font-size: var(--el-font-size-medium);
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
    padding: var(--el-space-xs) var(--el-space-sm);
    border-radius: var(--el-border-radius-base);
    color: var(--el-color-warning);
    background: var(--el-color-warning-light-9);
    font-size: var(--el-font-size-extra-small);
  }
  &__member-department {
    color: #98a1af;
    font-size: var(--el-font-size-base);
  }
  &__empty {
    margin: 0;
    padding: var(--el-space-3xl) 0;
    color: #909aa8;
    font-size: var(--el-font-size-medium);
    text-align: center;
  }
  &__footer {
    display: flex;
    height: 76px;
    padding: 0 var(--el-space-3xl);
    border-top: 1px solid var(--el-border-color);
    align-items: center;
    justify-content: flex-end;
    gap: var(--el-space-lg);
  }
  &__footer .el-button {
    min-width: 74px;
    height: 42px;
    font-size: var(--el-font-size-medium);
  }
}
</style>
