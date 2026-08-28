<script setup lang="ts">
import { RiArrowDownSFill, RiArrowRightSFill, RiSearchFill } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type { PermissionNode, PermissionSubjectType, SubjectNode } from './permissionQuery.types';

defineOptions({ name: 'PermissionQueryExplorer' });

const props = defineProps<{
  selectedId: string;
  subjectTrees: Record<PermissionSubjectType, SubjectNode[]>;
  subjectType: PermissionSubjectType;
  tree: PermissionNode[];
}>();
const emit = defineEmits<{ select: [type: PermissionSubjectType, id: string] }>();
const keyword = shallowRef('');
const subjectTabs: { id: PermissionSubjectType; label: string; placeholder: string }[] = [
  { id: 'member', label: '成员', placeholder: '请输入成员姓名' },
  { id: 'department', label: '部门', placeholder: '请输入部门名称' },
  { id: 'role', label: '角色', placeholder: '请输入角色名称' },
  { id: 'application', label: '应用', placeholder: '请输入名称来搜索' },
];
const activeTab = computed(
  () => subjectTabs.find((item) => item.id === props.subjectType) ?? subjectTabs[0],
);
const visibleNodes = computed(() => {
  const query = keyword.value.trim();
  if (!query) return props.subjectTrees[props.subjectType];
  return props.subjectTrees[props.subjectType].filter(
    (node) =>
      node.label.includes(query) || node.children?.some((child) => child.label.includes(query)),
  );
});
function selectTab(id: PermissionSubjectType) {
  keyword.value = '';
  emit(
    'select',
    id,
    props.subjectTrees[id][0]?.children?.[0]?.id ?? props.subjectTrees[id][0]?.id ?? '',
  );
}
</script>

<template>
  <section class="permission-query-explorer">
    <aside class="permission-query-explorer__subjects">
      <nav class="permission-query-explorer__tabs" role="tablist">
        <button
          v-for="tab in subjectTabs"
          :key="tab.id"
          class="permission-query-explorer__tab"
          :class="{ 'permission-query-explorer__tab--active': tab.id === props.subjectType }"
          type="button"
          @click="selectTab(tab.id)"
        >
          {{ tab.label }}
        </button>
      </nav>
      <label class="permission-query-explorer__search">
        <RiSearchFill /><input v-model="keyword" :placeholder="activeTab.placeholder" />
      </label>
      <el-scrollbar class="permission-query-explorer__subject-scrollbar">
        <div class="permission-query-explorer__subject-tree">
          <template v-for="node in visibleNodes" :key="node.id">
            <button
              class="permission-query-explorer__node"
              :class="{ 'permission-query-explorer__node--active': node.id === props.selectedId }"
              type="button"
              @click="emit('select', props.subjectType, node.id)"
            >
              <component
                v-if="node.children"
                :is="node.expanded ? RiArrowDownSFill : RiArrowRightSFill"
                class="permission-query-explorer__chevron"
              />
              <span v-else class="permission-query-explorer__chevron" />
              <component :is="node.icon" class="permission-query-explorer__node-icon" />
              <span>{{ node.label }}</span>
            </button>
            <button
              v-for="child in node.children"
              :key="child.id"
              class="permission-query-explorer__node permission-query-explorer__node--child"
              :class="{ 'permission-query-explorer__node--active': child.id === props.selectedId }"
              type="button"
              @click="emit('select', props.subjectType, child.id)"
            >
              <component :is="child.icon" class="permission-query-explorer__node-icon" />
              <span>{{ child.label }}</span>
            </button>
          </template>
        </div>
      </el-scrollbar>
    </aside>
    <section class="permission-query-explorer__permissions">
      <header><span>表单/仪表盘权限</span><span>操作</span></header>
      <el-scrollbar class="permission-query-explorer__permission-scrollbar">
        <div class="permission-query-explorer__permission-tree">
          <template v-for="node in props.tree" :key="node.id">
            <div class="permission-query-explorer__permission-node">
              <RiArrowDownSFill /><component :is="node.icon" /><span>{{ node.label }}</span>
            </div>
            <template v-for="folder in node.children" :key="folder.id">
              <div
                class="permission-query-explorer__permission-node permission-query-explorer__permission-node--level-1"
              >
                <component :is="folder.children ? RiArrowDownSFill : RiArrowRightSFill" />
                <component :is="folder.icon" /><span>{{ folder.label }}</span>
              </div>
              <template v-for="form in folder.children" :key="form.id">
                <div
                  class="permission-query-explorer__permission-node permission-query-explorer__permission-node--level-2"
                >
                  <RiArrowDownSFill /><component :is="form.icon" /><span>{{ form.label }}</span>
                </div>
                <div
                  v-for="permission in form.children"
                  :key="permission.id"
                  class="permission-query-explorer__permission-node permission-query-explorer__permission-node--level-3"
                >
                  <span>{{ permission.label }}</span>
                  <div>
                    <button type="button">调整成员</button><button type="button">调整权限</button>
                  </div>
                </div>
              </template>
            </template>
          </template>
        </div>
      </el-scrollbar>
    </section>
  </section>
