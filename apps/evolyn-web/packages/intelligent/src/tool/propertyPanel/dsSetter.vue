<template>
  <div class="ds-panel-wrapper" v-loading="isFetching">
    <div class="panel-item item-wrap">
      <div class="ds-label">请求来源：</div>
      <el-radio-group
        class="ds-type-radio"
        size="small"
        v-model="fetchMode"
        @change="handleDsTypeChange"
      >
        <!-- 直连模式 -->
        <el-radio label="direct">宿主系统</el-radio>
        <!-- 转发模式 -->
        <el-radio label="redirect">资源引擎</el-radio>
        <!-- 自定义接口 -->
        <el-radio label="custom">自定义</el-radio>
      </el-radio-group>
    </div>
    <div class="panel-item item-wrap" v-if="fetchMode !== 'custom'">
      <div class="ds-label">请求接口：</div>
      <el-select
        filterable
        v-model="apiId"
        class="ds-select"
        size="small"
        placeholder="请选择"
        v-if="fetchMode === 'direct'"
        @change="handleApiChange"
      >
        <el-option
          v-for="item in options"
          :key="item.value"
          :label="item.label"
          size="small"
          :value="item.value"
        >
        </el-option>
      </el-select>

      <el-select
        filterable
        v-model="apiId"
        class="ds-select"
        size="small"
        placeholder="请选择"
        v-else-if="fetchMode === 'redirect'"
        @change="handleApiChange"
      >
        <el-option
          v-for="item in options"
          :key="item.value"
          :label="item.label"
          size="small"
          :value="item.value"
        >
        </el-option>
      </el-select>
    </div>
    <!-- curl语句导入 -->
    <div class="panel-item item-wrap" v-if="fetchMode === 'custom'">
      <CurlImport @import="$_importData"></CurlImport>
    </div>
    <div class="panel-item item-wrap" v-if="fetchMode === 'custom'">
      <div class="ds-label">请求接口名称：</div>
      <el-input
        v-model="requestName"
        placeholder="请输入"
        size="small"
        @change="handleChange"
      >
      </el-input>
    </div>
    <div class="panel-item item-wrap" v-if="fetchMode === 'custom'">
      <div class="ds-label">请求接口地址：</div>
      <el-input
        v-model="requestUrl"
        placeholder="请输入"
        size="small"
        @change="handleChange"
      >
      </el-input>
    </div>
    <div class="panel-item item-wrap" v-if="fetchMode === 'custom'">
      <div class="ds-label">请求方法：</div>
      <el-select
        filterable
        v-model="requestMethod"
        class="ds-select"
        size="small"
        placeholder="请选择"
        @change="handleChange"
      >
        <el-option
          v-for="item in requestMethodMap"
          :key="item.value"
          :label="item.label"
          size="small"
          :value="item.value"
        >
        </el-option>
      </el-select>
    </div>
    <div class="item-wrap">
      <div class="ds-label">请求参数配置：</div>
      <el-tabs v-model="activeType">
        <el-tab-pane label="Query" name="query">
          <param-collector
            v-model="queryParams"
            :lf="lf"
            :param-list="queryParamList"
            :context="context"
            @change="handleQueryParamsChange"
          ></param-collector>
        </el-tab-pane>
        <el-tab-pane label="Body" name="body">
          <param-collector
            v-model="bodyParams"
            :lf="lf"
            :param-list="bodyParamList"
            :context="context"
            @change="handleBodyParamsChange"
          ></param-collector>
        </el-tab-pane>
      </el-tabs>
    </div>
    <div class="item-wrap" v-if="fetchMode === 'direct'">
      <div class="ds-label">请求失败配置：</div>
      <div>
        当请求失败时，是否继续执行后续逻辑：
        <el-switch
          v-model="continueOnError"
          @change="handleErrorChange"
          active-text="是"
          inactive-text="否"
          size="mini"
        ></el-switch>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import type { IntelligentValueSource } from '../../schema';
import ParamCollector from '../paramCollector/index.vue';
import CurlImport from './curlImport.vue';

defineOptions({ name: 'IntelligentDataSourceSetter' });

type FetchMode = 'direct' | 'redirect' | 'custom';
type RequestMethod = 'GET' | 'POST';

interface RequestParam {
  key: string;
  keyType?: string;
  paramType?: string;
  required?: boolean | number;
  value?: IntelligentValueSource;
}

interface ApiResource {
  id: string;
  name?: string;
  resourceKey?: string;
  apiKey?: string;
  fetchMode?: FetchMode;
  resourceId?: string;
  tenant?: string;
  extJson?: unknown;
  httpApiResourceVO?: { url?: string; requestType?: RequestMethod; contentType?: string };
}

interface DataSourceConfig {
  id?: string;
  name?: string;
  key?: string;
  url?: string;
  method?: RequestMethod;
  contentType?: string;
  fetchMode?: FetchMode;
  resourceId?: string;
  tenant?: string;
  extJson?: unknown;
  queryParams?: RequestParam[];
  bodyParams?: RequestParam[];
  headerParams?: RequestParam[];
  continueOnError?: boolean;
}

interface ImportedRequest {
  requestMethod: RequestMethod;
  requestUrl?: string;
  queryParams: RequestParam[];
  bodyParams: RequestParam[];
}

