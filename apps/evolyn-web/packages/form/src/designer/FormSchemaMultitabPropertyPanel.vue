<script setup lang="ts">
import { RiAddFill, RiDeleteBin6Fill, RiDragMoveFill, RiFileCopyFill } from '@remixicon/vue';
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElRadioButton,
  ElRadioGroup,
} from 'element-plus';
import Draggable from 'vuedraggable';
import type { FormMultitabLayout, FormTabStyle } from '../schema/types';

/** 标签页属性编辑区：仅展示配置并上抛语义事件，协议变更仍由编辑器状态层统一执行。 */
defineProps<{ layout: FormMultitabLayout }>();

const emit = defineEmits<{
  setTabStyle: [style: FormTabStyle];
  addTab: [];
  removeTab: [tabName: string];
  duplicateTab: [tabName: string];
  renameTab: [tabName: string, title: string];
  reorderTabs: [tabNames: string[]];
}>();

function reorder(entries: FormMultitabLayout['container']): void {
  emit(
    'reorderTabs',
    entries.map((tab) => tab.name),
  );
}
</script>

<template>
  <el-form class="form-schema-multitab-property" label-position="top" @submit.prevent>
    <el-form-item label="样式">
      <el-radio-group
        :model-value="layout.tabStyle"
        class="form-schema-multitab-property__styles"
        @update:model-value="emit('setTabStyle', $event as FormTabStyle)"
      >
        <el-radio-button value="style1">线条</el-radio-button>
        <el-radio-button value="style2">卡片</el-radio-button>
      </el-radio-group>
    </el-form-item>

    <el-form-item label="多标签显示">
      <div class="form-schema-multitab-property__tabs">
        <Draggable
          :model-value="layout.container"
          item-key="name"
          handle=".form-schema-multitab-property__drag"
          class="form-schema-multitab-property__tab-list"
          ghost-class="form-schema-multitab-property__tab--ghost"
          :animation="180"
          @update:model-value="reorder"
        >
          <template #item="{ element: tab }">
            <div class="form-schema-multitab-property__tab">
              <span class="form-schema-multitab-property__drag" title="拖拽排序">
                <el-icon><RiDragMoveFill /></el-icon>
              </span>
              <el-input
                :model-value="tab.title"
                :maxlength="64"
                aria-label="标签页标题"
                @update:model-value="emit('renameTab', tab.name, String($event ?? ''))"
              />
              <el-button
                text
                title="复制标签页"
                :icon="RiFileCopyFill"
                :disabled="layout.container.length >= 20"
                @click="emit('duplicateTab', tab.name)"
              />
              <el-button
                text
                type="danger"
                title="删除标签页"
                :icon="RiDeleteBin6Fill"
                :disabled="layout.container.length <= 1"
                @click="emit('removeTab', tab.name)"
              />
            </div>
          </template>
        </Draggable>

        <el-button
          class="form-schema-multitab-property__add"
          :icon="RiAddFill"
          :disabled="layout.container.length >= 20"
          @click="emit('addTab')"
        >
          添加标签页
        </el-button>
      </div>
    </el-form-item>
  </el-form>
</template>

<style scoped lang="scss">
.form-schema-multitab-property {
  &__styles {
    width: 100%;

    :deep(.el-radio-button) {
      flex: 1;
    }

    :deep(.el-radio-button__inner) {
      width: 100%;
    }
  }

  &__tabs,
  &__tab-list {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
    width: 100%;
  }

  &__tab {
    display: flex;
    gap: var(--el-space-xs);
    align-items: center;
    min-height: 40px;
    padding: var(--el-space-xs);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
  }

  &__tab--ghost {
    border-color: var(--el-color-primary);
    opacity: 0.5;
  }

  &__drag {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    padding: var(--el-space-xs);
    color: var(--el-text-color-secondary);
    cursor: grab;
  }

  &__add {
    width: 100%;
  }
}
</style>
