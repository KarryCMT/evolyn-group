<script setup lang="ts">
import {
  RiApps2Fill,
  RiAttachmentFill,
  RiDatabase2Fill,
  RiFileList3Fill,
  RiPuzzle2Fill,
  RiRouteFill,
  RiShieldUserFill,
  RiUser3Fill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import type { Component } from 'vue';
import { computed, onMounted, ref, shallowRef } from 'vue';
import { getCurrentEdition } from '~/api/edition';
import TenantEditionFeatureGroups from '~/components/tenant/edition/TenantEditionFeatureGroups.vue';
import TenantEditionOverview from '~/components/tenant/edition/TenantEditionOverview.vue';
import TenantEditionQuotaGrid from '~/components/tenant/edition/TenantEditionQuotaGrid.vue';
import type { EditionFeatureGroup, EditionQuotaCard } from '~/components/tenant/edition/types';
import type { CurrentEdition, EditionQuota } from '~/types';

defineOptions({ name: 'TenantEditionPage' });

// 版本信息一期：所有数据来自 GET /editions/current（订阅事实源），
// 页面不再持有任何静态版本、余额、容量或权益常量
const edition = shallowRef<CurrentEdition | null>(null);
const loading = ref(false);
const loaded = ref(false);
const selectedQuota = shallowRef<EditionQuotaCard | null>(null);

async function loadEdition() {
  loading.value = true;
  try {
    edition.value = await getCurrentEdition();
  } finally {
    loading.value = false;
    loaded.value = true;
  }
}

onMounted(loadEdition);

// ---- 资源容量卡片：真实用量 + 待计量占位 ----

const quotaMeta: Record<string, { icon: Component; tone: EditionQuotaCard['tone'] }> = {
  members: { icon: RiUser3Fill, tone: 'blue' },
  apps: { icon: RiApps2Fill, tone: 'cyan' },
  storage_bytes: { icon: RiAttachmentFill, tone: 'green' },
  forms: { icon: RiFileList3Fill, tone: 'orange' },
  workflow_runs_month: { icon: RiRouteFill, tone: 'purple' },
};

const quotaNotes: Record<string, string> = {
  storage_bytes: '统计包含上传中文件的声明预留与已完成文件的实际大小。',
  members: '成员数达到上限后，新增成员将被阻止。',
};

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unitIndex = -1;
  do {
    value /= 1024;
    unitIndex += 1;
  } while (value >= 1024 && unitIndex < units.length - 1);
  const rounded = value >= 100 ? Math.round(value) : Math.round(value * 10) / 10;
  return `${rounded} ${units[unitIndex]}`;
}

/** 已用值文案：待计量资源不展示伪数值 */
function usageLabel(quota: EditionQuota): string {
  if (quota.meteringStatus !== 'ready' || quota.usage === undefined) {
    return '暂未启用统计';
  }
  switch (quota.key) {
    case 'members':
      return `已使用：${quota.usage} 人`;
    case 'apps':
    case 'forms':
      return `已使用：${quota.usage} 个`;
    case 'workflow_runs_month':
      return `已使用：${quota.usage} 次`;
    case 'storage_bytes':
      return `已使用：${formatBytes(quota.usage)}`;
    default:
      return `已使用：${quota.usage}`;
  }
}

/** 上限文案：-1 不限量 / 0 不可用 / 正数为上限 */
function limitLabel(quota: EditionQuota): string {
  if (quota.limit === -1) return '不限量';
  if (quota.limit === 0) return '不可用';
  switch (quota.key) {
    case 'members':
      return `上限：${quota.limit} 人`;
    case 'apps':
    case 'forms':
      return `上限：${quota.limit} 个`;
    case 'workflow_runs_month':
      return `上限：${quota.limit} 次/月`;
    case 'storage_bytes':
      return `上限：${formatBytes(quota.limit)}`;
    default:
      return `上限：${quota.limit}`;
  }
}

const quotaCards = computed<EditionQuotaCard[]>(() => {
  const quotas = edition.value?.quotas ?? [];
  return quotas.map((quota) => {
    const meta = quotaMeta[quota.key] ?? { icon: RiDatabase2Fill, tone: 'cyan' };
    const percent = quota.meteringStatus === 'ready' ? (quota.usagePercent ?? 0) : 0;
    return {
      id: quota.key,
      icon: meta.icon,
      title: quota.name,
      tone: meta.tone,
      meteringStatus: quota.meteringStatus,
      progress: Math.min(100, Math.round(percent)),
      usageLabel: usageLabel(quota),
      limitLabel: limitLabel(quota),
      note: quotaNotes[quota.key],
      warning: percent >= 100 ? '已超出当前版本上限' : undefined,
      limitSource: quota.limitSource,
      asOf: quota.asOf,
      resetCycle: quota.resetCycle,
    };
  });
});

