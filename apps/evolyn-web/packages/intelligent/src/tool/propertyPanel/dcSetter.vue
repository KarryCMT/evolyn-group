<template>
  <div class="dc-panel-wrapper">
    <div class="item-wrap">
      <div class="dc-label">变量：</div>
      <div v-for="(source, idx) in params.convertList" :key="idx" class="param-wrapper">
        <div class="dc-label">KEY: </div>
        <el-input
          v-model="source.key"
          class="dc-param-input"
          placeholder="请输入"
          size="small"
          @change="handleResourceKeyChange($event, idx)"
        ></el-input>
        <div class="dc-label"><span>=</span>VALUE:</div>
        <ValueCollector
          class="value-select"
          :model-value="source.value"
          @update:model-value="handleResourceValueChange($event, idx)"
        />

        <el-link class="delete-button" type="danger" :underline="false" @click="deleteParam(idx)">删除</el-link>
      </div>
      <el-link type="primary" class="add-button" :underline="false" @click="addParam">
        ＋ 添加变量
      </el-link>
    </div>

    <div class="item-wrap">
      <div class="dc-label">转换函数体：</div>
      <el-alert title="如上配置的转换源数据 key 可直接作为变量名在下面函数体中使用" type="warning"> </el-alert>
      <div id="my-editor"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import type { IntelligentValueSource } from '../../schema';
import ValueCollector from '../valueCollector/index.vue';

defineOptions({ name: 'IntelligentDataConvertSetter' });

interface ConvertParam {
  key: string;
  value: IntelligentValueSource;
}

interface ConvertConfig {
  convertList: ConvertParam[];
  convertCode: string;
}

defineProps<{ context?: unknown; lf?: unknown }>();
const model = defineModel<ConvertConfig>({
  default: () => ({ convertList: [], convertCode: '' }),
});
const emit = defineEmits<{ change: [value: ConvertConfig] }>();
const params = ref<ConvertConfig>({ convertList: [], convertCode: '' });

watch(
  model,
  (value) => {
    params.value = {
      convertCode: value.convertCode ?? '',
      convertList: (value.convertList ?? []).map((item) => ({
        ...item,
        value: { ...item.value },
      })),
    };
  },
  { immediate: true, deep: true },
);

function publish(): void {
  const value: ConvertConfig = {
    convertCode: params.value.convertCode,
    convertList: params.value.convertList.map((item) => ({
      ...item,
      value: { ...item.value },
    })),
  };
  model.value = value;
  emit('change', value);
}

function addParam(): void {
  params.value.convertList.push({ key: '', value: { type: 'constant' } });
  publish();
}

function deleteParam(index: number): void {
  params.value.convertList.splice(index, 1);
  publish();
}

function handleResourceKeyChange(value: string, index: number): void {
  params.value.convertList[index].key = value;
  publish();
}

function handleResourceValueChange(value: IntelligentValueSource, index: number): void {
  params.value.convertList[index].value = value;
  publish();
}
</script>

<style scoped lang="less">
.convert-panel-wrapper {
  width: 100%;
}
.panel-item {
  width: 100%;
  display: inline-flex;
}
.item-wrap {
  background: #f3f6fa;
  border-radius: 4px;
  font-family: PingFangSC-Regular;
  font-size: 12px;
  color: #303a51;
  line-height: 16px;
  font-weight: 400;
  padding: 9px 12px;
}
.dc-label {
  color: #333;
  line-height: 32px;
  height: 32px;
  margin-right: 8px;
  display: flex;
  & > span {
    margin-right: 8px;
  }
}
.dc-radio-group {
  display: flex;
  align-items: center;
}
.dc-select {
  flex: 1;
}
.dc-input {
  flex: 1;
}
.dc-param-input {
  margin-right: 10px;
  width: 180px;
}

.param-wrapper {
  display: flex;
  // align-items: center;
  margin-bottom: 10px;
}
.delete-button {
  margin-left: 8px;
}
.add-button {
  margin-top: 10px;
}

#my-editor {
  position: relative;
  height: 240px;
}
</style>
