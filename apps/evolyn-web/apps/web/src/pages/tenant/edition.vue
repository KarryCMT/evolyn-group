<script setup lang="ts">
import {
  RiAiGenerate2Fill,
  RiApps2Fill,
  RiAttachmentFill,
  RiBarChartBoxFill,
  RiCalculatorFill,
  RiCoinsFill,
  RiDatabase2Fill,
  RiFileList3Fill,
  RiFileSettingsFill,
  RiLayoutGridFill,
  RiLightbulbFill,
  RiLock2Fill,
  RiPaletteFill,
  RiPenNibFill,
  RiPieChart2Fill,
  RiPrinterFill,
  RiPuzzle2Fill,
  RiRecycleFill,
  RiRobotFill,
  RiRouteFill,
  RiShareForwardFill,
  RiShieldUserFill,
  RiTableFill,
  RiUser3Fill,
  RiUserSettingsFill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { shallowRef } from 'vue';
import TenantEditionFeatureGroups from '~/components/tenant/edition/TenantEditionFeatureGroups.vue';
import TenantEditionOverview from '~/components/tenant/edition/TenantEditionOverview.vue';
import TenantEditionQuotaGrid from '~/components/tenant/edition/TenantEditionQuotaGrid.vue';
import type { EditionFeatureGroup, EditionQuotaCard } from '~/components/tenant/edition/types';

defineOptions({ name: 'TenantEditionPage' });

// 接口接入前使用和灵衍云试用版一致的展示数据；后续仅需替换该数据源。
const quotaCards: EditionQuotaCard[] = [
  {
    icon: RiUser3Fill,
    id: 'members',
    limitLabel: '上限：30人',
    note: '成员数达到上限后，请升级版本或调整成员席位。',
    progress: 4,
    title: '可用人数',
    tone: 'blue',
    usageLabel: '已使用：1人',
  },
  {
    icon: RiUserSettingsFill,
    id: 'admins',
    limitLabel: '总量：99,999人',
    note: '子管理员 = 普通管理员 + 应用管理员，不包含赠送的 5 个系统管理员。',
    progress: 0,
    title: '子管理员',
    tone: 'purple',
    usageLabel: '已用：0人',
  },
  {
    icon: RiAttachmentFill,
    id: 'attachments',
    limitLabel: '每年总量：120 GB',
    note: '超出年度总量后消耗云币支付，19.92 云币 / GB。',
    progress: 0,
    title: '附件上传量',
    tone: 'cyan',
    usageLabel: '总量内已用：0 Bytes',
  },
  {
    icon: RiFileList3Fill,
    id: 'records',
    limitLabel: '总量：750,000条',
    note: '表单数据量按企业内全部应用汇总计算。',
    progress: 1,
    title: '总数据量',
    tone: 'orange',
    usageLabel: '已用：2,152条',
  },
  {
    icon: RiDatabase2Fill,
    id: 'data-factory',
    limitLabel: '总量：50个',
    note: '用量已超免费版限制。',
    progress: 16,
    title: '数据工厂',
    tone: 'green',
    usageLabel: '已用：8个',
    warning: '用量已超免费版限制',
  },
  {
    icon: RiLightbulbFill,
    id: 'assistant',
    limitLabel: '每年总量：1,000,000次',
    note: '超出年度总量后消耗云币支付，0.01 云币 / 次。',
    progress: 0,
    title: '智能助手',
    tone: 'purple',
    usageLabel: '总量内已用：0次',
  },
  {
    icon: RiRouteFill,
    id: 'workflow',
    limitLabel: '总量：20个',
    note: '流程分析支持按实例查看运行效率。',
    progress: 5,
    title: '流程分析',
    tone: 'cyan',
    usageLabel: '已用：1个',
  },
  {
    icon: RiApps2Fill,
    id: 'applications',
    limitLabel: '无限制',
    progress: 40,
    title: '应用',
    tone: 'cyan',
    usageLabel: '已用：6个',
  },
  {
    icon: RiTableFill,
    id: 'aggregate-tables',
    limitLabel: '总量：120个',
    note: '聚合表总量按企业维度统计。',
    progress: 1,
    title: '聚合表',
    tone: 'purple',
    usageLabel: '已用：2个',
  },
  {
    icon: RiAiGenerate2Fill,
    id: 'ai',
    limitLabel: '每年总量：500 点',
    progress: 0,
    title: 'AI 额度',
    tone: 'cyan',
    usageLabel: '已用：0 点',
  },
];

const featureGroups: EditionFeatureGroup[] = [
  {
    id: 'form-collaboration',
    title: '表单协作',
    items: [
      { available: true, icon: RiPenNibFill, id: 'signature', title: '手写签名' },
      { available: true, icon: RiCalculatorFill, id: 'calculation', title: '计算' },
      { available: true, icon: RiPaletteFill, id: 'form-theme', title: '自定义表单外链样式' },
      { available: true, icon: RiFileSettingsFill, id: 'detail-page', title: '自定义详情页' },
      {
        available: true,
        icon: RiShieldUserFill,
        id: 'submit-success',
        title: '自定义提交成功页面',
      },
      { available: true, icon: RiAttachmentFill, id: 'export-attachment', title: '批量导出附件' },
      { available: true, icon: RiShareForwardFill, id: 'cross-app', title: '跨应用' },
      {
        available: true,
        icon: RiPrinterFill,
        id: 'print',
        meta: '已使用3个自定义打印模板',
        title: '自定义打印',
      },
      { available: true, icon: RiLayoutGridFill, id: 'view', title: '高级视图' },
      { available: true, icon: RiFileList3Fill, id: 'form-extension', title: '表单开放性增强' },
      { available: true, icon: RiRobotFill, id: 'assistant-node', title: '智能助手高级节点' },
      { available: true, icon: RiRecycleFill, id: 'workflow-test', title: '流程测试' },
    ],
  },
  {
    id: 'basic-data',
    title: '基础数据',
    items: [
      {
        available: true,
        icon: RiFileList3Fill,
        id: 'record-limit',
        meta: '300,000条',
        title: '单表数据上限',
      },
      {
        available: true,
        icon: RiAttachmentFill,
        id: 'file-limit',
        meta: '20 MB',
        title: '单个文件上传上限',
      },
    ],
  },
  {
    id: 'visualization',
    title: '数据可视化',
    items: [
      { available: true, icon: RiBarChartBoxFill, id: 'advanced-charts', title: '仪表盘高级图表' },
      { available: true, icon: RiLightbulbFill, id: 'warning', title: '仪表盘数据预警' },
      { available: true, icon: RiPieChart2Fill, id: 'style', title: '仪表盘样式增强' },
    ],
  },
  {
    id: 'business',
    title: '业务套件',
    items: [
      { available: true, icon: RiFileList3Fill, id: 'knowledge', meta: '50个', title: '知识库' },
      { available: false, icon: RiApps2Fill, id: 'crm', title: 'CRM' },
    ],
  },
  {
    id: 'enterprise',
    title: '企业管理',
    items: [
      { available: true, icon: RiUserSettingsFill, id: 'member-state', title: '成员启用/停用' },
      {
        available: true,
        icon: RiUser3Fill,
        id: 'external-people',
        meta: '500个/对接互联组织',
        title: '互联对接人',
      },
      {
        available: true,
        icon: RiDatabase2Fill,
        id: 'external-connections',
        meta: '60个',
        title: '互联组织连接数',
      },
      { available: true, icon: RiLock2Fill, id: 'sso', title: '单点登录' },
      { available: true, icon: RiPaletteFill, id: 'login-page', title: '自定义登录页' },
      { available: true, icon: RiRecycleFill, id: 'sync', title: '成员自动同步' },
      { available: true, icon: RiShieldUserFill, id: 'reminder', title: '提醒屏蔽' },
      { available: true, icon: RiFileSettingsFill, id: 'watermark', title: '企业水印' },
      { available: true, icon: RiPaletteFill, id: 'brand', title: '企业风格' },
    ],
  },
  {
    id: 'product-management',
    requiresUpgrade: true,
    title: '产品管理',
    items: [
      { available: true, icon: RiApps2Fill, id: 'management-group', title: '应用管理组' },
      { available: true, icon: RiLock2Fill, id: 'permission-query', title: '权限查询' },
      { available: true, icon: RiBarChartBoxFill, id: 'analytics', title: '使用统计（高级版）' },
      { available: true, icon: RiLayoutGridFill, id: 'workbench', title: '自定义工作台（高级版）' },
      { available: true, icon: RiCoinsFill, id: 'payment', title: '支付服务' },
      { available: true, icon: RiRecycleFill, id: 'recycle-bin', meta: '180天', title: '回收站' },
      { available: true, icon: RiShareForwardFill, id: 'api', title: '应用搭建API和数据推送' },
      { available: false, icon: RiAttachmentFill, id: 'global-attachment', title: '全局附件管控' },
    ],
  },
  {
    id: 'open-platform',
    title: '开放平台',
    items: [
      { available: true, icon: RiApps2Fill, id: 'open-platform', title: '开放平台' },
      { available: true, icon: RiBarChartBoxFill, id: 'platform-api', title: '平台API和消息推送' },
      {
        available: false,
        icon: RiFileSettingsFill,
        id: 'resource-api',
        title: '资源用量与审计日志API',
      },
      { available: true, icon: RiPuzzle2Fill, id: 'plugins', title: '自建插件' },
    ],
  },
];

const selectedQuota = shallowRef<EditionQuotaCard | null>(null);

function notifyAction(action: Parameters<typeof handleOverviewAction>[0]) {
  const labels = {
    certificate: '上传凭证入口正在建设中',
    consumption: '正在为您打开云币消耗统计',
    consult: '顾问将尽快与您联系',
    payment: '正在为您打开支付设置',
    recharge: '正在为您打开云币充值',
    'usage-log': '正在为您打开云币使用日志',
  } as const;

  ElMessage.success(labels[action]);
}

function handleOverviewAction(
  action: 'consult' | 'certificate' | 'recharge' | 'payment' | 'consumption' | 'usage-log',
) {
  notifyAction(action);
}

function handleUpgrade() {
  ElMessage.success('已为您准备升级方案，请联系顾问完成版本升级');
}

/** 对话框关闭时只重置当前详情，容量数据仍由静态展示模型统一维护。 */
function updateQuotaDialog(visible: boolean) {
  if (!visible) {
    selectedQuota.value = null;
  }
}
</script>

<template>
  <section class="tenant-edition-page" aria-label="版本信息">
    <TenantEditionOverview
      balance="2.00"
      expiry-date="2026-08-28"
      version-name="试用版"
      @action="handleOverviewAction"
    />
    <TenantEditionQuotaGrid
      :cards="quotaCards"
      @detail="selectedQuota = $event"
      @upgrade="handleUpgrade"
    />
    <TenantEditionFeatureGroups :groups="featureGroups" @upgrade="handleUpgrade" />

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
  padding: 28px;
  background: #fff;

  &__quota-detail {
    margin: 0;

    div {
      display: grid;
      grid-template-columns: 92px minmax(0, 1fr);
      gap: 12px;
      padding: 13px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: 0;
      }
    }

    dt,
    dd {
      margin: 0;
      font-size: 14px;
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
    padding: 18px 16px 28px;
  }
}
</style>
