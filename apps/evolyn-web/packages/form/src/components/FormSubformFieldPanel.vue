<template>
  <div class="form-subform-field-panel">
    <div class="form-subform-field-panel__header">
      <span>{{ '子字段' }}</span>
      <el-popover
        v-model:visible="addPopoverVisible"
        placement="bottom-start"
        width="220"
        trigger="click"
      >
        <template #reference>
          <el-button text type="primary">
            <el-icon><Plus /></el-icon>
            {{ '添加子字段' }}
          </el-button>
        </template>
        <div class="form-subform-field-panel__palette">
          <button
            v-for="item in fieldPalette"
            :key="item.widgetName"
            type="button"
            @click="addChildField(item)"
          >
            <el-icon><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </button>
        </div>
      </el-popover>
    </div>

    <Draggable
      :list="childFields"
      item-key="fieldKey"
      handle=".form-subform-field-panel__drag"
      ghost-class="form-subform-field-panel__item--ghost"
      chosen-class="form-subform-field-panel__item--chosen"
      class="form-subform-field-panel__list"
      :animation="180"
      @change="ensureSelectedField"
    >
      <template #item="{ element: childField }">
        <div
          class="form-subform-field-panel__item"
          :class="{ 'is-active': selectedChildFieldKey === childField.fieldKey }"
          @click="selectedChildFieldKey = childField.fieldKey"
        >
          <el-icon class="form-subform-field-panel__drag"><Rank /></el-icon>
          <el-icon><component :is="getFieldIcon(childField.widgetName)" /></el-icon>
          <span class="form-subform-field-panel__item-label">{{ childField.fieldLabel }}</span>
          <button type="button" :title="'复制'" @click.stop="copyChildField(childField)">
            <el-icon><CopyDocument /></el-icon>
          </button>
          <el-popconfirm
            placement="bottom-end"
            :width="220"
            hide-icon
            confirm-button-type="danger"
            :title="'确定删除此项？删除后不可恢复，请确保你的工作不受影响'"
            :cancel-button-text="'取消'"
            :confirm-button-text="'删除'"
            @confirm="removeChildField(childField.fieldKey)"
          >
            <template #reference>
              <!-- 子字段列表中的删除入口同样需要确认，避免绕过画布确认直接删除。 -->
              <button type="button" :title="'删除'" @click.stop>
                <el-icon><Delete /></el-icon>
              </button>
            </template>
          </el-popconfirm>
        </div>
      </template>
    </Draggable>
    <div v-if="!childFields.length" class="form-subform-field-panel__empty">
      {{ '添加子字段' }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElButton, ElIcon, ElPopconfirm, ElPopover } from 'element-plus';
import { computed, ref } from 'vue';
import Draggable from 'vuedraggable';
import {
  Calendar,
  Connection,
  CopyDocument,
  Delete,
  Document,
  Grid,
  Plus,
  Rank,
  User,
} from '@element-plus/icons-vue';
import type { FormDesignField, FormDesignFieldSource, FormDesignTemplateField } from '../types';
import { cloneFormDesign, useFormDesignFactory } from '../hooks/useFormDesignFactory';

/**
 * 新子表单右侧子字段配置组件。
 * @property field 当前选中的子表单父字段，子字段统一读写 field.fieldConf.fields。
 * @property selectedChildFieldKey 当前选中的子字段业务标识，用于和画布内部选中态同步。
 */
const props = defineProps<{
  field: FormDesignField;
  selectedChildFieldKey?: string;
}>();

const emits = defineEmits<{
  (event: 'update:selectedChildFieldKey', value: string): void;
}>();

const addPopoverVisible = ref(false);
const { createSubformChildField } = useFormDesignFactory();

const fieldIconMap = {
  input: Document,
  number: Grid,
  datetime: Calendar,
  selectGroup: Plus,
  userGroup: User,
  deptGroup: Connection,
};

const fieldPalette = computed(() => [
  {
    label: '文本',
    widgetName: 'input',
    dataType: 'String',
    icon: Document,
  },
  {
    label: '数字',
    widgetName: 'number',
    dataType: 'Number',
    icon: Grid,
  },
  {
    label: '日期时间',
    widgetName: 'datetime',
    dataType: 'Date',
    icon: Calendar,
  },
  {
    label: '下拉框',
    widgetName: 'selectGroup',
    dataType: 'Object',
    icon: Plus,
  },
  {
    label: '成员选择',
    widgetName: 'userGroup',
    dataType: 'Object',
    icon: User,
  },
  {
    label: '部门选择',
    widgetName: 'deptGroup',
    dataType: 'Object',
    icon: Connection,
  },
]);

