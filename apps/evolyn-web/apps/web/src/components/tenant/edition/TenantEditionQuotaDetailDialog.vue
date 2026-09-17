<script setup lang="ts">
import type { EditionQuotaCard } from './types';
import { ElDialog } from 'element-plus';

defineProps<{
  limitSourceLabels: Readonly<Record<string, string>>;
}>();

const quota = defineModel<EditionQuotaCard | null>({ required: true });
</script>

<template>
  <ElDialog
    :model-value="quota !== null"
    class="tenant-edition-quota-dialog"
    width="440px"
    :title="quota ? `${quota.title}详情` : '容量详情'"
    @update:model-value="(visible) => !visible && (quota = null)"
  >
    <template v-if="quota">
      <dl class="tenant-edition-quota-dialog__detail">
        <div>
          <dt>当前用量</dt>
          <dd>{{ quota.usageLabel }}</dd>
        </div>
        <div>
          <dt>套餐上限</dt>
          <dd>{{ quota.limitLabel }}</dd>
        </div>
        <div v-if="quota.limitSource">
          <dt>上限来源</dt>
          <dd>{{ limitSourceLabels[quota.limitSource] ?? quota.limitSource }}</dd>
        </div>
        <div v-if="quota.resetCycle">
          <dt>重置周期</dt>
          <dd>每自然月重置</dd>
        </div>
        <div v-if="quota.asOf">
          <dt>统计时间</dt>
          <dd>{{ quota.asOf }}</dd>
        </div>
        <div v-if="quota.note">
          <dt>使用说明</dt>
          <dd>{{ quota.note }}</dd>
        </div>
      </dl>
    </template>
    <template #footer>
      <el-button type="primary" @click="quota = null"> 知道了 </el-button>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.tenant-edition-quota-dialog__detail {
  margin: 0;

  div {
    display: grid;
    grid-template-columns: 92px minmax(0, 1fr);
    gap: var(--el-space-lg);
    padding: var(--el-space-lg) 0;
    border-bottom: 1px solid var(--el-border-color-lighter);

    &:last-child {
      border-bottom: 0;
    }
  }

  dt,
  dd {
    margin: 0;
    font-size: var(--el-font-size-base);
    line-height: 22px;
  }

  dt {
    color: var(--el-text-color-secondary);
  }

  dd {
    color: var(--el-text-color-regular);
  }
}
</style>
