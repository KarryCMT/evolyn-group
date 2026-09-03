<script setup lang="ts">
import {
  RiCalendarLine,
  RiCheckboxMultipleLine,
  RiFileTextLine,
  RiHashtag,
  RiListCheck3,
  RiMapPinLine,
  RiPhoneLine,
  RiRadioButtonLine,
  RiSearchLine,
  RiText,
  RiUserLine,
  RiUserSharedLine,
} from '@remixicon/vue';
import { computed, shallowRef, type Component } from 'vue';
import { ElIcon, ElInput } from 'element-plus';
import { isSubmitRuleEligibleType } from '../schema/invisible-value-policy';
import type { FormItem, FormWidgetType, SubmitRule } from '../schema/types';

/**
 * 特殊赋值规则的字段选择浮层：只负责搜索、展示和回传稳定字段键，策略归属
 * 仍由父弹窗维护，确保一次点击可把字段从其他规则组原子地移动到当前组。
 */
const props = defineProps<{
  items: FormItem[];
  strategy: SubmitRule;
  rules: Readonly<Record<string, SubmitRule>>;
}>();

const emit = defineEmits<{ select: [widgetName: string] }>();

const keyword = shallowRef('');

const iconByType: Partial<Record<FormWidgetType, Component>> = {
  text: RiText,
  textarea: RiFileTextLine,
  number: RiHashtag,
  datetime: RiCalendarLine,
  radiogroup: RiRadioButtonLine,
  checkboxgroup: RiCheckboxMultipleLine,
  combo: RiListCheck3,
  combocheck: RiListCheck3,
  user: RiUserLine,
  usergroup: RiUserSharedLine,
  dept: RiUserSharedLine,
  deptgroup: RiUserSharedLine,
  phone: RiPhoneLine,
  address: RiMapPinLine,
  location: RiMapPinLine,
};

const fields = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  return props.items.filter((item) => {
    if (!isSubmitRuleEligibleType(item.widget.type)) return false;
    if (!normalized) return true;
    return (
      item.label.toLocaleLowerCase().includes(normalized) ||
      item.widget.widgetName.toLocaleLowerCase().includes(normalized)
    );
  });
});

function fieldIcon(type: FormWidgetType): Component {
  return iconByType[type] ?? RiText;
}

function assignedToCurrentGroup(widgetName: string): boolean {
  return props.rules[widgetName] === props.strategy;
}

function assignedToAnotherGroup(widgetName: string): boolean {
  const assigned = props.rules[widgetName];
  return assigned !== undefined && assigned !== props.strategy;
}
</script>

<template>
  <div class="form-submit-rule-field-picker" aria-label="添加字段">
    <el-input
      v-model="keyword"
      class="form-submit-rule-field-picker__search"
      autofocus
      placeholder="搜索"
      aria-label="搜索字段"
    >
      <template #prefix>
        <el-icon class="form-submit-rule-field-picker__search-icon"><RiSearchLine /></el-icon>
      </template>
    </el-input>

    <div class="form-submit-rule-field-picker__list" role="listbox" aria-label="可选字段">
      <button
        v-for="item in fields"
        :key="item.widget.widgetName"
        type="button"
        class="form-submit-rule-field-picker__item"
        :class="{
          'form-submit-rule-field-picker__item--current': assignedToCurrentGroup(
            item.widget.widgetName,
          ),
          'form-submit-rule-field-picker__item--assigned': assignedToAnotherGroup(
            item.widget.widgetName,
          ),
        }"
        role="option"
        :aria-selected="assignedToCurrentGroup(item.widget.widgetName)"
        @click="emit('select', item.widget.widgetName)"
      >
        <el-icon class="form-submit-rule-field-picker__item-icon">
          <component :is="fieldIcon(item.widget.type)" />
        </el-icon>
        <span class="form-submit-rule-field-picker__item-label">{{ item.label }}</span>
      </button>
      <p v-if="fields.length === 0" class="form-submit-rule-field-picker__empty">未找到匹配字段</p>
    </div>
  </div>
</template>

<style lang="scss">
.form-submit-rule-field-picker {
  width: 480px;
  overflow: hidden;
  background: var(--el-bg-color-overlay);
  border-radius: 14px;

  &__search {
    height: 44px;
    padding: 0 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    .el-input__wrapper {
      padding: 0;
      background: transparent;
      box-shadow: none !important;
    }

    .el-input__inner {
      height: 44px;
      font-size: 16px;
      color: var(--el-text-color-primary);
    }

    .el-input__inner::placeholder {
      color: var(--el-text-color-placeholder);
    }
  }

  &__search-icon {
    margin-right: 8px;
    font-size: 20px;
    color: var(--el-text-color-regular);
  }

  &__list {
    max-height: 440px;
    padding: 8px;
    overflow-y: auto;
  }

  &__item {
    display: flex;
    width: 100%;
    min-height: 44px;
    gap: 12px;
    align-items: center;
    padding: 0 10px;
    color: var(--el-text-color-primary);
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 10px;

    &:hover,
    &--current {
      background: var(--el-fill-color-light);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--assigned:not(&--current) {
      color: var(--el-text-color-disabled);
    }
  }

  &__item-icon {
    flex: 0 0 auto;
    font-size: 20px;
    color: currentcolor;
  }

  &__item-label {
    overflow: hidden;
    font-size: 16px;
    line-height: 1.5;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__empty {
    margin: 18px 14px;
    font-size: 14px;
    color: var(--el-text-color-secondary);
  }
}
</style>
