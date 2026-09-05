<script setup lang="ts">
import { ElCheckbox, ElInput, ElPopover, ElScrollbar, ElTooltip } from 'element-plus';
import { RiLayoutColumnFill, RiSearchFill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import type { DataColumn } from '../types.js';

defineOptions({ name: 'DataColumnSettings' });

const props = defineProps<{
  columns: readonly DataColumn[];
  /** 已隐藏列的 field 集合（显隐状态由 DataWorkspace 持有，组件只投影勾选态）。 */
  hidden: ReadonlySet<string>;
}>();

const emit = defineEmits<{
  /** 单列勾选状态翻转；「至少保留一列」约束由 DataWorkspace 裁决。 */
  toggle: [field: string];
  /** 全选行批量翻转；fields 为当前生效范围（搜索过滤后的清单）。 */
  toggleAll: [fields: string[], visible: boolean];
}>();

const open = shallowRef(false);
const keyword = shallowRef('');

// 关闭弹层时清空搜索，避免下次打开残留上次的过滤视图
watch(open, (value) => {
  if (!value) keyword.value = '';
});

/** 空格分词过滤：多个关键词需全部命中列名才展示。 */
const filteredColumns = computed(() => {
  const keywords = keyword.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
  if (!keywords.length) return props.columns;
  return props.columns.filter((column) => {
    const title = column.title.toLowerCase();
    return keywords.every((word) => title.includes(word));
  });
});

/** 全选行只作用于当前过滤结果；半选态 = 过滤范围内可见与隐藏并存。 */
const allVisible = computed(
  () =>
    filteredColumns.value.length > 0 &&
    filteredColumns.value.every((column) => !props.hidden.has(column.field)),
);
const partiallyVisible = computed(
  () =>
    filteredColumns.value.some((column) => !props.hidden.has(column.field)) && !allVisible.value,
);

function onToggleAll(visible: string | number | boolean) {
  emit(
    'toggleAll',
    filteredColumns.value.map((column) => column.field),
    Boolean(visible),
  );
}

const visibleCount = computed(() => props.columns.length - props.hidden.size);

/** 取消勾选后可见列将为 0 时禁用该项，保证表格至少保留一列。 */
function disabledOf(column: DataColumn): boolean {
  return !props.hidden.has(column.field) && visibleCount.value <= 1;
}
</script>

<template>
  <ElPopover
    v-model:visible="open"
    trigger="click"
    placement="bottom-end"
    popper-class="data-column-settings__popper"
  >
    <template #reference>
      <!-- 参考灵衍云工具型操作用纯图标：竖条列图标 + 原生 title 悬浮提示 -->
      <button
        type="button"
        class="data-column-settings"
        :class="{ 'data-column-settings--active': open }"
        aria-label="列设置"
        title="列设置"
      >
        <RiLayoutColumnFill />
      </button>
    </template>

    <div class="data-column-settings__panel" role="group" aria-label="列设置">
      <ElInput
        v-model="keyword"
        class="data-column-settings__search"
        :prefix-icon="RiSearchFill"
        placeholder="搜索（多个关键词用空格隔开）"
        clearable
      />

      <ElCheckbox
        class="data-column-settings__all"
        :model-value="allVisible"
        :indeterminate="partiallyVisible"
        :disabled="!filteredColumns.length"
        @change="onToggleAll"
      >
        全选
      </ElCheckbox>

      <ElScrollbar class="data-column-settings__list" :max-height="300">
        <!-- 字段项垂直单列排列，每行 = 类型图标 + 字段名 + 右侧勾选框 -->
        <div class="data-column-settings__grid">
          <ElCheckbox
            v-for="column in filteredColumns"
            :key="column.field"
            class="data-column-settings__item"
            :model-value="!hidden.has(column.field)"
            :disabled="disabledOf(column)"
            @change="emit('toggle', column.field)"
          >
            <component
              :is="column.icon"
              v-if="column.icon"
              class="data-column-settings__type-icon"
            />
            <!-- 字段名超宽省略号截断，悬浮经 Tooltip 展示全名 -->
            <ElTooltip :content="column.title" placement="top" :show-after="200">
              <span class="data-column-settings__label">{{ column.title }}</span>
            </ElTooltip>
          </ElCheckbox>
        </div>
        <p v-if="!filteredColumns.length" class="data-column-settings__empty">无匹配字段</p>
      </ElScrollbar>
    </div>
  </ElPopover>
</template>

<style scoped lang="scss">
.data-column-settings {
  // 纯图标按钮：方形点击区与工具栏按钮高度（34px）对齐；透明底、无边框，
  // 悬停/激活给浅主色底反馈（参考界面的工具型图标按钮形态）
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
  background: transparent;
  border: 0;
  border-radius: var(--el-border-radius-base);
  cursor: pointer;
  transition:
    color 0.18s ease,
    background-color 0.18s ease;

  svg {
    width: 17px;
    height: 17px;
  }

  &:hover,
  &--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}
</style>

<style lang="scss">
/* 传送至 body 的 popper 内容以唯一块类限定，避免影响其他弹层 */
.data-column-settings__popper.el-popover.el-popper {
  /* 浮窗宽度随最宽字段行自适应，上限 346。下限取 284 = 完整占位文案
     （260px）+ 弹窗左右内边距（24px），保证搜索占位文字完整显示；
     宽度约束集中在弹窗层，避免依赖 grid item 的 min-width 撑宽
     （浏览器轨道测量差异会使其失效导致搜索框溢出）；类名叠加提升
     优先级以覆盖 EP 默认的 min-width: 150 */
  min-width: 284px;
  max-width: 346px;
}

.data-column-settings__popper {
  .data-column-settings__panel {
    display: grid;
    gap: 4px;
  }

  // 弹窗内文案统一显式 14px，不随 EP 组件尺寸变量漂移
  .data-column-settings__search .el-input__inner {
    font-size: 14px;
  }

  .el-checkbox__label {
    font-size: 14px;
  }

  .data-column-settings__search {
    /* 宽度由弹窗层的 min-width 统一保障（见上），搜索框只做整行拉伸 */
    width: 100%;
    margin-bottom: 4px;
  }

  .data-column-settings__all {
    height: 28px;
    width: 100%;
    margin-right: 0;
    padding-bottom: 6px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  // 参考浮窗勾选态文字保持常规色（EP 默认选中后 label 变主题色）
  .el-checkbox__input.is-checked + .el-checkbox__label {
    color: var(--el-checkbox-text-color);
  }

  .data-column-settings__grid {
    display: grid;
    gap: 2px;
    padding-top: 6px;
    /* 允许链路整体收缩：item 的 min-width: auto（=nowrap 文字宽）会把
       grid 轨道顶出弹窗 max-width 边界 */
    min-width: 0;
  }

  // 字段项整行占满：图标+字段名居左，勾选框经 order 调换后贴右（参考浮窗形态）
  .data-column-settings__item {
    width: 100%;
    height: 28px;
    margin-right: 0;
    /* grid item 默认 min-width: auto 会拒绝收缩到字段名文字宽以下，
       必须显式归零，label 的省略号截断才能生效 */
    min-width: 0;
    gap: 8px;

    .el-checkbox__input {
      order: 2;
    }

    .el-checkbox__label {
      order: 1;
      display: inline-flex;
      align-items: center;
      flex: 1;
      gap: 6px;
      min-width: 0;
      padding-left: 0;
    }
  }

  .data-column-settings__type-icon {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
    color: var(--el-text-color-secondary);
  }

  .data-column-settings__label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .data-column-settings__empty {
    margin: 10px 0;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    text-align: center;
  }
}
</style>