</template>

<style scoped lang="scss">
.permission-query-explorer {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: 356px minmax(0, 1fr);
}
.permission-query-explorer__subjects {
  display: flex;
  min-width: 0;
  flex-direction: column;
  padding: var(--el-space-3xl);
  border-right: 1px solid var(--el-border-color);
}
.permission-query-explorer__tabs {
  display: grid;
  height: 44px;
  padding: var(--el-space-xs);
  grid-template-columns: repeat(4, 1fr);
  border-radius: var(--el-border-radius-medium);
  background: var(--el-fill-color);
}
.permission-query-explorer__tab {
  border: 0;
  border-radius: var(--el-border-radius-medium);
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--el-font-size-medium);
}
.permission-query-explorer__tab:hover {
  background: var(--el-fill-color-light);
}
.permission-query-explorer__tab--active {
  color: var(--el-color-primary);
  background: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-light);
}
.permission-query-explorer__search {
  display: flex;
  height: 44px;
  margin: var(--el-space-xl) 0;
  padding: 0 var(--el-space-lg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-medium);
  align-items: center;
  gap: var(--el-space-md);
}
.permission-query-explorer__search svg {
  width: 20px;
  height: 20px;
  color: var(--el-text-color-secondary);
}
.permission-query-explorer__search input {
  width: 100%;
  border: 0;
  outline: 0;
  color: var(--el-text-color-primary);
  background: transparent;
  font: inherit;
}
.permission-query-explorer__subject-scrollbar,
.permission-query-explorer__permission-scrollbar {
  min-height: 0;
  flex: 1;
}
.permission-query-explorer__subject-tree {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-xs);
}
.permission-query-explorer__node {
  display: flex;
  min-height: 40px;
  padding: 0 var(--el-space-lg);
  border: 0;
  border-radius: var(--el-border-radius-medium);
  align-items: center;
  gap: var(--el-space-md);
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--el-font-size-medium);
  text-align: left;
}
.permission-query-explorer__node:hover {
  background: var(--el-fill-color-light);
}
.permission-query-explorer__node--active {
  background: var(--el-color-primary-light-9);
}
.permission-query-explorer__node--child {
  padding-left: var(--el-space-6xl);
}
.permission-query-explorer__chevron {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.permission-query-explorer__node-icon {
  width: 20px;
  height: 20px;
  color: var(--el-color-primary);
}
.permission-query-explorer__permissions {
  display: flex;
  min-width: 0;
  flex-direction: column;
  padding: var(--el-space-3xl) var(--el-space-3xl) 0;
}
.permission-query-explorer__permissions header {
  display: flex;
  min-height: 56px;
  padding: 0 var(--el-space-xl);
  align-items: center;
  justify-content: space-between;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  font-size: var(--el-font-size-medium);
  font-weight: 600;
}
.permission-query-explorer__permission-tree {
  padding-top: var(--el-space-md);
}
.permission-query-explorer__permission-node {
  display: flex;
  min-height: 54px;
  align-items: center;
  gap: var(--el-space-md);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
}
.permission-query-explorer__permission-node svg {
  width: 20px;
  height: 20px;
  color: var(--el-color-primary);
}
.permission-query-explorer__permission-node--level-1 {
  padding-left: var(--el-space-3xl);
}
.permission-query-explorer__permission-node--level-2 {
  padding-left: 56px;
}
.permission-query-explorer__permission-node--level-3 {
  padding-left: 128px;
}
.permission-query-explorer__permission-node--level-3 > div {
  display: flex;
  margin-left: auto;
  gap: var(--el-space-3xl);
}
.permission-query-explorer__permission-node button {
  padding: var(--el-space-xs);
  border: 0;
  color: var(--el-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
}
.permission-query-explorer__permission-node button:hover {
  border-radius: var(--el-border-radius-base);
  background: var(--el-color-primary-light-9);
}
</style>
