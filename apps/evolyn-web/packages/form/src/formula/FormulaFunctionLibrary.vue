<script setup lang="ts">
import { RiArrowDownSLine, RiArrowRightSLine, RiSearchLine } from '@remixicon/vue';
import { ElIcon } from 'element-plus';
import { computed, shallowRef } from 'vue';
import {
  FORMULA_CATEGORY_LABELS,
  FORMULA_COMMON_FUNCTION_NAMES,
  FORMULA_FUNCTION_BY_NAME,
} from './catalog';
import type { FormulaEditorFunction, FormulaFunctionCategory } from './types';

type FormulaLibraryCategory = 'common' | FormulaFunctionCategory;

const props = defineProps<{
  functions: readonly FormulaEditorFunction[];
}>();

const emit = defineEmits<{
  insert: [functionSpec: FormulaEditorFunction];
}>();

const searchKeyword = shallowRef('');
const expandedCategory = shallowRef<FormulaLibraryCategory | null>('common');
const activeFunctionName = shallowRef<string>('CONCATENATE');

const categories: ReadonlyArray<{ key: FormulaLibraryCategory; label: string }> = [
  { key: 'common', label: '常用函数' },
  ...Object.entries(FORMULA_CATEGORY_LABELS).map(([key, label]) => ({
    key: key as FormulaFunctionCategory,
    label,
  })),
];

const groupedFunctions = computed(() => {
  const keyword = searchKeyword.value.trim().toUpperCase();
  const commonNames = new Set(FORMULA_COMMON_FUNCTION_NAMES);
  return categories
    .filter((category) => !keyword || category.key !== 'common')
    .map((category) => ({
      ...category,
      functions: props.functions.filter((item) => {
        const matchesCategory =
          category.key === 'common'
            ? commonNames.has(item.name as (typeof FORMULA_COMMON_FUNCTION_NAMES)[number])
            : item.category === category.key;
        return (
          matchesCategory &&
          (!keyword || item.name.includes(keyword) || item.description.includes(keyword))
        );
      }),
    }))
    .filter((group) => group.functions.length > 0);
});

const activeFunction = computed(
  () =>
    props.functions.find((item) => item.name === activeFunctionName.value) ??
    groupedFunctions.value.find((group) => group.key === expandedCategory.value)?.functions[0] ??
    FORMULA_FUNCTION_BY_NAME.get('CONCATENATE'),
);

function toggleCategory(category: FormulaLibraryCategory): void {
  if (expandedCategory.value === category) {
    expandedCategory.value = null;
    return;
  }
  expandedCategory.value = category;
  const first = groupedFunctions.value.find((group) => group.key === category)?.functions[0];
  const activeStillInGroup = groupedFunctions.value
    .find((group) => group.key === category)
    ?.functions.some((item) => item.name === activeFunctionName.value);
  if (!activeStillInGroup && first) activeFunctionName.value = first.name;
}

function selectFunction(functionSpec: FormulaEditorFunction): void {
  activeFunctionName.value = functionSpec.name;
}

function insertActiveFunction(): void {
  if (activeFunction.value) emit('insert', activeFunction.value);
}
</script>

<template>
  <div class="formula-function-library">
    <section class="formula-function-library__groups" aria-label="函数分组">
      <label class="formula-function-library__search">
        <el-icon><RiSearchLine /></el-icon>
        <input v-model="searchKeyword" type="search" placeholder="搜索函数" />
      </label>
      <section
        v-for="group in groupedFunctions"
        :key="group.key"
        class="formula-function-library__group"
      >
        <button
          type="button"
          class="formula-function-library__group-heading"
          :class="{ 'is-expanded': expandedCategory === group.key }"
          :aria-expanded="expandedCategory === group.key"
          @click="toggleCategory(group.key)"
        >
          <el-icon>
            <RiArrowDownSLine v-if="expandedCategory === group.key" />
            <RiArrowRightSLine v-else />
          </el-icon>
          {{ group.label }}
        </button>
        <div v-show="expandedCategory === group.key" class="formula-function-library__group-items">
          <button
            v-for="item in group.functions"
            :key="item.name"
            type="button"
            :class="{ 'is-active': activeFunction?.name === item.name }"
            @click="selectFunction(item)"
            @dblclick="emit('insert', item)"
          >
            <strong>{{ item.name }}</strong>
            <small>{{ item.description }}</small>
          </button>
        </div>
      </section>
      <p v-if="groupedFunctions.length === 0" class="formula-function-library__empty">未找到函数</p>
    </section>

    <aside v-if="activeFunction" class="formula-function-library__guide">
      <p class="formula-function-library__category">
        {{ FORMULA_CATEGORY_LABELS[activeFunction.category] }}
      </p>
      <h3>{{ activeFunction.name }}</h3>
      <p>{{ activeFunction.description }}</p>
      <p>
        用法：<code>{{ activeFunction.syntax }}</code>
      </p>
      <button type="button" @click="insertActiveFunction">插入函数</button>
    </aside>
  </div>