// ---- 功能权益：按接口分组投影，可用性以服务端为准 ----

const featureIcons: Record<string, Component> = {
  application_management: RiApps2Fill,
  member_management: RiUser3Fill,
  department_management: RiFileList3Fill,
  group_management: RiPuzzle2Fill,
  role_permission: RiShieldUserFill,
  file_upload: RiAttachmentFill,
};

const featureGroups = computed<EditionFeatureGroup[]>(() => {
  const groups: EditionFeatureGroup[] = [];
  const index = new Map<string, EditionFeatureGroup>();
  for (const feature of edition.value?.features ?? []) {
    let group = index.get(feature.group);
    if (!group) {
      group = { id: feature.group, title: feature.group, items: [] };
      index.set(feature.group, group);
      groups.push(group);
    }
    group.items.push({
      id: feature.key,
      title: feature.name,
      available: feature.available,
      icon: featureIcons[feature.key] ?? RiPuzzle2Fill,
      meta: feature.description,
    });
  }
  return groups;
});

const limitSourceLabels: Record<string, string> = {
  plan_version: '套餐默认',
  tenant_override: '租户特批覆盖',
  legacy_quota: '历史配额覆盖',
  expiry_fallback: '订阅到期回退（免费版）',
};

// ---- 交互 ----

/** 咨询升级：公开联系入口未配置时如实提示，不伪造提交成功（设计 4.3.1） */
function handleConsult() {
  ElMessage.info('如需升级版本，请联系平台管理员');
}

/** 对话框关闭时只重置当前详情，容量数据仍以接口数据源为准。 */
function updateQuotaDialog(visible: boolean) {
  if (!visible) {
    selectedQuota.value = null;
  }
}
</script>

<template>
  <section v-loading="loading" class="tenant-edition-page" aria-label="版本信息">
    <TenantEditionOverview
      v-if="edition"
      :subscription="edition.subscription"
      @consult="handleConsult"
    />
    <el-empty v-else-if="loaded" description="暂无法获取版本信息，请稍后重试">
      <el-button @click="loadEdition">重新加载</el-button>
    </el-empty>

    <TenantEditionQuotaGrid v-if="edition" :cards="quotaCards" @detail="selectedQuota = $event" />
    <TenantEditionFeatureGroups v-if="edition" :groups="featureGroups" />

    <el-dialog
      :model-value="selectedQuota !== null"
      class="tenant-edition-page__dialog"
      width="440px"
      :title="selectedQuota ? `${selectedQuota.title}详情` : '容量详情'"
      @update:model-value="updateQuotaDialog"
    >
      <template v-if="selectedQuota">
        <dl class="tenant-edition-page__quota-detail">
          <div>
            <dt>当前用量</dt>
            <dd>{{ selectedQuota.usageLabel }}</dd>
          </div>
          <div>
            <dt>套餐上限</dt>
            <dd>{{ selectedQuota.limitLabel }}</dd>
          </div>
          <div v-if="selectedQuota.limitSource">
            <dt>上限来源</dt>
            <dd>{{ limitSourceLabels[selectedQuota.limitSource] ?? selectedQuota.limitSource }}</dd>
          </div>
          <div v-if="selectedQuota.resetCycle">
            <dt>重置周期</dt>
            <dd>每自然月重置</dd>
          </div>
          <div v-if="selectedQuota.asOf">
            <dt>统计时间</dt>
            <dd>{{ selectedQuota.asOf }}</dd>
          </div>
          <div v-if="selectedQuota.note">
            <dt>使用说明</dt>
            <dd>{{ selectedQuota.note }}</dd>
          </div>
        </dl>
      </template>
      <template #footer>
        <el-button type="primary" @click="selectedQuota = null">知道了</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped lang="scss">
.tenant-edition-page {
  box-sizing: border-box;
  min-width: 1080px;
  min-height: 100%;
  padding: var(--el-space-3xl);
  background: var(--el-bg-color);

  &__quota-detail {
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
}

@media (max-width: 1080px) {
  .tenant-edition-page {
    min-width: 0;
  }
}

@media (max-width: 640px) {
  .tenant-edition-page {
    padding: var(--el-space-xl) var(--el-space-xl) var(--el-space-3xl);
  }
}
</style>
