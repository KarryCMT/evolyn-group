<script setup lang="ts">
import type { DepartmentDto } from '~/api/department';
import type { MemberListItemDto } from '~/api/member';
import { RiCloseFill, RiGroupFill, RiSearchFill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import { getDepartmentTree } from '~/api/department';
import { listMembers } from '~/api/member';

defineOptions({ name: 'MemberPickerDialog' });

const props = withDefaults(
  defineProps<{
    /** 单选与多选共用弹窗，选择模式由字段类型决定。 */
    multiple: boolean;
    selectedIds: string[];
    scope?: 'tenant' | 'department';
    departmentIds?: string[];
  }>(),
  { scope: 'tenant', departmentIds: () => [] },
);

const emit = defineEmits<{ confirm: [members: MemberListItemDto[]] }>();

const visible = defineModel<boolean>({ default: false });

const loading = shallowRef(false);
const loadError = shallowRef('');
const keyword = shallowRef('');
const activeTab = shallowRef<'members' | 'dynamic'>('members');
const departments = shallowRef<DepartmentDto[]>([]);
const members = shallowRef<MemberListItemDto[]>([]);
const selected = shallowRef<string[]>([]);
const departmentFilterId = shallowRef<number | null>(null);

const selectedSet = computed(() => new Set(selected.value));
const keywords = computed(() => keyword.value.trim().split(/\s+/).filter(Boolean));
const departmentSubtrees = computed(() => {
  const result = new Map<number, Set<number>>();
  const collect = (department: DepartmentDto): Set<number> => {
    const ids = new Set<number>([department.id]);
    for (const child of department.children ?? []) {
      for (const id of collect(child)) ids.add(id);
    }
    result.set(department.id, ids);
    return ids;
  };
  for (const department of departments.value) collect(department);
  return result;
});
const allowedDepartmentIds = computed(() => new Set(props.departmentIds.map(Number)));
const visibleMembers = computed(() => {
  const selectedDepartmentIds =
    departmentFilterId.value === null
      ? null
      : (departmentSubtrees.value.get(departmentFilterId.value) ?? new Set<number>());
  return members.value.filter((member) => {
    const memberDepartmentIds = member.departments.map((department) => department.id);
    if (
      props.scope === 'department' &&
      !memberDepartmentIds.some((id) => allowedDepartmentIds.value.has(id))
    ) {
      return false;
    }
    if (selectedDepartmentIds && !memberDepartmentIds.some((id) => selectedDepartmentIds.has(id))) {
      return false;
    }
    if (!keywords.value.length) return true;
    const searchableText = `${member.name} ${member.departments.map((item) => item.name).join(' ')}`;
    return keywords.value.every((word) => searchableText.includes(word));
  });
});
function memberName(id: string): string {
  return members.value.find((member) => String(member.id) === id)?.name ?? `成员 ${id}`;
}

function toggleMember(memberId: string): void {
  if (props.multiple) {
    selected.value = selectedSet.value.has(memberId)
      ? selected.value.filter((id) => id !== memberId)
      : [...selected.value, memberId];
    return;
  }
  selected.value = selected.value[0] === memberId ? [] : [memberId];
}

function removeMember(memberId: string): void {
  selected.value = selected.value.filter((id) => id !== memberId);
}

function confirm(): void {
  const selectedMembers = selected.value
    .map((id) => members.value.find((member) => String(member.id) === id))
    .filter((member): member is MemberListItemDto => Boolean(member));
  emit('confirm', props.multiple ? selectedMembers : selectedMembers.slice(0, 1));
  visible.value = false;
}

async function loadDirectory(): Promise<void> {
  loading.value = true;
  loadError.value = '';
  try {
    // 当前成员接口支持分页；选择弹窗先取可用上限，后续可无缝替换为服务端关键词分页。
    const [tree, page] = await Promise.all([
      getDepartmentTree(),
      listMembers({ status: 'active', page: 1, pageSize: 500 }),
    ]);
    departments.value = tree;
    members.value = page.items;
  } catch {
    loadError.value = '成员目录加载失败，请稍后重试';
  } finally {
    loading.value = false;
  }
}

watch(visible, (open) => {
  if (!open) return;
  selected.value = props.multiple ? [...props.selectedIds] : props.selectedIds.slice(0, 1);
  keyword.value = '';
  departmentFilterId.value = null;
  activeTab.value = 'members';
  void loadDirectory();
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="form-member-picker"
    width="680px"
    :show-close="false"
    append-to-body
  >
    <header class="form-member-picker__header">
      <h2>成员列表</h2>
      <button type="button" aria-label="关闭" @click="visible = false">
        <RiCloseFill />
      </button>
    </header>
    <section class="form-member-picker__selected" aria-label="已选成员">
      <span v-for="id in selected" :key="id" class="form-member-picker__tag">
        <i>{{ memberName(id).slice(0, 1) }}</i
        >{{ memberName(id) }}
        <RiCloseFill @click="removeMember(id)" />
      </span>
    </section>
    <label class="form-member-picker__search">
      <RiSearchFill /><input v-model="keyword" placeholder="搜索（多个关键词用空格隔开）" />
    </label>
    <div class="form-member-picker__tabs" role="tablist" aria-label="成员选择来源">
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'members'"
        :class="{ 'form-member-picker__tab--active': activeTab === 'members' }"
        @click="activeTab = 'members'"
      >
        成员
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'dynamic'"
        :class="{ 'form-member-picker__tab--active': activeTab === 'dynamic' }"
        @click="activeTab = 'dynamic'"
      >
        动态参数
      </button>
    </div>
    <section v-if="activeTab === 'members'" v-loading="loading" class="form-member-picker__body">
      <div class="form-member-picker__tree">
        <button
          class="form-member-picker__all"
          :class="{ 'form-member-picker__all--active': departmentFilterId === null }"
          type="button"
          @click="departmentFilterId = null"
        >
          <RiGroupFill />全部成员
        </button>
        <el-tree
          :data="departments"
          node-key="id"
          default-expand-all
          highlight-current
          :expand-on-click-node="false"
          :props="{ label: 'name', children: 'children' }"
          @node-click="departmentFilterId = ($event as DepartmentDto).id"
        />
      </div>
      <div class="form-member-picker__results">
        <p v-if="!loadError" class="form-member-picker__result-title">
          已选 {{ selected.length }}/{{ members.length }}
        </p>
        <p v-else class="form-member-picker__empty">
          {{ loadError }}
        </p>
        <label
          v-for="member in visibleMembers"
          :key="member.id"
          class="form-member-picker__member"
          @click.prevent="toggleMember(String(member.id))"
        >
          <span class="form-member-picker__avatar">{{ member.name.slice(0, 1) }}</span>
          <span>{{ member.name }}</span>
          <span class="form-member-picker__member-department">{{
            member.departments.map((item) => item.name).join('、')
          }}</span>
          <el-checkbox v-if="multiple" :model-value="selectedSet.has(String(member.id))" />
          <el-radio v-else :model-value="selected[0]" :value="String(member.id)" />
        </label>
        <p
          v-if="!loading && !loadError && !visibleMembers.length"
          class="form-member-picker__empty"
        >
          没有符合条件的成员
        </p>
      </div>
    </section>
    <section v-else class="form-member-picker__dynamic-empty">动态参数将在流程表单中提供。</section>
    <footer class="form-member-picker__footer">
      <el-button @click="visible = false"> 取消 </el-button
      ><el-button type="primary" @click="confirm"> 确定 </el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.form-member-picker) {
  border-radius: var(--el-border-radius-large);
}
:global(.form-member-picker .el-dialog__header) {
  display: none;
}
:global(.form-member-picker .el-dialog__body) {
  padding: 0;
}
.form-member-picker {
  &__header,
  &__footer,
  &__member,
  &__all,
  &__search,
  &__tag {
    display: flex;
    align-items: center;
  }
  &__header {
    height: 46px;
    padding: 0 var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color);
    justify-content: space-between;
  }
  &__header h2 {
    margin: 0;
    font-size: var(--el-font-size-medium);
  }
  &__header button {
    display: inline-flex;
    padding: var(--el-space-xs);
    border: 0;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;
  }
  &__header svg {
    width: 20px;
    height: 20px;
  }
  &__selected {
    display: flex;
    min-height: 44px;
    max-height: 108px;
    margin: var(--el-space-xl) var(--el-space-xl) var(--el-space-md);
    padding: var(--el-space-sm);
    gap: var(--el-space-sm);
    overflow-y: auto;
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-base);
  }
  &__tag {
    height: 28px;
    padding: 0 var(--el-space-sm);
    gap: var(--el-space-xs);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-small);
    font-size: var(--el-font-size-small);
  }
  &__tag i,
  &__avatar {
    display: inline-flex;
    width: 20px;
    height: 20px;
    align-items: center;
    justify-content: center;
    color: var(--el-color-white);
    background: var(--el-color-primary);
    border-radius: var(--el-border-radius-half);
    font-size: var(--el-font-size-extra-small);
    font-style: normal;
  }
  &__tag svg {
    cursor: pointer;
  }
  &__search {
    height: 30px;
    margin: 0 var(--el-space-xl);
    padding: 0 var(--el-space-sm);
    gap: var(--el-space-xs);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-small);
  }
  &__search input {
    width: 100%;
    border: 0;
    outline: 0;
    color: var(--el-text-color-regular);
    background: transparent;
    font: inherit;
    font-size: var(--el-font-size-small);
  }
  &__tabs {
    display: flex;
    gap: var(--el-space-xl);
    height: 42px;
    margin: 0 var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  &__tabs button {
    padding: 0;
    border: 0;
    border-bottom: 2px solid transparent;
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: pointer;
    font: inherit;
    font-size: var(--el-font-size-small);
  }
  &__tab--active {
    color: var(--el-color-primary) !important;
    border-bottom-color: var(--el-color-primary) !important;
  }
  &__body {
    display: grid;
    min-height: 274px;
    margin: var(--el-space-sm) var(--el-space-xl) 0;
    grid-template-columns: 1fr 1fr;
  }
  &__tree {
    padding: var(--el-space-sm);
    border-right: 1px solid var(--el-border-color-lighter);
  }
  &__tree :deep(.el-tree) {
    --el-tree-node-content-height: 30px;
    background: transparent;
    font-size: var(--el-font-size-small);
  }
  &__all {
    width: 100%;
    height: 32px;
    padding: 0 var(--el-space-sm);
    gap: var(--el-space-xs);
    border: 0;
    border-radius: var(--el-border-radius-small);
    color: var(--el-text-color-regular);
    background: transparent;
    text-align: left;
    cursor: pointer;
  }
  &__all--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__results {
    min-width: 0;
    padding: var(--el-space-sm);
  }
  &__result-title {
    margin: 0 0 var(--el-space-sm);
    color: var(--el-color-primary);
    font-size: var(--el-font-size-small);
  }
  &__member {
    height: 32px;
    padding: 0 var(--el-space-xs);
    gap: var(--el-space-sm);
    border-radius: var(--el-border-radius-small);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-small);
    cursor: pointer;
  }
  &__member:hover {
    background: var(--el-fill-color-light);
  }
  &__member .el-checkbox,
  &__member .el-radio {
    margin-left: auto;
  }
  &__member-department {
    overflow: hidden;
    margin-left: auto;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--el-font-size-extra-small);
  }
  &__empty,
  &__dynamic-empty {
    padding: var(--el-space-3xl);
    margin: 0;
    color: var(--el-text-color-secondary);
    text-align: center;
    font-size: var(--el-font-size-small);
  }
  &__dynamic-empty {
    min-height: 274px;
  }
  &__footer {
    height: 52px;
    padding: 0 var(--el-space-xl);
    border-top: 1px solid var(--el-border-color);
    justify-content: flex-end;
    gap: var(--el-space-sm);
  }
}
</style>
