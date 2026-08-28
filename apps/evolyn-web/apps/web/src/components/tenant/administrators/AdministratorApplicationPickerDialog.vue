<script setup lang="ts">
import {
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCheckboxCircleFill,
  RiCloseFill,
  RiContactsBook3Fill,
  RiPieChart2Fill,
  RiSearch2Line,
} from '@remixicon/vue';
import { markRaw, type Component } from 'vue';
import { computed, shallowRef, watch } from 'vue';
import { getApplicationIconName } from '~/types';
import type { AdministratorApplication } from './administrator.types';

defineOptions({ name: 'AdministratorApplicationPickerDialog' });

const props = defineProps<{ applications: AdministratorApplication[]; selectedIds: number[] }>();
const visible = defineModel<boolean>({ default: false });
const emit = defineEmits<{ confirm: [payload: { ids: number[]; all: boolean }] }>();
const keyword = shallowRef('');
const draftIds = shallowRef<number[]>([]);
const filteredApplications = computed(() => {
  const query = keyword.value.trim();
  return query ? props.applications.filter((app) => app.name.includes(query)) : props.applications;
});
const selectedApps = computed(() =>
  props.applications.filter((app) => draftIds.value.includes(app.id)),
);
const allSelected = computed(() => draftIds.value.length === props.applications.length);

// 图标键 → Remix Fill 图标（键值与后端服务端枚举一致，口径同工作台应用卡片；
// 颜色统一主题色变量）
const iconByKey: Record<string, Component> = {
  bookmark: markRaw(RiBookmark3Fill),
  briefcase: markRaw(RiBriefcase4Fill),
  contacts: markRaw(RiContactsBook3Fill),
  chart: markRaw(RiPieChart2Fill),
  check: markRaw(RiCheckboxCircleFill),
};

function toggleAll(value: string | number | boolean) {
  draftIds.value = value === true ? props.applications.map((app) => app.id) : [];
}
function submit() {
  // 全选落语义全量（allApplications=true）：新建应用自动纳入可编辑范围
  emit('confirm', {
    ids: draftIds.value,
    all: draftIds.value.length === props.applications.length && draftIds.value.length > 0,
  });
  visible.value = false;
}
function remove(id: number) {
  draftIds.value = draftIds.value.filter((item) => item !== id);
}
watch(visible, (isVisible) => {
  if (isVisible) {
    keyword.value = '';
    // 初始勾选：语义全量时展开为全部应用（提交时仍会折叠回全量语义）
    draftIds.value = [...allApplicationsIdsOf(props.selectedIds, props.applications)];
  }
});

/** 详情只存清单或全量标记：全量时勾选展开为全部，否则原样回显。 */
function allApplicationsIdsOf(selectedIds: number[], applications: AdministratorApplication[]) {
  return selectedIds.length === applications.length && applications.length > 0
    ? applications.map((app) => app.id)
    : selectedIds;
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="administrator-application-picker"
    width="944px"
    :show-close="false"
    append-to-body
  >
    <header class="administrator-application-picker__header">
      <h2>应用列表</h2>
      <button type="button" aria-label="关闭" @click="visible = false"><RiCloseFill /></button>
    </header>
    <section class="administrator-application-picker__body">
      <div class="administrator-application-picker__catalog">
        <label class="administrator-application-picker__search"
          ><RiSearch2Line /><input v-model="keyword" placeholder="搜索"
        /></label>
        <el-checkbox
          :model-value="allSelected"
          :indeterminate="draftIds.length > 0 && !allSelected"
          @change="toggleAll"
          >全选</el-checkbox
        >
        <el-scrollbar class="administrator-application-picker__list-scroll">
          <el-checkbox-group v-model="draftIds" class="administrator-application-picker__list">
            <el-checkbox v-for="app in filteredApplications" :key="app.id" :value="app.id"
              ><i class="administrator-application-picker__app-icon">
                <component
                  :is="iconByKey[getApplicationIconName(app.icon)] ?? iconByKey.bookmark"
                /> </i
              >{{ app.name }}</el-checkbox
            >
          </el-checkbox-group>
        </el-scrollbar>
      </div>
      <div class="administrator-application-picker__selected">
        <span
          v-for="app in selectedApps"
          :key="app.id"
          class="administrator-application-picker__tag"
          ><i class="administrator-application-picker__app-icon">
            <component
              :is="iconByKey[getApplicationIconName(app.icon)] ?? iconByKey.bookmark"
            /> </i
          >{{ app.name }}<RiCloseFill @click="remove(app.id)"
        /></span>
      </div>
    </section>
    <footer class="administrator-application-picker__footer">
      <el-button @click="visible = false">取消</el-button
      ><el-button type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.administrator-application-picker) {
  border-radius: var(--el-border-radius-large);
}
:global(.administrator-application-picker .el-dialog__header) {
  display: none;
}
:global(.administrator-application-picker .el-dialog__body) {
  padding: 0;
}
.administrator-application-picker {
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
  &__body {
    display: grid;
    height: 652px;
    grid-template-columns: 1fr 1fr;
    margin: var(--el-space-3xl) var(--el-space-3xl) var(--el-space-xl);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-medium);
    overflow: hidden;
  }
  &__catalog {
    display: flex;
    flex-direction: column;
    min-height: 0;
    padding: var(--el-space-xl) var(--el-space-2xl);
    border-right: 1px solid var(--el-border-color);
    gap: var(--el-space-lg);
  }
  &__search {
    display: flex;
    height: 42px;
    padding: 0 var(--el-space-lg);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-md);
    color: #687383;
    background: #f5f6f8;
  }
  &__search svg {
    width: 20px;
    height: 20px;
  }
  &__search input {
    width: 100%;
    border: 0;
    outline: 0;
    background: transparent;
    font: inherit;
  }
  &__list-scroll {
    flex: 1;
    min-height: 0;
  }
  &__list {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-lg);
  }
  &__list :deep(.el-checkbox) {
    height: 27px;
    margin-right: 0;
  }
  &__list :deep(.el-checkbox__label) {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-lg);
    color: #4c5666;
    font-size: var(--el-font-size-medium);
  }
  &__app-icon {
    display: inline-flex;
    width: 28px;
    height: 28px;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    font-size: var(--el-font-size-extra-small);
    font-style: normal;

    svg {
      width: 16px;
      height: 16px;
    }
  }
  &__selected {
    display: flex;
    padding: var(--el-space-xl) var(--el-space-2xl);
    align-content: flex-start;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: var(--el-space-md);
  }
  &__tag {
    display: inline-flex;
    height: 42px;
    gap: var(--el-space-md);
    padding: 0 var(--el-space-md);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    color: #4d5766;
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-medium);
  }
  &__tag svg {
    margin-left: var(--el-space-xs);
    color: #6d7785;
    cursor: pointer;
  }
  &__tag svg:hover {
    color: var(--el-color-danger);
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