defineProps<{ lf?: unknown; context?: unknown; current?: unknown }>();
const model = defineModel<DataSourceConfig>({ default: () => ({ fetchMode: 'direct' }) });
const emit = defineEmits<{ change: [value: DataSourceConfig] }>();
const requestMethodMap = [
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
] satisfies Array<{ value: RequestMethod; label: string }>;

const requestName = ref('');
const requestUrl = ref('');
const requestMethod = ref<RequestMethod>('GET');
const fetchMode = ref<FetchMode>('direct');
const apiId = ref('');
const activeType = ref<'query' | 'body'>('query');
const continueOnError = ref(false);
const apiList = ref<ApiResource[]>([]);
const options = ref<Array<{ label: string; value: string }>>([]);
const queryParamList = ref<RequestParam[]>([]);
const bodyParamList = ref<RequestParam[]>([]);
const queryParams = ref<RequestParam[]>([]);
const bodyParams = ref<RequestParam[]>([]);
const headerParams = ref<RequestParam[]>([]);
const isFetching = ref(false);
const api = computed(() => apiList.value.find((item) => item.id === apiId.value));

watch(
  model,
  async (value) => {
    fetchMode.value = value.fetchMode ?? 'direct';
    apiId.value = value.id ?? '';
    queryParams.value = cloneParams(value.queryParams);
    bodyParams.value = cloneParams(value.bodyParams);
    headerParams.value = cloneParams(value.headerParams);
    continueOnError.value = value.continueOnError ?? false;
    requestName.value = value.name ?? '';
    requestUrl.value = value.url ?? '';
    requestMethod.value = value.method ?? 'GET';
    if (fetchMode.value === 'custom') {
      queryParamList.value = cloneParams(queryParams.value);
      bodyParamList.value = cloneParams(bodyParams.value);
    } else if (apiId.value) {
      if (!options.value.length) await getApiOptions();
      getParamList();
    }
  },
  { immediate: true, deep: true },
);

onMounted(() => {
  void getApiOptions();
});

function cloneParams(value?: RequestParam[]): RequestParam[] {
  return (value ?? []).map((item) => ({
    ...item,
    value: item.value ? { ...item.value } : undefined,
  }));
}

async function getApiOptions(): Promise<void> {
  // 数据源目录接入前保留稳定的空态，不再依赖 Vue 2 created 钩子。
  options.value = [];
}

function getParamList(): void {
  const resourceId = fetchMode.value === 'direct' ? api.value?.id : api.value?.resourceId;
  if (!resourceId) return;
}

function handleDsTypeChange(): void {
  reset();
  apiId.value = '';
  void getApiOptions();
  handleChange();
}

function handleApiChange(value: string): void {
  reset();
  apiId.value = value;
  getParamList();
  handleChange();
}

function handleQueryParamsChange(value: RequestParam[]): void {
  queryParams.value = cloneParams(value);
  handleChange();
}

function handleBodyParamsChange(value: RequestParam[]): void {
  bodyParams.value = cloneParams(value);
  handleChange();
}

function handleErrorChange(): void {
  handleChange();
}

function handleChange(): void {
  let value: DataSourceConfig;
  if (fetchMode.value === 'custom') {
    value = {
      name: requestName.value,
      url: requestUrl.value,
      method: requestMethod.value,
      fetchMode: 'custom',
      queryParams: cloneParams(queryParams.value),
      bodyParams: cloneParams(bodyParams.value),
      headerParams: cloneParams(headerParams.value),
    };
  } else {
    const selectedApi = api.value;
    value = {
      id: selectedApi?.id,
      name: selectedApi?.name,
      key: fetchMode.value === 'direct' ? selectedApi?.resourceKey : selectedApi?.apiKey,
      url: selectedApi?.httpApiResourceVO?.url,
      method: selectedApi?.httpApiResourceVO?.requestType,
      contentType: selectedApi?.httpApiResourceVO?.contentType,
      fetchMode: selectedApi?.fetchMode ?? fetchMode.value,
      resourceId: selectedApi?.resourceId,
      tenant: selectedApi?.tenant,
      extJson: selectedApi?.extJson,
      queryParams: cloneParams(queryParams.value),
      bodyParams: cloneParams(bodyParams.value),
      headerParams: cloneParams(headerParams.value),
      continueOnError: continueOnError.value,
    };
  }
  model.value = value;
  emit('change', value);
}

function reset(): void {
  queryParamList.value = [];
  bodyParamList.value = [];
  queryParams.value = [];
  bodyParams.value = [];
  headerParams.value = [];
}

function $_importData(value: ImportedRequest): void {
  requestMethod.value = value.requestMethod;
  requestUrl.value = value.requestUrl ?? '';
  queryParams.value = cloneParams(value.queryParams);
  bodyParams.value = cloneParams(value.bodyParams);
  queryParamList.value = cloneParams(value.queryParams);
  bodyParamList.value = cloneParams(value.bodyParams);
  handleChange();
}
</script>

<style scoped lang="less">
.ds-panel-wrapper {
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
.ds-label {
  color: #333;
  line-height: 32px;
  height: 32px;
  margin-right: 8px;
  min-width: 90px;
}
.ds-type-radio {
  display: flex;
  align-items: center;
}
.ds-select {
  flex: 1;
}
</style>
