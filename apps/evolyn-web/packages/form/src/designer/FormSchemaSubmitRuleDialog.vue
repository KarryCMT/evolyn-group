<script setup lang="ts">
import { RiAddLine, RiCloseLine, RiQuestionFill } from '@remixicon/vue';
import { computed, reactive, shallowRef, watch } from 'vue';
import { ElButton, ElDialog, ElIcon, ElPopover, ElTooltip } from 'element-plus';
import { isSubmitRuleEligibleType, submitRuleLabel } from '../schema/invisible-value-policy';
import type { FormItem, SubmitRule } from '../schema/types';
import FormSchemaSubmitRuleFieldPicker from './FormSchemaSubmitRuleFieldPicker.vue';

/** 特殊赋值规则编辑弹窗：props 为已保存值，draft 为关闭前唯一可变副本。 */
const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    submitRule: SubmitRule;
    widgetSubmitRules: Record<string, SubmitRule>;
    items: FormItem[];
  }>(),
  {
    modelValue: false,
    submitRule: 2,
    widgetSubmitRules: () => ({}),
    items: () => [],
  },
);

const emit = defineEmits<{
  'update:model-value': [open: boolean];
  save: [rules: Record<string, SubmitRule>];
}>();

interface RuleGroup {
  value: SubmitRule;
  label: string;
  hint: string;
}

interface SelectedField {
  name: string;
  label: string;
}

/**
 * 与表单默认值相同的策略无需特殊配置，故只显示另外两段。这正是参考界面中
 * 默认“保持原值”时仅出现“空值 / 始终重新计算”的原因。
 */
const allGroups = computed<RuleGroup[]>(() =>
  ([1, 2, 3] as const).map((value) => ({
    value,
    label: submitRuleLabel(value),
    hint:
      value === 1
        ? '新建时无原值，将按空值写入'
        : value === 2
          ? '不可见时清除原有值'
          : '不可见时沿用可见字段的计算、提交逻辑',
  })),
);

const groups = computed(() => allGroups.value.filter((group) => group.value !== props.submitRule));

/** 所有交互先落本地草稿；取消或遮罩关闭绝不改变父级表单 Schema。 */
const draft = reactive<Record<string, SubmitRule>>({});
const pickerOpen = reactive<Partial<Record<SubmitRule, boolean>>>({});
const activePickerStrategy = shallowRef<SubmitRule | null>(null);

const fields = computed<SelectedField[]>(() =>
  props.items
    .filter((item) => isSubmitRuleEligibleType(item.widget.type))
    .map((item) => ({ name: item.widget.widgetName, label: item.label })),
);

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetDraft();
  },
  { immediate: true },
);

function resetDraft(): void {
  Object.keys(draft).forEach((key) => delete draft[key]);
  Object.assign(draft, props.widgetSubmitRules);
  for (const strategy of [1, 2, 3] as const) delete pickerOpen[strategy];
  activePickerStrategy.value = null;
}

function selectedOf(strategy: SubmitRule): SelectedField[] {
  return fields.value.filter((field) => draft[field.name] === strategy);
}

function openPicker(strategy: SubmitRule): void {
  activePickerStrategy.value = strategy;
  pickerOpen[strategy] = true;
}

function handlePickerVisibility(strategy: SubmitRule, visible: boolean): void {
  pickerOpen[strategy] = visible;
  if (!visible && activePickerStrategy.value === strategy) activePickerStrategy.value = null;
}

function assign(name: string, strategy: SubmitRule): void {
  // 选入另一段即移动；映射结构确保任一字段不可能重复归属。
  draft[name] = strategy;
}

function unassign(name: string): void {
  delete draft[name];
}

function close(): void {
  emit('update:model-value', false);
}

