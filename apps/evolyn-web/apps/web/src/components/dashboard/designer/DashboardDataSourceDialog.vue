<script setup lang="ts">
import type { DashboardFormDataSource } from '~/types';
import { RiAddLine, RiFileList3Line, RiSearchLine } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';

const props = defineProps<{
  sources: DashboardFormDataSource[];
  loading: boolean;
  errorMessage: string;
}>();
const emit = defineEmits<{
  confirm: [source: DashboardFormDataSource];
  retry: [];
  newForm: [];
}>();
const open = defineModel<boolean>({ required: true });

const search = shallowRef('');
const selectedCode = shallowRef('');
const visibleSources = computed(() => {
  const keyword = search.value.trim().toLocaleLowerCase();
  if (!keyword) return props.sources;
  return props.sources.filter((source) => source.name.toLocaleLowerCase().includes(keyword));
});
const selectedSource = computed(
  () => props.sources.find((source) => source.code === selectedCode.value) ?? null,
);

watch(open, (visible) => {
  if (!visible) return;
  search.value = '';
  selectedCode.value = '';
});

function confirm() {
  if (!selectedSource.value) return;
  emit('confirm', selectedSource.value);
}
</script>

<template>
  <el-dialog
    v-model="open"
    class="data-source-dialog"
    width="612px"
    :close-on-click-modal="false"
    align-center
  >
    <template #header>
      <div class="data-source-dialog__title">
        <strong>添加图表</strong>
        <span>请选择图表数据源</span>
      </div>
    </template>

    <div class="data-source-dialog__body">
      <label class="data-source-dialog__search">
        <RiSearchLine aria-hidden="true" />
        <input v-model="search" type="search" placeholder="搜索" aria-label="搜索数据源">
      </label>

      <div class="data-source-dialog__tabs" role="tablist" aria-label="数据源类型">
        <button class="is-active" type="button" role="tab" aria-selected="true">
          表单
        </button>
        <el-tooltip content="数据流暂未开放" placement="top">
          <button type="button" role="tab" aria-selected="false" disabled>
            数据流
          </button>
        </el-tooltip>
        <el-tooltip content="聚合表暂未开放" placement="top">
          <button type="button" role="tab" aria-selected="false" disabled>
            聚合表
          </button>
        </el-tooltip>
      </div>

      <div v-loading="loading" class="data-source-dialog__list" role="listbox">
        <el-alert
          v-if="errorMessage"
          :title="errorMessage"
          type="error"
          :closable="false"
          show-icon
        >
          <template #default>
            <button class="data-source-dialog__retry" type="button" @click="emit('retry')">
              重新加载
            </button>
          </template>
        </el-alert>
        <button
          v-for="(source, index) in visibleSources"
          :key="source.code"
          class="data-source-dialog__item"
          :class="{ 'is-selected': selectedCode === source.code }"
          type="button"
          role="option"
          :aria-selected="selectedCode === source.code"
          @click="selectedCode = source.code"
          @dblclick="selectedCode = source.code; confirm()"
        >
          <span class="data-source-dialog__item-icon" :class="`tone-${(index % 3) + 1}`">
            <RiFileList3Line aria-hidden="true" />
          </span>
          <span>{{ source.name }}</span>
        </button>
        <el-empty
          v-if="!loading && !errorMessage && visibleSources.length === 0"
          :description="search ? '未找到匹配的数据源' : '暂无可用的已发布表单'"
          :image-size="72"
        />
      </div>
    </div>

    <template #footer>
      <div class="data-source-dialog__footer">
        <el-button @click="emit('newForm')">
          <RiAddLine />新建表单
        </el-button>
        <div>
          <el-button @click="open = false">
            取消
          </el-button>
          <el-button type="primary" :disabled="!selectedSource" @click="confirm">
            确定
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
/* Element Plus Dialog 通过 Teleport 挂载，外层尺寸使用 :global 约束。 */
:global(.data-source-dialog) {
  max-width: calc(100vw - 40px);
  height: min(708px, calc(100vh - 40px));
  margin: 0;
  overflow: hidden;
  border-radius: 12px;
}

:global(.data-source-dialog .el-dialog__header) {
  padding: 22px 28px 18px;
  margin: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

:global(.data-source-dialog .el-dialog__body) {
  height: calc(100% - 148px);
  padding: 0;
}

:global(.data-source-dialog .el-dialog__footer) {
  padding: 16px 28px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.data-source-dialog__title {
  display: flex;
  gap: 14px;
  align-items: baseline;
}

.data-source-dialog__title strong {
  font-size: 20px;
  font-weight: 600;
}

.data-source-dialog__title span {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.data-source-dialog__body {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 28px;
}

.data-source-dialog__search {
  display: flex;
  gap: 10px;
  align-items: center;
  height: 42px;
  padding: 0 14px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border: 1px solid transparent;
  border-radius: 6px;
}

.data-source-dialog__search:focus-within {
  border-color: var(--el-color-primary);
}

.data-source-dialog__search svg {
  flex: none;
  width: 18px;
}

.data-source-dialog__search input {
  width: 100%;
  color: var(--el-text-color-primary);
  outline: 0;
  background: transparent;
  border: 0;
}

.data-source-dialog__tabs {
  display: flex;
  gap: 30px;
  height: 56px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.data-source-dialog__tabs button {
  position: relative;
  padding: 0;
  font: inherit;
  color: var(--el-text-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
}

.data-source-dialog__tabs button.is-active {
  font-weight: 600;
  color: var(--el-color-primary);
}

.data-source-dialog__tabs button.is-active::after {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 2px;
  content: '';
  background: var(--el-color-primary);
}

.data-source-dialog__tabs button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.data-source-dialog__list {
  flex: 1;
  min-height: 0;
  padding: 8px 0;
  overflow: auto;
}

.data-source-dialog__item {
  display: flex;
  gap: 10px;
  align-items: center;
  width: 100%;
  min-height: 42px;
  padding: 0 10px;
  font: inherit;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 5px;
}

.data-source-dialog__item:hover,
.data-source-dialog__item.is-selected {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.data-source-dialog__item.is-selected {
  border-color: var(--el-color-primary-light-7);
}

.data-source-dialog__item-icon {
  display: grid;
  flex: none;
  width: 22px;
  height: 22px;
  color: #fff;
  background: #2ba8e8;
  border-radius: 4px;
  place-items: center;
}

.data-source-dialog__item-icon.tone-2 {
  background: #ff9b31;
}

.data-source-dialog__item-icon.tone-3 {
  background: #5bb9eb;
}

.data-source-dialog__item-icon svg {
  width: 15px;
}

.data-source-dialog__retry {
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
}

.data-source-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.data-source-dialog__footer > div {
  display: flex;
  gap: 8px;
}
</style>
