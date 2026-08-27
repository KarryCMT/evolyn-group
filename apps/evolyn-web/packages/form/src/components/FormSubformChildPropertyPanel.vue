<template>
  <div class="form-subform-child-property">
    <h3>{{ widgetNameLabel }}</h3>
    <label class="form-subform-child-property__label">{{ t('显示名称') }}</label>
    <el-input v-model="field.fieldLabel" />
    <label class="form-subform-child-property__label">ID</label>
    <el-input :model-value="field.fieldKey" @update:model-value="updateFieldKey(String($event))" />
    <label class="form-subform-child-property__label">{{ t('提示内容') }}</label>
    <el-input v-model="field.description" />
    <label class="form-subform-child-property__label">{{ t('校验') }}</label>
    <div class="form-subform-child-property__switches">
      <el-checkbox v-model="field.isRequired">{{ t('必填') }}</el-checkbox>
      <el-checkbox v-model="field.isHidden">{{ t('隐藏') }}</el-checkbox>
      <el-checkbox v-model="field.isEnabled">{{ t('启用') }}</el-checkbox>
    </div>

    <template v-if="field.widgetName === 'input'">
      <label class="form-subform-child-property__label">{{ t('文本类型') }}</label>
      <el-checkbox v-model="textIsMultiLine">{{ t('多行文本') }}</el-checkbox>
    </template>

    <!-- 默认值放在下拉选项之前，选择字段时优先展示当前生效值。 -->
    <label v-if="showDefaultValue" class="form-subform-child-property__label">{{
      t('默认值')
    }}</label>
    <FormDesignFieldControl
      v-if="showDefaultValue"
      :field="field"
      :model-value="field.defaultValue"
      @update:model-value="updateDefaultValue"
    />

    <template v-if="field.widgetName === 'selectGroup'">
      <label class="form-subform-child-property__label">{{ t('选项') }}</label>
      <div
        v-for="(_, index) in selectItems"
        :key="index"
        class="form-subform-child-property__option"
      >
        <el-input
          :model-value="selectItems[index]?.text"
          size="small"
          @update:model-value="updateSelectItem(index, String($event))"
        />
        <button type="button" @click="removeSelectItem(index)">
          <el-icon><Delete /></el-icon>
        </button>
      </div>
      <el-button text type="primary" @click="addSelectItem">{{ t('添加选项') }}</el-button>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ElButton, ElCheckbox, ElIcon, ElInput } from 'element-plus';
import { computed } from 'vue';
import { Delete } from '@element-plus/icons-vue';
import type { FormDesignFieldDefaultValue, FormDesignTemplateField } from '../types';
import FormDesignFieldControl from './FormDesignFieldControl.vue';

interface SelectItem {
  text: string;
  value: string;
}

/**
 * 子表单内部字段的独立属性面板。
 * @property field 当前选中的子字段，直接读写 fieldConf.fields 中的对象。
 * @event update-field-key 子字段业务标识被修改后同步外层选中状态。
 */
const props = defineProps<{
  field: FormDesignTemplateField;
}>();

const emits = defineEmits<{
  (event: 'update-field-key', value: string): void;
}>();

const t = (text: string) => text;

const widgetNameLabelMap = computed<Record<string, string>>(() => ({
  input: t('文本'),
  number: t('数字'),
  datetime: t('日期时间'),
  selectGroup: t('下拉框'),
  userGroup: t('成员选择'),
  deptGroup: t('部门选择'),
}));

const widgetNameLabel = computed(
  () => widgetNameLabelMap.value[props.field.widgetName] || props.field.fieldLabel,
);

const ensureFieldConf = () => {
  if (!props.field.fieldConf) props.field.fieldConf = {};
  return props.field.fieldConf;
};

const updateFieldKey = (fieldKey: string) => {
  // 持久化 id 由后端维护，属性面板只允许修改业务 fieldKey。
  props.field.fieldKey = fieldKey;
  emits('update-field-key', fieldKey);
};

const textIsMultiLine = computed({
  get: () => Boolean(props.field.fieldConf?.isMultiLine),
  set: (value: boolean) => {
    props.field.fieldConf = {
      ...props.field.fieldConf,
      isMultiLine: value,
    };
  },
});

const selectItems = computed<SelectItem[]>(() => {
  const items = props.field.fieldConf?.items;
  return Array.isArray(items) ? (items as SelectItem[]) : [];
});

const updateDefaultValue = (value: FormDesignFieldDefaultValue) => {
  props.field.defaultValue = value;
};

const updateSelectItem = (index: number, value: string) => {
  const previousValue = selectItems.value[index]?.value;
  selectItems.value[index] = {
    text: value,
    value,
  };
  ensureFieldConf().items = [...selectItems.value];
  // 子表单下拉框的默认项改名时同步更新默认值。
  if (props.field.defaultValue === previousValue) props.field.defaultValue = value;
};

const addSelectItem = () => {
  const value = `${t('选项')}${selectItems.value.length + 1}`;
  ensureFieldConf().items = [
    ...selectItems.value,
    {
      text: value,
      value,
    },
  ];
};

const removeSelectItem = (index: number) => {
  const nextItems = [...selectItems.value];
  const [removedItem] = nextItems.splice(index, 1);
  ensureFieldConf().items = nextItems;
  // 默认项被删除后清空默认值，避免保存无效选项。
  if (props.field.defaultValue === removedItem?.value) props.field.defaultValue = '';
};

const showDefaultValue = computed(() => Boolean(widgetNameLabelMap.value[props.field.widgetName]));
</script>

<style lang="scss">
.form-subform-child-property {
  h3 {
    margin: 0 0 var(--el-space-lg);
    font-size: var(--el-font-size-base);
    color: var(--el-text-color-primary);
  }

  &__label {
    display: block;
    margin: var(--el-space-lg) 0 var(--el-space-xs);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__switches,
  &__option {
    display: flex;
    align-items: center;
  }

  &__switches {
    flex-wrap: wrap;
    gap: var(--el-space-sm);
  }

  &__option {
    gap: var(--el-space-xs);
    margin-bottom: var(--el-space-sm);

    button {
      width: var(--el-space-3xl);
      color: var(--el-text-color-secondary);
      cursor: pointer;
      background: transparent;
      border: 0;
    }
  }

  .el-select,
  .el-date-editor,
  .el-input-number {
    width: 100%;
  }
}
</style>