function confirm(): void {
  const normalized: Record<string, SubmitRule> = {};
  for (const [name, strategy] of Object.entries(draft)) {
    if (strategy !== props.submitRule) normalized[name] = strategy;
  }
  emit('save', normalized);
  close();
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    append-to-body
    destroy-on-close
    :show-close="false"
    class="form-submit-rule-dialog"
    aria-label="特殊字段赋值规则"
    @update:model-value="emit('update:model-value', $event)"
    @closed="resetDraft"
  >
    <template #header>
      <div class="form-submit-rule-dialog__header">
        <h2 class="form-submit-rule-dialog__title">特殊字段赋值规则</h2>
        <button
          type="button"
          class="form-submit-rule-dialog__close"
          aria-label="关闭特殊字段赋值规则"
          @click="close"
        >
          <el-icon><RiCloseLine /></el-icon>
        </button>
      </div>
    </template>

    <div class="form-submit-rule-dialog__body">
      <p class="form-submit-rule-dialog__intro">
        此处设置的字段，不可见时按照下方对应规则赋值，不受默认赋值规则的影响
      </p>

      <section
        v-for="group in groups"
        :key="group.value"
        class="form-submit-rule-dialog__section"
        :aria-label="`${group.label}特殊字段`"
      >
        <header class="form-submit-rule-dialog__section-header">
          <div class="form-submit-rule-dialog__section-heading">
            <h3 class="form-submit-rule-dialog__section-title">{{ group.label }}</h3>
            <el-tooltip :content="group.hint" placement="top">
              <el-icon class="form-submit-rule-dialog__help" :aria-label="`${group.label}说明`">
                <RiQuestionFill />
              </el-icon>
            </el-tooltip>
          </div>

          <el-popover
            :visible="pickerOpen[group.value] === true"
            placement="bottom-end"
            :width="480"
            :teleported="true"
            :show-arrow="false"
            popper-class="form-submit-rule-field-picker-popper"
            @update:visible="handlePickerVisibility(group.value, $event)"
          >
            <template #reference>
              <button
                type="button"
                class="form-submit-rule-dialog__add"
                @click="openPicker(group.value)"
              >
                <el-icon><RiAddLine /></el-icon>
                添加字段
              </button>
            </template>
            <FormSchemaSubmitRuleFieldPicker
              v-if="activePickerStrategy === group.value"
              :items="items"
              :rules="draft"
              :strategy="group.value"
              @select="assign($event, group.value)"
            />
          </el-popover>
        </header>

        <div
          class="form-submit-rule-dialog__tags"
          :class="{ 'is-empty': selectedOf(group.value).length === 0 }"
        >
          <button
            v-for="field in selectedOf(group.value)"
            :key="field.name"
            type="button"
            class="form-submit-rule-dialog__tag"
            :aria-label="`移除 ${field.label}`"
            @click="unassign(field.name)"
          >
            <span>{{ field.label }}</span>
            <el-icon><RiCloseLine /></el-icon>
          </button>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="form-submit-rule-dialog__footer">
        <a
          class="form-submit-rule-dialog__help-link"
          href="/docs/form/invisible-field-value"
          target="_blank"
          rel="noopener noreferrer"
        >
          查看帮助文档
        </a>
        <div class="form-submit-rule-dialog__actions">
          <el-button class="form-submit-rule-dialog__cancel" @click="close">取消</el-button>
          <el-button class="form-submit-rule-dialog__confirm" type="primary" @click="confirm">
            确定
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style lang="scss">
.form-submit-rule-dialog {
  // 特殊规则只管理字段标签，不需要占用全屏编辑空间；使用设计器常规设置弹窗宽度，
  // 字段较多时仅滚动内容区，页头和操作按钮始终保持可见。
  width: min(640px, calc(100vw - 40px)) !important;
  display: flex;
  max-height: calc(100dvh - 40px);
  margin: 20px auto !important;
  overflow: hidden;
  background: var(--el-bg-color);
  border-radius: 12px;
  box-shadow: var(--el-box-shadow-light);
  flex-direction: column;

  .el-dialog__header {
    padding: 0;
    margin: 0;
    border-bottom: 1px solid var(--el-border-color);
  }

  .el-dialog__body {
    display: flex;
    min-height: 0;
    padding: 0;
    flex: 0 1 auto;
  }

  .el-dialog__footer {
    padding: 0;
    border-top: 1px solid var(--el-border-color);
  }

  &__header {
    display: flex;
    height: 56px;
    padding: 0 24px;
    align-items: center;
    justify-content: space-between;
  }

  &__title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    line-height: 1;
    color: var(--el-text-color-primary);
    letter-spacing: 0;
  }

  &__close {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 8px;

    &:hover {
      background: var(--el-fill-color-light);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__body {
    max-height: min(360px, calc(100dvh - 160px));
    padding: 20px 24px;
    overflow-y: auto;
  }

  &__intro {
    margin: 0 0 16px;
    font-size: 14px;
    line-height: 1.5;
    color: var(--el-text-color-regular);
  }

  &__section {
    padding: 0 0 16px;

    & + & {
      padding-top: 20px;
      border-top: 1px solid var(--el-border-color-lighter);
    }
  }

  &__section-header,
  &__section-heading,
  &__footer,
  &__actions,
  &__tag,
  &__add {
    display: flex;
    align-items: center;
  }

  &__section-header,
  &__footer {
    justify-content: space-between;
  }

  &__section-heading {
    gap: 6px;
  }

  &__section-title {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    line-height: 1.25;
    color: var(--el-text-color-primary);
  }

  &__help {
    font-size: 14px;
    color: var(--el-text-color-secondary);
  }

  &__add {
    gap: 4px;
    padding: 2px 0;
    font-size: 14px;
    line-height: 1.5;
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    border: 0;

    .el-icon {
      font-size: 16px;
    }
    &:hover {
      color: var(--el-color-primary-light-3);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 4px;
    }
  }

  &__tags {
    display: flex;
    min-height: 32px;
    gap: 6px;
    flex-wrap: wrap;
    align-content: flex-start;
    padding-top: 8px;

    &.is-empty {
      min-height: 32px;
    }
  }

  &__tag {
    gap: 6px;
    height: 32px;
    padding: 0 8px;
    font-size: 14px;
    line-height: 1;
    color: var(--el-text-color-primary);
    cursor: pointer;
    background: var(--el-fill-color-light);
    border: 0;
    border-radius: 6px;

    .el-icon {
      font-size: 16px;
      color: var(--el-text-color-regular);
    }
    &:hover {
      background: var(--el-fill-color);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__footer {
    height: 64px;
    padding: 0 24px;
  }

  &__help-link {
    font-size: 14px;
    color: var(--el-color-primary);
    text-decoration: underline;
    text-underline-offset: 4px;

    &:hover {
      color: var(--el-color-primary-light-3);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 4px;
    }
  }

  &__actions {
    gap: 8px;
  }

  &__cancel,
  &__confirm {
    width: 72px;
    height: 32px;
    margin: 0 !important;
    font-size: 14px;
    border-radius: 8px;
  }

  &__cancel {
    color: var(--el-text-color-primary);
    background: var(--el-bg-color);
    border-color: var(--el-border-color);
  }

  &__confirm {
    --el-button-bg-color: var(--el-color-primary);
    --el-button-border-color: var(--el-color-primary);
    --el-button-hover-bg-color: var(--el-color-primary-light-3);
    --el-button-hover-border-color: var(--el-color-primary-light-3);
  }
}

.form-submit-rule-field-picker-popper.el-popper {
  padding: 0 !important;
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-overlay);
  border: 0 !important;
  border-radius: 14px;
  box-shadow: 0 16px 36px rgb(15 23 42 / 18%);
}

@media (width <= 760px) {
  .form-submit-rule-dialog {
    width: calc(100vw - 24px) !important;
    min-height: 0;
    margin-top: 12px !important;

    &__header {
      height: 52px;
      padding: 0 16px;
    }
    &__title {
      font-size: 16px;
    }
    &__body {
      max-height: calc(100dvh - 136px);
      padding: 16px;
    }
    &__intro {
      font-size: 14px;
    }
    &__section-title {
      font-size: 16px;
    }
    &__add {
      font-size: 14px;
    }
    &__tag {
      height: 34px;
      font-size: 14px;
    }
    &__footer {
      height: 56px;
      padding: 0 16px;
    }
    &__help-link {
      font-size: 14px;
    }
    &__cancel,
    &__confirm {
      width: 68px;
      height: 32px;
      font-size: 14px;
    }
    &__actions {
      gap: 10px;
    }
  }
}
</style>