</template>

<style scoped lang="scss">
.formula-function-library {
  display: grid;
  grid-template-columns: minmax(250px, 0.85fr) minmax(300px, 1.35fr);
  min-width: 0;
  color: var(--el-text-color-primary);
}

.formula-function-library__groups {
  min-height: 216px;
  max-height: 216px;
  overflow: auto;
  border-right: 1px solid var(--el-border-color);
}

.formula-function-library__groups button {
  width: 100%;
  cursor: pointer;
  background: transparent;
  border: 0;
}

.formula-function-library__group-heading {
  display: flex;
  gap: 6px;
  min-height: 32px;
  padding: 0 10px;
  align-items: center;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.1px;
  text-align: left;
  border-radius: 6px;
}

.formula-function-library__group-items {
  margin: 2px 8px 6px 21px;
  padding-left: 8px;
  border-left: 1px solid var(--el-border-color-lighter);
}

.formula-function-library__group-heading:hover,
.formula-function-library__group-items button:hover {
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}

.formula-function-library__group-heading.is-expanded {
  color: var(--el-color-primary);
  background: transparent;
}

.formula-function-library__search {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  gap: 8px;
  height: 42px;
  padding: 0 12px;
  align-items: center;
  color: var(--el-text-color-placeholder);
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.formula-function-library__search input {
  width: 100%;
  color: var(--el-text-color-primary);
  font: inherit;
  background: transparent;
  border: 0;
  outline: 0;
}

.formula-function-library__group-items button {
  position: relative;
  display: grid;
  gap: 3px;
  min-height: 40px;
  padding: 5px 12px;
  text-align: left;
  border-radius: 0 6px 6px 0;
}

.formula-function-library__group-items button.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}

.formula-function-library__group-items button.is-active small {
  color: var(--el-color-primary-light-3);
}

.formula-function-library__group-items strong {
  font-size: 13px;
  font-weight: 600;
}

.formula-function-library__group-items small {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.formula-function-library__empty {
  padding: 12px 14px;
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.formula-function-library__guide {
  padding: 14px 16px;
}

.formula-function-library__category {
  margin: 0 0 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.formula-function-library__guide h3 {
  margin: 0 0 10px;
  font-size: 16px;
  font-weight: 650;
}

.formula-function-library__guide p {
  margin: 0 0 10px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.5;
}

.formula-function-library__guide code {
  color: #bb42db;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  overflow-wrap: anywhere;
}

.formula-function-library__guide button {
  height: 30px;
  padding: 0 12px;
  margin-top: 8px;
  color: var(--el-color-primary);
  font-size: 13px;
  cursor: pointer;
  background: var(--el-color-primary-light-9);
  border: 0;
  border-radius: 5px;
}

.formula-function-library__guide button:hover {
  color: var(--el-color-white);
  background: var(--el-color-primary);
}

@media (width <= 960px) {
  .formula-function-library {
    grid-template-columns: minmax(220px, 0.85fr) minmax(250px, 1.2fr);
  }
}

@media (width <= 760px) {
  .formula-function-library {
    grid-template-columns: 1fr;
  }

  .formula-function-library__groups {
    max-height: 220px;
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color);
  }
}
</style>
