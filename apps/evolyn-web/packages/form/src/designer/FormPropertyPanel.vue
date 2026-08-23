<script setup lang="ts">
import { shallowRef } from 'vue';

type InspectorTab = 'field' | 'form';

const inspectorTab = shallowRef<InspectorTab>('field');
</script>

<template>
  <aside class="form-property-panel" aria-label="属性设置">
    <div class="form-property-panel__tabs" role="tablist" aria-label="属性类型">
      <button
        class="form-property-panel__tab"
        :class="{ 'form-property-panel__tab--active': inspectorTab === 'field' }"
        type="button"
        role="tab"
        :aria-selected="inspectorTab === 'field'"
        @click="inspectorTab = 'field'"
      >
        字段属性
      </button>
      <button
        class="form-property-panel__tab"
        :class="{ 'form-property-panel__tab--active': inspectorTab === 'form' }"
        type="button"
        role="tab"
        :aria-selected="inspectorTab === 'form'"
        @click="inspectorTab = 'form'"
      >
        表单属性
      </button>
    </div>
    <div class="form-property-panel__empty" role="tabpanel">
      <template v-if="inspectorTab === 'field'">
        <p class="form-property-panel__empty-copy">点击选择字段来设置属性</p>
        <p class="form-property-panel__empty-copy">按住 Ctrl 或 Command 可选择多个字段</p>
        <p class="form-property-panel__empty-copy">按住 Shift 单击字段可连续选择</p>
      </template>
      <template v-else>
        <p class="form-property-panel__empty-copy">表单属性将在设计器接入后提供</p>
      </template>
    </div>
  </aside>
</template>

<style scoped lang="scss">
.form-property-panel {
  display: flex;
  box-sizing: border-box;
  width: 300px;
  min-width: 300px;
  max-width: 300px;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-lighter);

  &__tabs {
    display: flex;
    min-height: 60px;
    align-items: center;
    justify-content: space-around;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__tab {
    position: relative;
    height: 60px;
    padding: 0 18px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    font-size: 16px;
    font-weight: 600;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &::after {
      position: absolute;
      right: 18px;
      bottom: 0;
      left: 18px;
      height: 2px;
      content: '';
      background: transparent;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &--active {
      color: var(--el-color-primary);

      &::after {
        background: var(--el-color-primary);
      }
    }
  }

  &__empty {
    display: flex;
    min-height: 0;
    padding: 48px 36px;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
  }

  &__empty-copy {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 24px;
  }
}
</style>