let childFieldSeed = 0;

const ensureSubformConf = () => {
  if (!props.field.fieldConf) props.field.fieldConf = {};
  if (!Array.isArray(props.field.fieldConf.fields)) props.field.fieldConf.fields = [];
  return props.field.fieldConf.fields as FormDesignTemplateField[];
};

const childFields = computed(() => ensureSubformConf());

const selectedChildFieldKey = computed({
  get: () => props.selectedChildFieldKey || '',
  set: (value: string) => {
    emits('update:selectedChildFieldKey', value);
  },
});

const getFieldIcon = (widgetName: string) => {
  return fieldIconMap[widgetName as keyof typeof fieldIconMap] || Document;
};

const createFieldKey = () => `_widget_${Date.now()}${childFieldSeed++}`;

const addChildField = (source: FormDesignFieldSource & { label: string }) => {
  const field = createSubformChildField(source.widgetName, source.dataType, source.label);
  if (!field) return;
  childFields.value.push(field);
  addPopoverVisible.value = false;
};

const copyChildField = (field: FormDesignTemplateField) => {
  const fieldKey = createFieldKey();
  const nextField = {
    ...cloneFormDesign(field),
    // 复制字段按新增数据处理，持久化 id 等待后端返回。
    id: null,
    fieldKey,
    fieldLabel: `${field.fieldLabel} copy`,
  };
  const index = childFields.value.findIndex((item) => item.fieldKey === field.fieldKey);
  childFields.value.splice(index + 1, 0, nextField);
};

const removeChildField = (fieldKey: string) => {
  const index = childFields.value.findIndex((item) => item.fieldKey === fieldKey);
  if (index === -1) return;
  childFields.value.splice(index, 1);
  if (selectedChildFieldKey.value === fieldKey) {
    selectedChildFieldKey.value = '';
  }
};

const ensureSelectedField = () => {
  const hasSelectedField = childFields.value.some(
    (item) => item.fieldKey === selectedChildFieldKey.value,
  );
  if (selectedChildFieldKey.value && !hasSelectedField) selectedChildFieldKey.value = '';
};
</script>

<style lang="scss">
.form-subform-field-panel {
  margin-top: var(--el-space-xl);

  &__header,
  &__item {
    display: flex;
    align-items: center;
  }

  &__header {
    justify-content: space-between;
    font-size: var(--el-font-size-extra-smallze-extra-small);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__palette {
    display: flex;
    flex-direction: column;
    gap: var(--gp-space-xs);

    button {
      display: flex;
      gap: var(--gp-space-sm);
      align-items: center;
      padding: var(--gp-space-sm) var(--gp-space-md);
      color: var(--el-text-color-primary);
      cursor: pointer;
      background-color: transparent;
      border: 0;
      border-radius: var(--el-border-radius-base);

      &:hover {
        background-color: var(--el-fill-color-light);
      }
    }
  }

  &__list {
    min-height: var(--gp-space-xl);
    margin-top: var(--gp-space-md);
  }

  &__item {
    gap: var(--gp-space-sm);
    min-height: var(--gp-space-4xl);
    padding: 0 var(--gp-space-sm);
    margin-bottom: var(--gp-space-xs);
    cursor: pointer;
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);

    &.is-active {
      background-color: var(--el-color-primary-light-1);
      border-color: var(--el-color-primary);
    }

    &--ghost {
      background-color: var(--el-fill-color-light);
      opacity: 0.5;
    }

    &--chosen {
      box-shadow: var(--el-box-shadow);
    }

    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: var(--gp-space-3xl);
      height: var(--gp-space-3xl);
      color: var(--el-text-color-secondary);
      cursor: pointer;
      background-color: transparent;
      border: 0;

      &:hover {
        color: var(--el-text-color-primary);
      }
    }
  }

  &__drag {
    color: var(--el-text-color-secondary);
    cursor: grab;
  }

  &__item-label {
    flex: 1;
    min-width: 0;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-primary);
  }

  &__empty {
    padding: var(--gp-space-xl) 0;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
    text-align: center;
  }
}
</style>
