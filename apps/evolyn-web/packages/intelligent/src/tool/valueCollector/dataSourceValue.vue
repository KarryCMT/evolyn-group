<template>
  <div>
    <el-select v-model="nodeId" size="small" placeholder="请选择数据节点" filterable @change="handleChange">
      <el-option v-for="item in options" :key="item.value" :label="item.label" size="small" :value="item.value">
        <span style="float: left" @mouseenter.stop="handleMouseEnter(item.value)">
          {{ item.label }}
        </span>
      </el-option>
    </el-select>
    <div v-if="nodeId">
      <FieldSelector
        v-if="apiId"
        v-model="field"
        class="field-selector"
        :options="fieldOptions"
        placeholder="请选择返回字段"
      />
      <div class="custom-field" v-else>
        <el-input
          class="custom-input"
          size="small"
          v-model="field"
          placeholder="支持以.分割的多级属性"
          @change="handleCustomField"
        ></el-input>
        <!-- <el-select v-model="dataType" size="small" placeholder="请选择数据类型" filterable>
          <el-option v-for="item in TypeOptions" :key="item.value" :label="item.label" size="small" :value="item.value">
            <span style="float: left">
              {{ item.label }}
            </span>
          </el-option>
        </el-select> -->
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';
import { focusNode } from '../../util/node';
import FieldSelector from '../fieldSelector/index.vue';

defineOptions({ name: 'DataSourceValue' });

interface DataSourceConfig {
  fetchMode?: string;
  id?: string;
  name?: string;
  resourceId?: string;
}

interface GraphNode {
  id: string;
  properties: { componentName?: string; ds?: DataSourceConfig; name?: string };
}

interface LogicFlowLike {
  getGraphData: () => { nodes: GraphNode[] };
  getNodeModelById: (id: string) => unknown;
  graphModel: { eventCenter: { emit: (event: string, value?: unknown) => void } };
}

interface DataSourceValueModel {
  apiId?: string;
  apiLabel?: string;
  dataType?: string;
  field: string;
  nodeId: string;
  type: 'dataSource';
}

const props = defineProps<{ context?: unknown; lf: LogicFlowLike; value?: Partial<DataSourceValueModel> }>();
const emit = defineEmits<{ change: [value: DataSourceValueModel] }>();
const nodeId = shallowRef('');
const field = shallowRef('');
const dataType = shallowRef('string');
const fieldOptions = computed<readonly { label: string; value: string }[]>(() => []);
const options = computed(() =>
  props.lf
    .getGraphData()
    .nodes.filter((item) => item.properties.componentName === 'dataSource' && item.properties.ds)
    .map((item) => {
      const source = item.properties.ds!;
      return {
        label: `${item.properties.name ?? item.id}_${source.name ?? ''}`,
        value: item.id,
        apiId: source.fetchMode === 'redirect' ? source.resourceId : source.id,
        nodeId: item.id,
      };
    }),
);
const apiId = computed(() => options.value.find((item) => item.value === nodeId.value)?.apiId);

watch(
  () => props.value,
  (value) => {
    if (value?.type !== 'dataSource') return;
    nodeId.value = value.nodeId ?? '';
    field.value = value.field ?? '';
    dataType.value = value.dataType ?? 'string';
  },
  { immediate: true },
);

function createValue(): DataSourceValueModel {
  const option = options.value.find((item) => item.value === nodeId.value);
  return {
    type: 'dataSource',
    field: field.value,
    apiId: apiId.value,
    apiLabel: option?.label ?? '',
    nodeId: nodeId.value,
    dataType: dataType.value,
  };
}
function handleMouseEnter(id: string): void {
  focusNode(props.lf.graphModel, id, 240);
  props.lf.graphModel.eventCenter.emit('node:hover-node', props.lf.getNodeModelById(id));
}
function handleChange(value: string): void {
  nodeId.value = value;
  emit('change', createValue());
}
function handleCustomField(): void {
  emit('change', createValue());
}
</script>

<style scoped lang="less">
:deep(.el-select) {
  width: 100%;
}
.field-selector {
  margin-top: 2px;
}
.custom-input {
  margin: 5px 0;
}
</style>
