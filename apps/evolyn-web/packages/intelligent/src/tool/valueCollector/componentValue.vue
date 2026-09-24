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
        <span 
          style="float: left"
        >
          <b>{{ item.name }}</b>的{{ item.desc }}
        </span>
      </el-option>
    </el-select>
    <el-input 
      v-show="showFieldInput"
      v-model="fieldName"
      placeholder="请输入字段名" 
      size="small"
      class="field-input"
      @change="handleFieldChange"
    >
    </el-input>
  </div>
</template>

<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';
import { EDITOR_EVENT } from '../../util/constant';
import { getNodeName } from '../../util/node';

defineOptions({ name: 'ComponentValue' });

interface LogicProp {
  description: string;
  name: string;
  propType: string;
}

interface LogicModel {
  getLogic: () => { props?: LogicProp[] };
  getModelName?: () => string;
  id: string;
}

interface ComponentValueModel {
  componentId?: string;
  componentName?: string;
  dataType?: string;
  field?: string;
  prop: 'value';
  propName: '值';
  type: 'component';
}

const props = defineProps<{
  context?: { eventCenter?: { emit: (event: string, value?: unknown) => void } };
  value?: Partial<ComponentValueModel>;
}>();
const emit = defineEmits<{ change: [value: ComponentValueModel] }>();
const val = shallowRef('');
const fieldName = shallowRef('');

function getInputModelList(): LogicModel[] {
  return [];
}

const selectModel = computed(() => getInputModelList().find((item) => item.id === val.value));
const valueProp = computed(() =>
  selectModel.value?.getLogic().props?.find((item) => item.name === 'value'),
);
const showFieldInput = computed(() => valueProp.value?.propType === 'object');
const options = computed(() =>
  getInputModelList().flatMap((item) => {
    const prop = item.getLogic().props?.find((candidate) => candidate.name === 'value');
    if (!prop) return [];
    return [{
      id: item.id,
      name: getNodeName(item),
      value: item.id,
      desc: prop.description,
      label: `${getNodeName(item)}的${prop.description}`,
    }];
  }),
);

watch(
  () => props.value,
  (value) => {
    if (value?.type !== 'component') return;
    val.value = value.componentId ?? '';
    fieldName.value = value.field ?? '';
  },
  { immediate: true },
);

function createValue(): ComponentValueModel {
  return {
    type: 'component',
    prop: 'value',
    field: fieldName.value,
    componentId: val.value,
    componentName: selectModel.value?.getModelName?.(),
    propName: '值',
    dataType: valueProp.value?.propType,
  };
}

function handleMouseEnter(modelId: string): void {
  const target = getInputModelList().find((item) => item.id === modelId);
  props.context?.eventCenter?.emit(EDITOR_EVENT.LOGIC_NODE_HOVER, target);
}
function handleChange(value: string): void {
  val.value = value;
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
.field-input {
  margin-top: 2px;
}
</style>
