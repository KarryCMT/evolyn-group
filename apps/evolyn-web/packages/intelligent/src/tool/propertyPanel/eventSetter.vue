<template>
  <div>
    <div class="event-panel-wrapper setter-wrap">
      <div class="event-label">触发后续行为的事件：</div>
      <el-select 
        :value="selectValue"
        clearable
        class="event-select" 
        size="small" 
        placeholder="请选择"
        @change="handleChange"
        >
        <el-option
          v-for="item in eventOptions"
          :key="item.value"
          :label="item.label"
          size="small"
          :value="item.value">
        </el-option>
      </el-select>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

defineOptions({ name: 'IntelligentEventSetter' });

interface EventOption {
  label: string;
  value: string;
}

interface EventValue {
  key?: string;
  keyDefine?: string;
}

interface EventLogicSource {
  getLogic: () => { events?: Array<{ description: string; name: string }> };
}

const props = defineProps<{
  context?: unknown;
  current: EventLogicSource;
  lf?: unknown;
}>();
const model = defineModel<EventValue>({ default: () => ({}) });
const emit = defineEmits<{ change: [value: EventValue] }>();
const selectValue = computed(() => model.value.key);
const eventOptions = computed<EventOption[]>(() =>
  (props.current.getLogic().events ?? []).map((item) => ({
    label: item.description,
    value: item.name,
  })),
);

function handleChange(value: string): void {
  const target = eventOptions.value.find((item) => item.value === value);
  const nextValue = { keyDefine: target?.label, key: target?.value };
  model.value = nextValue;
  emit('change', nextValue);
}
</script>

<style scoped lang="less">
.event-panel-wrapper {
  display: inline-flex;
  width: 100%;
}
.event-label {
  color: #333;
  line-height: 32px;
  height: 32px;
  margin-right: 8px;
}
.event-select {
  flex: 1;
}
.setter-wrap {
  background: #F3F6FA;
  border-radius: 4px;
  font-family: PingFangSC-Regular;
  font-size: 12px;
  color: #303A51;
  line-height: 16px;
  font-weight: 400;
  padding: 9px 12px;
}
</style>
