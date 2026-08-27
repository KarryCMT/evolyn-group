<template>
  <aside class="form-design-property">
    <template v-if="field">
      <FormSubformChildPropertyPanel
        v-if="selectedSubformChildField"
        :field="selectedSubformChildField"
        @update-field-key="selectedChildFieldKey = $event"
      />
      <template v-else>
        <h3>{{ widgetNameLabel }}</h3>
        <label class="form-design-property__label">{{ t('显示名称') }}</label>
        <el-input v-model="field.fieldLabel" />
        <label class="form-design-property__label form-design-property__label--help">
          <span>ID</span>
          <!-- ID 规则通过悬停提示展示，避免说明文字长期占用属性面板空间。 -->
          <el-tooltip effect="light" placement="top">
            <template #content>
              <div class="form-design-property__id-help">
                {{
                  t(
                    '定义字段在代码中被调用时的名称，不能与本触发动作内其他字段ID重复，仅支持「字母」「数字」和「下划线」。',
                  )
                }}
              </div>
            </template>
            <el-icon class="form-design-property__help-icon" tabindex="0" aria-label="ID">
              <RiQuestionFill />
            </el-icon>
          </el-tooltip>
        </label>
        <el-input
          :model-value="field.fieldKey"
          @update:model-value="$emit('update-field-key', String($event))"
        />
      </template>
      <template v-if="!isSubformField(field) && !selectedSubformChildField">
        <label class="form-design-property__label">{{ t('提示内容') }}</label>
        <el-input v-model="field.placeholder" />
        <label class="form-design-property__label">{{ t('校验') }}</label>
        <el-checkbox v-model="field.isRequired">{{ t('必填') }}</el-checkbox>
        <label class="form-design-property__label">{{ t('默认值') }}</label>
        <FormDesignFieldControl
          :field="field"
          :model-value="field.defaultValue"
          @update:model-value="$emit('update-default-value', $event)"
        />
      </template>
      <FormSubformFieldPanel
        v-else-if="!selectedSubformChildField"
        v-model:selected-child-field-key="selectedChildFieldKey"
        :field="field"
      />
      <template v-if="field.widgetName === 'selectGroup' && !selectedSubformChildField">
        <!-- 下拉选项标题与新增入口保持同一行，便于在编辑列表前直接添加。 -->
        <div class="form-design-property__option-header">
          <label class="form-design-property__label">{{ t('选项') }}</label>
          <el-button text size="small" @click="$emit('add-option')">{{ t('添加选项') }}</el-button>
        </div>
        <div
          v-for="(_, index) in field.options || []"
          :key="index"
          class="form-design-property__option"
        >
          <el-input
            :model-value="field.options?.[index]"
            size="small"
            @update:model-value="$emit('update-option', index, String($event))"
          />
          <button type="button" @click="$emit('remove-option', index)">
            <el-icon><Delete /></el-icon>
          </button>
        </div>
      </template>
    </template>
    <div v-else class="form-design-property__empty">
      <el-empty :description="t('请选择字段')" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ElButton, ElCheckbox, ElEmpty, ElIcon, ElInput, ElTooltip } from 'element-plus';
import { computed } from 'vue';
import { Delete } from '@element-plus/icons-vue';
import { RiQuestionFill } from '@remixicon/vue';
import type {
  FormDesignField,
  FormDesignFieldDefaultValue,
  FormDesignTemplateField,
} from '../types';
import FormDesignFieldControl from './FormDesignFieldControl.vue';
import FormSubformChildPropertyPanel from './FormSubformChildPropertyPanel.vue';
import FormSubformFieldPanel from './FormSubformFieldPanel.vue';

const props = defineProps<{
  field?: FormDesignField;
  widgetNameLabel: string;
  selectedSubformChildFieldKey?: string;
}>();

const emits = defineEmits<{
  (event: 'add-option'): void;
  (event: 'remove-option', index: number): void;
  (event: 'update-default-value', value: FormDesignFieldDefaultValue): void;
  (event: 'update-field-key', value: string): void;
  (event: 'update-option', index: number, value: string): void;
  (event: 'update-subform-child-selection', value: string): void;
}>();

const t = (text: string) => text;

const isSubformField = (field: FormDesignField) => field.widgetName === 'subforms';

const subformChildFields = computed(() => {
  const fields = props.field?.fieldConf?.fields;
  return Array.isArray(fields) ? (fields as FormDesignTemplateField[]) : [];
});

const selectedSubformChildField = computed(() => {
  if (!props.field || !isSubformField(props.field)) return undefined;
  return subformChildFields.value.find((item) => item.fieldKey === selectedChildFieldKey.value);
});

const selectedChildFieldKey = computed({
  get: () => props.selectedSubformChildFieldKey || '',
  set: (value: string) => {
    emits('update-subform-child-selection', value);
  },
});
</script>

<style lang="scss">
.form-design-property {
  box-sizing: border-box;
  flex-shrink: 0;
  width: 280px;
  padding: var(--el-space-3xl);
  overflow: auto;
  background-color: var(--el-fill-color-light);
  // border-top: 1px solid var(--el-border-color-lighter);
  // border-right: 1px solid var(--el-border-color-lighter);
  // border-bottom: 1px solid var(--el-border-color-lighter);
  // border-left: 1px solid var(--el-border-color-lighter);

  h3 {
    margin: 0 0 var(--gp-space-2xl);
    font-size: var(--el-font-size-medium);
    color: var(--el-text-color-primary);
  }

  &__label {
    display: block;
    margin: var(--gp-space-xl) 0 var(--gp-space-sm);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    color: var(--el-text-color-primary);

    &--help {
      display: inline-flex;
      gap: var(--gp-space-xs);
      align-items: center;
    }
  }

  &__help-icon {
    font-size: var(--el-font-size-base);
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  &__id-help {
    width: 220px;
    font-size: var(--el-font-size-extra-small);
    line-height: 1.5;
    color: var(--el-text-color-primary);
  }

  &__empty {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    min-height: 320px;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }

  &__option {
    display: flex;
    gap: var(--gp-space-xs);
    margin-bottom: var(--gp-space-sm);

    button {
      width: 28px;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      background: transparent;
      border: 0;
    }
  }

  &__option-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: var(--gp-space-xl) 0 var(--gp-space-sm);

    .form-design-property__label {
      margin: 0;
    }
  }
}
</style>
