<template>
  <div class="line-content" :class="isSelected && 'selected'">
    <el-popover
      placement="right"
      :width="200"
      trigger="click"
      v-model:visible="popVisible"
      v-if="properties.condition && properties.condition && properties.condition.conditions && properties.condition.conditions.length"
    >
      <div class="popover-content">
        <div class="condition-wrap">
          <div class="condition-title">我配置的条件</div>
          <div class="condition-content">
            <div class="condition-item" v-for="(item, index) in properties.condition.conditions" :key="index">
              <img
                class="icon"
                src=""
                alt=""
              />
              <span>{{ getConditionItem(item) }}</span>
            </div>
            <div class="condition-button" @click="goCondition">去配置</div>
          </div>
        </div>
        <div class="popover-item" @click="insertNode">
          <img class="icon" src="" alt="" />
          插入节点
        </div>
      </div>
      <template #reference>
        <div class="line-icon" :class="isSelected && 'selected'" @click.stop="handleIconClick">
          <img class="icon" src="" alt="" />
        </div>
      </template>
    </el-popover>
    <el-popover v-else v-model:visible="popVisible" placement="right" :width="100" trigger="click">
      <div class="popover-content">
        <div class="popover-item" @click="goCondition">
          <img class="icon" src="" alt="" />
          添加条件
        </div>
        <div class="popover-item" @click="insertNode">
          <img class="icon" src="" alt="" />
          插入节点
        </div>
      </div>
      <template #reference>
        <div
          v-show="isSelected || isHovered"
          class="line-icon"
          :class="isSelected && 'selected'"
          @click.stop="handleIconClick"
        >
          <img class="icon" src="" alt="" />
        </div>
      </template>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue';

defineOptions({ name: 'IntelligentLogicLineNode' });

interface LegacyConditionSource {
  type?: string;
  componentName?: string;
  propName?: string;
  nodeId?: string;
  value?: string;
}

interface EdgeModel {
  id: string;
  getData?: () => unknown;
}

interface EventCenter {
  on: (event: string, handler: () => void) => void;
  off?: (event: string, handler: () => void) => void;
  emit: (event: string, payload: unknown) => void;
}

interface GraphModel {
  eventCenter: EventCenter;
  getPointByClient: (point: { x: number; y: number }) => {
    canvasOverlayPosition: { x: number; y: number };
  };
  popover?: {
    show: (options: Record<string, unknown>) => string;
    hide: (key: string) => void;
  };
}

const props = defineProps<{
  model: EdgeModel;
  graphModel: GraphModel;
  isSelected?: boolean;
  isHovered?: boolean;
  properties?: { condition?: { conditions?: Array<{ key: LegacyConditionSource }> } };
}>();
const popVisible = ref(false);
const iconEvent = ref<MouseEvent>();
const popoverItemKey = ref('');
const closeEvents = ['blank:click', 'node:click', 'edge:click'];

onMounted(() => {
  for (const event of closeEvents) props.graphModel.eventCenter.on(event, hidePop);
});

onBeforeUnmount(() => {
  for (const event of closeEvents) props.graphModel.eventCenter.off?.(event, hidePop);
  hidePop();
});

function getConditionItem(item: { key: LegacyConditionSource }): string {
  return formatValue(item.key);
}

function formatValue(value: LegacyConditionSource): string {
  switch (value.type) {
    case 'component':
    case 'componentProp':
      return `${value.componentName ?? '组件'}的${value.propName ?? '属性'}`;
    case 'dataSource':
      return `数据节点${value.nodeId ?? ''}返回值`;
    case 'initParam':
      return `初始化参数值为: ${value.value ?? ''}`;
    default:
      return '未配置条件';
  }
}

function hidePop(): void {
  popVisible.value = false;
  if (popoverItemKey.value) props.graphModel.popover?.hide(popoverItemKey.value);
}

function handleIconClick(event: MouseEvent): void {
  iconEvent.value = event;
  props.graphModel.eventCenter.emit('edge:update-model', props.model);
}

function goCondition(): void {
  props.graphModel.eventCenter.emit('edge:option-click', props.model);
  popVisible.value = false;
}

function insertNode(): void {
  const event = iconEvent.value;
  if (!event || !props.graphModel.popover) return;
  const point = props.graphModel.getPointByClient({
    x: event.clientX - event.offsetX + 16,
    y: event.clientY - event.offsetY + 6,
  });
  const canvasPoint = point.canvasOverlayPosition;
  popoverItemKey.value = props.graphModel.popover.show({
    type: 'tip1',
    delay: 100,
    key: props.model.id,
    placement: 'right',
    trigger: 'click',
    width: 16,
    height: 16,
    x: canvasPoint.x - 6,
    y: canvasPoint.y,
    props: { showConnectBlock: false },
  });
  popVisible.value = false;
}
</script>

<style scoped lang="less">
.line-content {
  background: #fff;
  font-family: PingFangSC-Regular;
  font-size: 12px;
  color: #303a51;
  --node-primary-color: #2961ef;
}
.line-icon {
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 16px;
  border: 1px solid var(--node-primary-color);
  color: var(--node-primary-color);
  background: rgba(41, 97, 239, 0.08);
  font-size: 8px;
  &.selected {
    background: #fff;
    border: 2px solid var(--node-primary-color);
    box-shadow: 0 0 6px 0 rgba(41, 97, 239, 0.5);
  }
  .icon {
    width: 8px;
    height: 8px;
  }
}
.popover-item {
  --node-primary-color: #2961ef;
  background: #f3f6fa;
  border-radius: 2px;
  padding: 0 4px;
  cursor: pointer;
  height: 28px;
  color: var(--node-primary-color);
  text-align: center;
  font-weight: 400;
  display: flex;
  justify-content: center;
  align-items: center;
  .icon {
    margin-right: 4px;
    width: 16px;
    height: 16px;
  }
  &:hover {
    background: rgba(41, 97, 239, 0.08);
  }
  &:first-child {
    margin-bottom: 4px;
  }
}
.condition-content {
  padding: 8px 0;
  border-bottom: 1px solid #e4e7ed;
  margin-bottom: 10px;
}
.condition-title {
  line-height: 16px;
  font-weight: 500;
  padding-bottom: 4px;
  border-bottom: 1px solid #e4e7ed;
  font-family: PingFangSC-Medium;
  font-size: 12px;
  color: #303a51;
  line-height: 16px;
  font-weight: 500;
}
.condition-item {
  font-weight: 400;
  height: 25px;
  display: flex;
  align-items: center;
  padding: 0 4px;
  font-family: PingFangSC-Regular;
  font-size: 12px;
  color: #303a51;
  font-weight: 400;
  .icon {
    margin-right: 4px;
    width: 12px;
    height: 12px;
  }
  span {
    display: inline-block;
    max-width: 140px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
.condition-button {
  --node-primary-color: #2961ef;
  margin-top: 8px;
  line-height: 24px;
  text-align: center;
  background: #ffffff;
  border: 1px solid var(--node-primary-color);
  border-radius: 2px;
  color: var(--node-primary-color);
  cursor: pointer;
  &:hover {
    opacity: 0.8;
  }
}
</style>
