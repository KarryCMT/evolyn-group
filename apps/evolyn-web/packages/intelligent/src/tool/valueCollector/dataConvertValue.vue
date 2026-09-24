<template>
  <div>
    <el-select v-model="nodeId" size="small" placeholder="请选择数据节点" filterable @change="handleChange">
      <el-option v-for="item in options" :key="item.value" :label="item.label" size="small" :value="item.value">
        <span style="float: left" @mouseenter.stop="handleMouseEnter(item.value)">
          {{ item.label }}
        </span>
      </el-option>
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';
import { focusNode } from '../../util/node';

defineOptions({ name: 'DataConvertValue' });

interface GraphNode {
  id: string;
  properties: { componentName?: string; dc?: unknown; name?: string };
}

interface LogicFlowLike {
  getGraphData: () => { nodes: GraphNode[] };
  getNodeModelById: (id: string) => unknown;
  graphModel: {
    eventCenter: { emit: (event: string, value?: unknown) => void };
  };
}

interface DataConvertValueModel {
  nodeId: string;
  type: 'dataConvert';
}

const props = defineProps<{
  context?: unknown;
  lf: LogicFlowLike;
  value?: Partial<DataConvertValueModel> & { field?: string };
}>();
const emit = defineEmits<{ change: [value: DataConvertValueModel] }>();
const nodeId = shallowRef('');
const options = computed(() =>
  props.lf
    .getGraphData()
    .nodes.filter((node) => node.properties.componentName === 'dataConvert' && node.properties.dc)
    .map((node) => ({ label: node.properties.name ?? node.id, value: node.id, nodeId: node.id })),
);

watch(
  () => props.value,
  (value) => {
    if (value?.type === 'dataConvert') nodeId.value = value.nodeId ?? '';
  },
  { immediate: true },
);

function handleMouseEnter(id: string): void {
  focusNode(props.lf.graphModel, id, 240);
  props.lf.graphModel.eventCenter.emit('node:hover-node', props.lf.getNodeModelById(id));
}
function handleChange(value: string): void {
  nodeId.value = value;
  emit('change', { type: 'dataConvert', nodeId: value });
}
</script>
