<template>
  <div>
    <el-select
      v-model="val"
      size="small"
      placeholder="请选择页面组件"
      filterable
      @change="handleChange"
    >
      <el-option
        v-for="item in options"
        :key="item.value"
        :label="item.label"
        size="small"
        :value="item.value"
        @mouseenter="handleMouseEnter(item.value)"
      >
        <span style="float: left">
          <b>{{ item.name }}</b>
        </span>
      </el-option>
    </el-select>
    <el-select
      v-show="showPropSelet"
      v-model="propName"
      class="prop-select"
      size="small"
      placeholder="请选择属性"
      @change="handlePropChange"
    >
      <el-option
        v-for="item in propOptions"
        :key="item.value"
        :label="item.label"
        size="small"
        :value="item.value"
      />
    </el-select>
    <el-input
      v-show="showFieldInput"
      v-model="fieldName"
      placeholder="请输入字段名"
      size="small"
      class="field-input"
      @change="handleFieldChange"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';
import { getNodeName } from '../../util/node';

defineOptions({ name: 'ComponentPropValue' });

interface LogicProp {
  description: string;
  name: string;
  propType: string;
}

interface LogicModel {
  getLogic?: () => { props?: LogicProp[] };
  getModelName?: () => string;
  id: string;
}

interface ComponentPropValueModel {
  componentId?: string;
  componentName?: string;
  dataType?: string;
  field?: string;
  prop?: string;
  propName?: string;
  type: 'componentProp';
}

const props = defineProps<{ context?: unknown; value?: Partial<ComponentPropValueModel> }>();
const emit = defineEmits<{ change: [value: ComponentPropValueModel] }>();
const val = shallowRef('');
const propName = shallowRef('');
const fieldName = shallowRef('');

function getLogicModelList(): LogicModel[] {
  return [];
}

const options = computed(() =>
  getLogicModelList().map((item) => ({
    id: item.id,
    value: item.id,
    name: getNodeName(item),
    label: getNodeName(item),
  })),
);
const selectModel = computed(() => getLogicModelList().find((item) => item.id === val.value));
const propOptions = computed(() =>
  (selectModel.value?.getLogic?.().props ?? []).map((item) => ({
    label: item.description,
    value: item.name,
    dataType: item.propType,
  })),
);
const selectedProp = computed(() => propOptions.value.find((item) => item.value === propName.value));
const showPropSelet = computed(() => Boolean(val.value));
const showFieldInput = computed(
  () =>
    selectModel.value
      ?.getLogic?.()
      .props?.find((item) => item.name === propName.value)?.propType === 'object',
);

watch(
  () => props.value,
  (value) => {
    if (value?.type !== 'componentProp') return;
    val.value = value.componentId ?? '';
    propName.value = value.prop ?? '';
    fieldName.value = value.field ?? '';
  },
  { immediate: true },
);

function createValue(): ComponentPropValueModel {
  return {
    type: 'componentProp',
    prop: propName.value,
    field: fieldName.value,
    componentId: val.value,
    dataType: selectedProp.value?.dataType,
    componentName: selectModel.value?.getModelName?.(),
    propName: selectedProp.value?.label,
  };
}

function handleMouseEnter(_modelId: string): void {}
function handleChange(value: string): void {
  val.value = value;
  emit('change', createValue());
}
function handlePropChange(): void {
  emit('change', createValue());
}
function handleFieldChange(): void {
  emit('change', createValue());
}
</script>

<style scoped lang="less">
:deep(.el-select) {
  width: 100%;
}
.prop-select {
  margin-top: 2px;
}

.field-input {
  margin-top: 2px;
}
</style>
