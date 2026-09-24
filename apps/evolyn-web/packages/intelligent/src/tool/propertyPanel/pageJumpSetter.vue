<template>
  <div class="pj-panel-wrapper">
    <div class="panel-item item-wrap">
      <div class="pj-label">页面跳转类型：</div>
      <el-radio-group
        v-model="params.jumpType"
        class="pj-radio-group"
        size="small"
        @change="handleParamChange('jumpType')"
      >
        <el-radio label="internal">配置页面</el-radio>
        <el-radio label="custom">自定义页面</el-radio>
      </el-radio-group>
    </div>

    <div class="panel-item item-wrap">
      <div class="pj-label">页面打开方式：</div>
      <el-radio-group
        v-model="params.target"
        class="pj-radio-group"
        size="small"
        @change="handleParamChange('target')"
      >
        <el-radio label="_self">当前页面</el-radio>
        <el-radio label="_blank">打开新 Tab</el-radio>
      </el-radio-group>
    </div>
    <div v-if="params.jumpType === 'custom'" class="panel-item item-wrap">
      <div class="pj-label">目标系统：</div>
      <el-radio-group
        v-model="params.system"
        class="pj-radio-group"
        size="small"
        @change="handleParamChange('system')"
      >
        <el-radio label="_current">宿主系统</el-radio>
        <el-radio label="_other">其他</el-radio>
      </el-radio-group>
    </div>

    <div v-if="params.jumpType === 'internal'">
      <div class="panel-item item-wrap">
        <div class="pj-label">目标页面：</div>
        <el-select
          v-model="params.formKey"
          class="pj-select"
          placeholder="请选择"
          size="small"
          @change="handleParamChange('formKey')"
        >
          <el-option
            v-for="item in internalPageList"
            :key="item.formKey + item.version"
            :label="item.name"
            :value="item.formKey"
          />
        </el-select>
      </div>
      <div class="panel-item item-wrap">
        <div class="pj-label">页面路由：</div>
        <el-input
          v-model="params.path"
          class="pj-input"
          placeholder="请输入系统路由，例如：'preview'"
          size="small"
          @change="handleParamChange('path')"
        />
      </div>
    </div>
    <div v-else class="panel-item item-wrap">
      <div class="pj-label">目标页面：</div>
      <el-input
        v-if="params.system === '_current'"
        v-model="params.customUrl"
        type="url"
        class="pj-input"
        placeholder="请输入目标页面系统路径，例如：'path1/path2/path3'"
        size="small"
        @change="handleParamChange('customUrl')"
      />
      <el-input
        v-else
        v-model="params.customUrl"
        type="url"
        class="pj-input"
        placeholder="请输入目标页面，例如：'https://www.lingyanyun.com/path?q=xxx'"
        size="small"
        @change="handleCustomUrlChange"
      />
    </div>

    <div class="item-wrap">
      <div class="pj-label">路由参数：</div>
      <div v-for="(route, idx) in params.routeParams" :key="idx" class="param-wrapper">
        <div class="pj-label">KEY</div>
        <el-input
          v-model="route.key"
          class="pj-param-input"
          placeholder="请输入"
          size="small"
          @change="handleRouteParamKeyChange($event, idx)"
        />
        <div class="pj-label">VALUE</div>
        <ValueCollector
          class="value-select"
          :model-value="route.value"
          @update:model-value="handleRouteParamValueChange($event, idx)"
        />

        <el-link class="delete-button" type="danger" :underline="false" @click="deleteParam(idx)">删除</el-link>
      </div>
      <el-link type="primary" class="add-button" :underline="false" @click="addParam">
        ＋ 添加参数
      </el-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { ElMessage } from 'element-plus';
import type { IntelligentValueSource } from '../../schema';
import ValueCollector from '../valueCollector/index.vue';

defineOptions({ name: 'IntelligentPageJumpSetter' });

interface RouteParam {
  key: string;
  value: IntelligentValueSource;
}

interface PageJumpConfig {
  jumpType: 'internal' | 'custom';
  target: '_self' | '_blank';
  system: '_current' | '_other';
  formKey?: string;
  path: string;
  customUrl: string;
  routeParams: RouteParam[];
}

interface InternalPage {
  formKey: string;
  version: string | number;
  name: string;
}

defineProps<{ lf?: unknown; context?: unknown; current?: unknown }>();
const model = defineModel<PageJumpConfig>({ default: createDefaultConfig });
const emit = defineEmits<{ change: [value: PageJumpConfig] }>();
const params = ref<PageJumpConfig>(createDefaultConfig());
const internalPageList = ref<InternalPage[]>([]);

watch(
  model,
  (value) => {
    params.value = {
      ...createDefaultConfig(),
      ...value,
      routeParams: (value.routeParams ?? []).map((item) => ({
        ...item,
        value: { ...item.value },
      })),
    };
  },
  { immediate: true, deep: true },
);

function createDefaultConfig(): PageJumpConfig {
  return {
    jumpType: 'internal',
    target: '_self',
    system: '_current',
    path: '',
    customUrl: '',
    routeParams: [],
  };
}

function publish(): void {
  const value: PageJumpConfig = {
    ...params.value,
    routeParams: params.value.routeParams.map((item) => ({
      ...item,
      value: { ...item.value },
    })),
  };
  model.value = value;
  emit('change', value);
}

function addParam(): void {
  params.value.routeParams.push({ key: '', value: { type: 'constant' } });
  publish();
}

function deleteParam(index: number): void {
  params.value.routeParams.splice(index, 1);
  publish();
}

function handleParamChange(key: keyof PageJumpConfig): void {
  if (key === 'jumpType') {
    params.value.formKey = undefined;
    params.value.path = '';
    params.value.customUrl = '';
    params.value.routeParams = [];
  } else if (key === 'system') {
    params.value.customUrl = '';
  }
  publish();
}

function handleCustomUrlChange(value: string): void {
  try {
    const url = new URL(value);
    const entries = new Map<string, string>();
    url.searchParams.forEach((entryValue, key) => entries.set(key, entryValue));

    const hashQueryIndex = url.hash.indexOf('?');
    if (hashQueryIndex >= 0) {
      const hashParams = new URLSearchParams(url.hash.slice(hashQueryIndex + 1));
      hashParams.forEach((entryValue, key) => entries.set(key, entryValue));
      url.hash = url.hash.slice(0, hashQueryIndex);
    }
    url.search = '';
    params.value.customUrl = url.toString();
    params.value.routeParams = Array.from(entries, ([key, entryValue]) => ({
      key,
      value: { type: 'constant', value: entryValue },
    }));
  } catch {
    ElMessage.error('请输入正确的 URL');
    params.value.customUrl = '';
    params.value.routeParams = [];
  }
  publish();
}

function handleRouteParamKeyChange(value: string, index: number): void {
  const item = params.value.routeParams[index];
  if (!item) return;
  item.key = value;
  publish();
}

function handleRouteParamValueChange(value: IntelligentValueSource, index: number): void {
  const item = params.value.routeParams[index];
  if (!item) return;
  item.value = value;
  publish();
}
</script>

<style lang="less" scoped>
.pj-panel-wrapper {
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
.pj-label {
  color: #333;
  line-height: 32px;
  height: 32px;
  margin-right: 8px;
}
.pj-radio-group {
  display: flex;
  align-items: center;
}
.pj-select {
  flex: 1;
}
.pj-input {
  flex: 1;
}
.pj-param-input {
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
</style>
