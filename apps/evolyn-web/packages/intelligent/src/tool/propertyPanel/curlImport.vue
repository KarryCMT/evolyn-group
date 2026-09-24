<template>
  <div class="curl-import">
  <el-button class="import-btn" text @click="openDialog">从 cURL 语句导入</el-button>
  <el-dialog v-model="dialogInfo.visible" destroy-on-close title="cURL 语句导入" class="ds-dialog__import" append-to-body>
    <div>
      <el-input type="textarea" v-model="dialogInfo.value" :autosize="{minRows: 8}"></el-input>
    </div>
    <template #footer>
      <div class="btns">
        <el-button size="small" @click="cancelImport">取消</el-button>
        <el-button size="small" type="primary" @click="confirmImport">导入</el-button>
      </div>
    </template>
  </el-dialog>
</div>
</template>

<script setup lang="ts">
import { reactive } from 'vue';
import type { IntelligentValueSource } from '../../schema';

defineOptions({ name: 'IntelligentCurlImport' });

interface ImportedParam {
  key: string;
  keyType: 'custom';
  value: IntelligentValueSource;
  required: 0;
}

interface ImportedRequest {
  requestMethod: 'GET' | 'POST';
  requestUrl?: string;
  queryParams: ImportedParam[];
  bodyParams: ImportedParam[];
}

const emit = defineEmits<{ import: [value: ImportedRequest] }>();
const dialogInfo = reactive({ visible: false, value: '' });

function openDialog(): void {
  dialogInfo.visible = true;
  dialogInfo.value = '';
}

function cancelImport(): void {
  dialogInfo.visible = false;
}

function confirmImport(): void {
  emit('import', parseCurl(dialogInfo.value));
  dialogInfo.visible = false;
}

function parseCurl(source: string): ImportedRequest {
  const haveGetOption = /(?:^|\s)(?:-G|--get)(?:\s|$)/.test(source);
  const result: ImportedRequest = {
    requestMethod: haveGetOption ? 'GET' : 'GET',
    queryParams: [],
    bodyParams: [],
  };
  const method = source.match(/(?:-X|--request)\s+['"]?(GET|POST)['"]?/i)?.[1]?.toUpperCase();
  if (method === 'GET' || method === 'POST') result.requestMethod = method;

  const urlMatch = source.match(/https?:\/\/[^\s'"\\]+/i);
  const queryValues: Record<string, unknown> = {};
  const bodyValues: Record<string, unknown> = {};
  if (urlMatch) {
    const parsedUrl = new URL(urlMatch[0]);
    result.requestUrl = `${parsedUrl.origin}${parsedUrl.pathname}`;
    parsedUrl.searchParams.forEach((value, key) => {
      queryValues[key] = value;
    });
  }

  // 支持常见的 -d/--data* 单引号或双引号写法，导入后统一转成当前值协议。
  const dataExpression = /(?:^|\s)(?:-d|--data(?:-urlencode|-raw|-binary|-ascii)?)\s+(['"])([\s\S]*?)\1/g;
  for (const match of source.matchAll(dataExpression)) {
    const rawValue = match[2];
    let values: Record<string, unknown> = {};
    try {
      const parsed = JSON.parse(rawValue) as unknown;
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        values = parsed as Record<string, unknown>;
      }
    } catch {
      const separator = rawValue.indexOf('=');
      if (separator > 0) {
        values[rawValue.slice(0, separator)] = rawValue.slice(separator + 1);
      }
    }
    Object.assign(haveGetOption ? queryValues : bodyValues, values);
    if (!haveGetOption && Object.keys(values).length > 0) result.requestMethod = 'POST';
  }

  result.queryParams = toParamRows(queryValues);
  result.bodyParams = toParamRows(bodyValues);
  return result;
}

function toParamRows(values: Record<string, unknown>): ImportedParam[] {
  return Object.entries(values).map(([key, value]) => ({
    key,
    keyType: 'custom',
    value: {
      type: 'constant',
      value: typeof value === 'string' ? value : JSON.stringify(value),
    },
    required: 0,
  }));
}
</script>

<style lang="less" scoped>
.curl-import {
  .import-btn {
    font-size: 12px;
    padding: 0;
    border: 0;
    font-size: 12px;
  }
  :deep(.ds-dialog__import) {
    width: 800px;
    .el-dialog__body {

    }
    .btns {
      text-align: right;
    }
  }
}
</style>
