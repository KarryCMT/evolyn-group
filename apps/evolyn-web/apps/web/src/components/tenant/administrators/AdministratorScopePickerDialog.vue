<script setup lang="ts">
import { RiCloseFill, RiSearch2Line } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import type { TreeInstance, TreeNodeData } from 'element-plus';
import { getDepartmentTree, type DepartmentDto } from '~/api/department';
import { getOrganizationRoleTree } from '~/api/role';
import type { AdministratorScopeTarget } from './administrator.types';

defineOptions({ name: 'AdministratorScopePickerDialog' });

const props = defineProps<{
  target: AdministratorScopeTarget;
  selectedIds: number[];
}>();
const visible = defineModel<boolean>({ default: false });
const emit = defineEmits<{ confirm: [ids: number[]] }>();

const keyword = shallowRef('');
const loading = shallowRef(false);
const treeRef = shallowRef<TreeInstance>();
// 树节点统一 key：部门/角色/角色组分表自增 ID 可能撞号，按类型加前缀隔离
const departments = shallowRef<DepartmentDto[]>([]);
const roleGroups = shallowRef<Awaited<ReturnType<typeof getOrganizationRoleTree>>['groups']>([]);

const title = computed(() => (props.target === 'department' ? '选择部门' : '选择角色'));

interface ScopeNode {
  key: string;
  label: string;
  kind: 'department' | 'group' | 'role';
  children?: ScopeNode[];
}

/** 关键词过滤：命中节点的祖先路径保留（el-tree filter 由节点自带方法承担）。 */
const treeData = computed<ScopeNode[]>(() => {
  if (props.target === 'department') {
    return departments.value.map((department) => toDepartmentNode(department));
  }
  return roleGroups.value.map((group) => ({
    key: `group-${group.id}`,
    label: group.name,
    kind: 'group',
    children: group.roles.map((role) => ({
      key: `role-${role.id}`,
      label: role.name,
      kind: 'role',
    })),
  }));
});

function toDepartmentNode(department: DepartmentDto): ScopeNode {
  return {
    key: `dept-${department.id}`,
    label: department.name,
    kind: 'department',
    children: department.children?.map((child) => toDepartmentNode(child)),
  };
}

/** 勾选默认值：仅目标层级（部门/角色），父级级联展示由组件自行处理。 */
const defaultCheckedKeys = computed(() =>
  props.selectedIds.map((id) => `${props.target === 'department' ? 'dept' : 'role'}-${id}`),
);

watch(visible, async (isVisible) => {
  if (!isVisible) return;
  keyword.value = '';
  loading.value = true;
  try {
    if (props.target === 'department') {
      departments.value = await getDepartmentTree();
    } else {
      roleGroups.value = (await getOrganizationRoleTree()).groups;
    }
  } catch {
    ElMessage.error('数据加载失败，请稍后重试');
  } finally {
    loading.value = false;
  }
});

function filterNode(value: string, data: TreeNodeData) {
  return !value || String((data as ScopeNode).label).includes(value);
}

watch(keyword, (value) => {
  treeRef.value?.filter(value.trim());
});

function submit() {
  const keys = (treeRef.value?.getCheckedKeys(false) ?? []) as string[];
  const prefix = props.target === 'department' ? 'dept-' : 'role-';
  // 角色树不回传角色组节点：勾选组仅是批量勾选其下角色的快捷方式
  const ids = keys
    .filter((key) => key.startsWith(prefix))
    .map((key) => Number(key.slice(prefix.length)));
  emit('confirm', ids);
  visible.value = false;
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="administrator-scope-picker"
    width="604px"
    :show-close="false"
    append-to-body
  >
    <header class="administrator-scope-picker__header">
      <h2>{{ title }}</h2>
      <button type="button" aria-label="关闭" @click="visible = false"><RiCloseFill /></button>
    </header>
    <label class="administrator-scope-picker__search">
      <RiSearch2Line /><input v-model="keyword" placeholder="搜索" />
    </label>
    <div v-loading="loading" class="administrator-scope-picker__body">
      <el-scrollbar>
        <!-- v-if 随开合重建：default-checked-keys 仅在挂载时生效 -->
        <el-tree
          v-if="visible && !loading"
          ref="treeRef"
          :data="treeData"
          node-key="key"
          show-checkbox
          default-expand-all: default-checked-keys="defaultCheckedKeys"
          :props="{ label: 'label', children: 'children' }"
          :filter-node-method="filterNode"
        />
        <p v-if="!loading && treeData.length === 0" class="administrator-scope-picker__empty">
          暂无可选项
        </p>
      </el-scrollbar>
    </div>
    <footer class="administrator-scope-picker__footer">
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.administrator-scope-picker) {
  border-radius: var(--el-border-radius-large);
}
:global(.administrator-scope-picker .el-dialog__header) {
  display: none;
}
:global(.administrator-scope-picker .el-dialog__body) {
  padding: 0;
}
.administrator-scope-picker {
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
    font-size: var(--el-font-size-extra-large);
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
  &__search {
    display: flex;
    height: 44px;
    margin: var(--el-space-3xl) var(--el-space-3xl) 0;
    padding: 0 var(--el-space-lg);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-md);
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
    height: 420px;
    margin: var(--el-space-xl) var(--el-space-3xl) 0;
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--el-border-radius-medium);
  }
  &__body :deep(.el-tree) {
    --el-tree-node-content-height: 40px;
    padding: var(--el-space-md);
  }
  &__empty {
    margin: 0;
    padding: var(--el-space-3xl);
    color: #909aa8;
    font-size: var(--el-font-size-medium);
    text-align: center;
  }
  &__footer {
    display: flex;
    height: 76px;
    padding: 0 var(--el-space-3xl);
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
