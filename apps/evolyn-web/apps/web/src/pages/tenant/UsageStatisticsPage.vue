<script setup lang="ts">
import type { EvolynTableColumn } from '@evolyn.do/ui';
import { EvolynTable } from '@evolyn.do/ui';
import {
  RiBarChartFill,
  RiDatabase2Fill,
  RiDownload2Fill,
  RiFileChartFill,
  RiGroupFill,
  RiInformationFill,
  RiLoginBoxFill,
  RiRefreshFill,
  RiTimeFill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { isDark } from '~/composables/dark';

defineOptions({ name: 'UsageStatisticsPage' });

type UsageTab = 'resource' | 'administrator' | 'member' | 'efficiency' | 'login';
type ResourceDimension = 'application' | 'member';
type Granularity = 'day' | 'week' | 'month';
type UsageRecord = Record<string, string | number>;

interface TrendSeries {
  label: string;
  color: 'primary' | 'success';
  values: number[];
}

interface MetricCard {
  label: string;
  value: string;
  description?: string;
  icon?: typeof RiDatabase2Fill;
}

const PAGE_SIZE = 10;
const activeTab = shallowRef<UsageTab>('resource');
const resourceDimension = shallowRef<ResourceDimension>('application');
const selectedApplications = shallowRef<string[]>([]);
const selectedMembers = shallowRef<string[]>([]);
const granularity = shallowRef<Granularity>('day');
const dateRange = shallowRef<string[]>(['2026-07-27', '2026-08-26']);
const currentPage = shallowRef(1);

const applications = ['项目协作', '合同管理', '客户管理', '人事服务', '采购管理'];
const members = ['陈同学', '李同学', '王同学', '赵同学'];
const tabOptions: Array<{ name: UsageTab; label: string }> = [
  { name: 'resource', label: '资源用量' },
  { name: 'administrator', label: '管理员活跃数据' },
  { name: 'member', label: '成员活跃数据' },
  { name: 'efficiency', label: '系统效益' },
  { name: 'login', label: '登录数据' },
];

/** 前端还原阶段使用本地展示数据；后续仅替换为统计 API 的响应。 */
const resourceRows: UsageRecord[] = [
  ['项目协作', '李同学', '2026-08-20', '2026-08-25', 386, '28.6 MB', 15, 8, 9, 4, 6, 3],
  ['合同管理', '李同学', '2026-08-14', '2026-08-24', 133, '12.4 MB', 10, 3, 2, 2, 7, 1],
  ['客户管理', '王同学', '2026-08-14', '2026-08-23', 57, '8.2 MB', 7, 4, 4, 3, 4, 2],
  ['人事服务', '陈同学', '2026-08-21', '2026-08-22', 382, '6.8 MB', 16, 6, 16, 4, 17, 5],
  ['采购管理', '赵同学', '2026-08-14', '2026-08-21', 1421, '42.1 MB', 15, 6, 16, 3, 17, 8],
].map(
  ([
    application,
    creator,
    createdAt,
    lastAccessedAt,
    records,
    storage,
    forms,
    workflows,
    dashboards,
    integrations,
    assistants,
    analyses,
  ]) => ({
    application,
    creator,
    createdAt,
    lastAccessedAt,
    records,
    storage,
    forms,
    workflows,
    dashboards,
    integrations,
    assistants,
    analyses,
  }),
);

const memberResourceRows: UsageRecord[] = [
  ['李同学', 5, 1298, 48, 37, 26, 10, 31, 12],
  ['王同学', 2, 515, 22, 11, 14, 6, 12, 5],
  ['陈同学', 1, 382, 16, 6, 16, 4, 17, 5],
  ['赵同学', 1, 1421, 15, 6, 16, 3, 17, 8],
].map(
  ([
    member,
    applications,
    records,
    forms,
    workflows,
    dashboards,
    integrations,
    assistants,
    analyses,
  ]) => ({
    member,
    applications,
    records,
    forms,
    workflows,
    dashboards,
    integrations,
    assistants,
    analyses,
  }),
);

const administratorRows: UsageRecord[] = [
  ['2026-08-25', '李同学', 1, 2],
  ['2026-08-24', '李同学', 1, 5],
  ['2026-08-23', '王同学', 1, 1],
].map(([date, member, people, count]) => ({ date, member, people, count }));

const memberRows: UsageRecord[] = [
  ['2026-08-25', '项目协作', 1, 2, 0, 0, 1, 2, 0, 0, 0, 0],
  ['2026-08-24', '合同管理', 1, 5, 1, 3, 1, 3, 0, 0, 1, 1],
  ['2026-08-23', '客户管理', 1, 1, 0, 0, 1, 2, 0, 0, 0, 0],
].map(
  ([
    date,
    application,
    visitors,
    visits,
    creators,
    creates,
    editors,
    edits,
    deleters,
    deletes,
    exporters,
    exports,
  ]) => ({
    date,
    application,
    visitors,
    visits,
    creators,
    creates,
    editors,
    edits,
    deleters,
    deletes,
    exporters,
    exports,
  }),
);

const loginRows: UsageRecord[] = [
  ['2026-08-25', '李同学', 1],
  ['2026-08-21', '李同学', 2],
  ['2026-08-20', '王同学', 2],
  ['2026-08-19', '李同学', 1],
  ['2026-08-18', '陈同学', 2],
].map(([date, member, count]) => ({ date, member, count }));

const days = Array.from({ length: 15 }, (_, index) => `08-${String(index + 12).padStart(2, '0')}`);
const trendByTab: Record<Exclude<UsageTab, 'resource' | 'efficiency'>, TrendSeries[]> = {
  administrator: [
    { label: '人数', color: 'primary', values: [0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1] },
    { label: '次数', color: 'success', values: [0, 0, 0, 0, 0, 0, 1, 0, 0, 3, 5, 2, 1, 0, 2] },
  ],
  member: [
    { label: '人数', color: 'primary', values: [0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1] },
    { label: '次数', color: 'success', values: [0, 0, 0, 0, 0, 0, 2, 1, 0, 5, 8, 4, 2, 0, 3] },
  ],
  login: [
    { label: '人数', color: 'primary', values: [0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 1] },
    { label: '次数', color: 'success', values: [0, 0, 0, 0, 0, 0, 0, 2, 1, 2, 0, 0, 0, 0, 2] },
  ],
};

const trendMax = computed(() => {
  const series =
    activeTab.value === 'resource' || activeTab.value === 'efficiency'
      ? []
      : trendByTab[activeTab.value];
  return Math.max(1, ...series.flatMap((item) => item.values));
});

const resourceCards = computed<MetricCard[]>(() => [
  { label: '应用数', value: '9', description: '当前租户内已创建应用', icon: RiDatabase2Fill },
  { label: '数据总量', value: '2,152', description: '应用内数据记录总数', icon: RiFileChartFill },
  { label: '附件上传量', value: '98.1 MB', description: '已使用的文件存储空间' },
  { label: '普通表单数', value: '63', description: '已启用表单' },
  { label: '流程表单数', value: '27', description: '已配置流程的表单' },
  { label: '仪表盘数', value: '64', description: '应用内仪表盘' },
  { label: '集成表数', value: '11', description: '可用集成配置' },
  { label: '智能助手数', value: '65', description: '已启用智能助手' },
]);

const tableColumns = computed<EvolynTableColumn[]>(() => {
  if (activeTab.value === 'resource') {
    if (resourceDimension.value === 'member') {
      return [
        { field: 'member', title: '创建者', width: 156 },
        { field: 'applications', title: '应用数', width: 110, sort: true },
        { field: 'records', title: '数据总量', width: 130, sort: true },
        { field: 'forms', title: '普通表单数', width: 130, sort: true },
        { field: 'workflows', title: '流程表单数', width: 130, sort: true },
        { field: 'dashboards', title: '仪表盘数', width: 120, sort: true },
        { field: 'integrations', title: '集成表数', width: 120, sort: true },
        { field: 'assistants', title: '智能助手数', width: 136, sort: true },
        { field: 'analyses', title: '流程分析数', width: 130, sort: true },
      ];
    }
    return [
      { field: 'application', title: '应用', width: 180 },
      { field: 'creator', title: '创建者', width: 130 },
      { field: 'createdAt', title: '创建日期', width: 130, sort: true },
      { field: 'lastAccessedAt', title: '最近一次访问时间', width: 160, sort: true },
      { field: 'records', title: '数据总量', width: 116, sort: true },
      { field: 'storage', title: '附件上传量', width: 128, sort: true },
      { field: 'forms', title: '普通表单数', width: 128, sort: true },
      { field: 'workflows', title: '流程表单数', width: 128, sort: true },
      { field: 'dashboards', title: '仪表盘数', width: 116, sort: true },
      { field: 'integrations', title: '集成表数', width: 116, sort: true },
      { field: 'assistants', title: '智能助手数', width: 128, sort: true },
      { field: 'analyses', title: '流程分析数', width: 124, sort: true },
    ];
  }
  if (activeTab.value === 'administrator') {
    return [
      { field: 'date', title: '时间', width: 180, sort: true },
      { field: 'member', title: '成员', width: 220 },
      { field: 'people', title: '编辑人数', width: 160, sort: true },
      { field: 'count', title: '编辑次数', width: 160, sort: true },
    ];
  }
  if (activeTab.value === 'member') {
    return [
      { field: 'date', title: '时间', width: 130, sort: true },
      { field: 'application', title: '应用', width: 160 },
      { field: 'visitors', title: '访问人数', width: 118, sort: true },
      { field: 'visits', title: '访问次数', width: 118, sort: true },
      { field: 'creators', title: '创建数据人数', width: 138, sort: true },
      { field: 'creates', title: '创建数据次数', width: 138, sort: true },
      { field: 'editors', title: '修改数据人数', width: 138, sort: true },
      { field: 'edits', title: '修改数据次数', width: 138, sort: true },
      { field: 'deleters', title: '删除数据人数', width: 138, sort: true },
      { field: 'deletes', title: '删除数据次数', width: 138, sort: true },
      { field: 'exporters', title: '导出数据人数', width: 138, sort: true },
      { field: 'exports', title: '导出数据次数', width: 138, sort: true },
    ];
  }
  return [
    { field: 'date', title: '时间', width: 190, sort: true },
    { field: 'member', title: '成员', width: 250 },
    { field: 'count', title: '登录次数', width: 160, sort: true },
  ];
});

const filteredRows = computed<UsageRecord[]>(() => {
  const rows =
    activeTab.value === 'resource'
      ? resourceDimension.value === 'application'
        ? resourceRows
        : memberResourceRows
      : activeTab.value === 'administrator'
        ? administratorRows
        : activeTab.value === 'member'
          ? memberRows
          : loginRows;
  const applicationFilter = selectedApplications.value;
  const memberFilter = selectedMembers.value;
  return rows.filter((row) => {
    const matchesApplication =
      applicationFilter.length === 0 ||
      !('application' in row) ||
      applicationFilter.includes(String(row.application));
    const member = String(row.member ?? row.creator ?? '');
    return matchesApplication && (memberFilter.length === 0 || memberFilter.includes(member));
  });
});

const pageRows = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE;
  return filteredRows.value.slice(start, start + PAGE_SIZE);
});

const showsActivityFilters = computed(() =>
  ['administrator', 'member', 'login'].includes(activeTab.value),
);
const sectionTitle = computed(() => {
  if (activeTab.value === 'administrator') return '编辑人数&次数';
  if (activeTab.value === 'member') return '成员活跃趋势';
  if (activeTab.value === 'login') return '登录人数&次数';
  return '';
});

function switchTab(name: string | number) {
  activeTab.value = name as UsageTab;
  currentPage.value = 1;
}

function refreshData() {
  currentPage.value = 1;
  ElMessage.success('已按当前筛选条件刷新展示数据');
}

function exportData() {
  const headers = tableColumns.value.map((column) => String(column.title));
  const fields = tableColumns.value.map((column) => String(column.field));
  const quote = (value: unknown) => `"${String(value ?? '').replaceAll('"', '""')}"`;
  const content = [headers, ...filteredRows.value.map((row) => fields.map((field) => row[field]))]
    .map((row) => row.map(quote).join(','))
    .join('\n');
  const blob = new Blob([`\uFEFF${content}`], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `使用统计-${tabOptions.find((item) => item.name === activeTab.value)?.label}.csv`;
  anchor.click();
  URL.revokeObjectURL(url);
  ElMessage.success(`已导出 ${filteredRows.value.length} 条展示数据`);
}

function chartPoints(values: number[]) {
  const max = trendMax.value;
  return values
    .map((value, index) => {
      const x = values.length === 1 ? 0 : (index / (values.length - 1)) * 100;
      const y = 100 - (value / max) * 86 - 7;
      return `${x},${y}`;
    })
    .join(' ');
}
</script>

<template>
  <section class="usage-statistics-page" aria-label="使用统计">
    <el-tabs v-model="activeTab" class="usage-statistics-page__tabs" @tab-change="switchTab">
      <el-tab-pane v-for="tab in tabOptions" :key="tab.name" :label="tab.label" :name="tab.name" />
    </el-tabs>

    <div class="usage-statistics-page__body">
      <header class="usage-statistics-page__filters">
        <template v-if="activeTab === 'resource'">
          <label class="usage-statistics-page__filter">
            <span>统计维度</span>
            <el-select v-model="resourceDimension" class="usage-statistics-page__dimension-select">
              <el-option label="按应用查看" value="application" />
              <el-option label="按成员查看" value="member" />
            </el-select>
          </label>
          <el-select
            v-if="resourceDimension === 'application'"
            v-model="selectedApplications"
            class="usage-statistics-page__selector"
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="选择应用"
          >
            <el-option
              v-for="application in applications"
              :key="application"
              :label="application"
              :value="application"
            />
          </el-select>
          <el-select
            v-else
            v-model="selectedMembers"
            class="usage-statistics-page__selector"
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="选择成员"
          >
            <el-option v-for="member in members" :key="member" :label="member" :value="member" />
          </el-select>
        </template>

        <template v-else-if="showsActivityFilters">
          <label v-if="activeTab !== 'login'" class="usage-statistics-page__filter">
            <span>统计维度</span>
            <el-select v-model="resourceDimension" class="usage-statistics-page__dimension-select">
              <el-option label="按应用查看" value="application" />
              <el-option label="按成员查看" value="member" />
            </el-select>
          </label>
          <el-select
            v-if="activeTab !== 'login'"
            v-model="selectedApplications"
            class="usage-statistics-page__selector"
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="选择应用"
          >
            <el-option
              v-for="application in applications"
              :key="application"
              :label="application"
              :value="application"
            />
          </el-select>
          <el-select
            v-else
            v-model="selectedMembers"
            class="usage-statistics-page__selector"
            filterable
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="选择成员"
          >
            <el-option v-for="member in members" :key="member" :label="member" :value="member" />
          </el-select>
          <span class="usage-statistics-page__time-label">时间筛选</span>
          <el-radio-group v-model="granularity" class="usage-statistics-page__granularity">
            <el-radio-button label="day"> 日 </el-radio-button>
            <el-radio-button label="week"> 周 </el-radio-button>
            <el-radio-button label="month"> 月 </el-radio-button>
          </el-radio-group>
          <el-date-picker
            v-model="dateRange"
            class="usage-statistics-page__date-range"
            type="daterange"
            value-format="YYYY-MM-DD"
            range-separator="~"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
          />
        </template>

        <label v-else class="usage-statistics-page__filter">
          <span>统计范围</span>
          <el-select class="usage-statistics-page__dimension-select" model-value="累计">
            <el-option label="累计" value="累计" />
          </el-select>
        </label>

        <div class="usage-statistics-page__filter-actions">
          <el-button v-if="activeTab === 'resource'" type="primary" @click="refreshData">
            <RiRefreshFill />刷新
          </el-button>
          <el-button plain type="primary" @click="exportData"> <RiDownload2Fill />导出 </el-button>
        </div>
      </header>

      <template v-if="activeTab === 'resource'">
        <section class="usage-statistics-page__overview" aria-labelledby="usage-resource-overview">
          <div class="usage-statistics-page__section-heading">
            <h2 id="usage-resource-overview">概览</h2>
            <span>当前租户中各类资源的期末存量</span>
          </div>
          <div class="usage-statistics-page__resource-overview">
            <article
              v-for="(card, index) in resourceCards"
              :key="card.label"
              class="usage-statistics-page__metric-card"
              :class="{ 'usage-statistics-page__metric-card--primary': index < 2 }"
            >
              <component
                :is="card.icon"
                v-if="card.icon"
                class="usage-statistics-page__metric-icon"
                aria-hidden="true"
              />
              <span>{{ card.label }}</span>
              <strong>{{ card.value }}</strong>
              <small>{{ card.description }}</small>
            </article>
          </div>
        </section>
      </template>

      <template v-else-if="activeTab === 'efficiency'">
        <section class="usage-statistics-page__efficiency" aria-label="系统效益概览">
          <article
            class="usage-statistics-page__efficiency-card usage-statistics-page__efficiency-card--time"
          >
            <RiTimeFill aria-hidden="true" />
            <span>节省时间</span><strong>532 <em>小时</em></strong>
          </article>
          <article
            class="usage-statistics-page__efficiency-card usage-statistics-page__efficiency-card--cost"
          >
            <RiBarChartFill aria-hidden="true" />
            <span>节省成本</span><strong>30,339 <em>元</em></strong>
          </article>
          <article class="usage-statistics-page__efficiency-notice">
            <h2>统计说明</h2>
            <p>
              统计数据从 2026-08-20
              开始累计。节省时间以预设业务动作的平均处理时长计算，节省成本按当前租户配置的人工成本折算。
            </p>
          </article>
        </section>
      </template>

      <template v-else>
        <section
          v-if="activeTab === 'login'"
          class="usage-statistics-page__login-overview"
          aria-label="登录概览"
        >
          <article>
            <RiGroupFill aria-hidden="true" /><span>通讯录人数</span><strong>18</strong>
          </article>
          <article>
            <RiLoginBoxFill aria-hidden="true" /><span>本月登录人数</span><strong>12</strong
            ><small>较上月 +2</small>
          </article>
          <article>
            <RiBarChartFill aria-hidden="true" /><span>登录系统比例</span><strong>66.7%</strong>
          </article>
        </section>
        <section class="usage-statistics-page__trend" :aria-label="sectionTitle">
          <div class="usage-statistics-page__section-heading">
            <h2>{{ sectionTitle }}</h2>
            <span>可查看最近一年数据</span>
          </div>
          <div class="usage-statistics-page__chart">
            <div class="usage-statistics-page__chart-grid" aria-hidden="true">
              <i v-for="line in 4" :key="line" />
            </div>
            <svg
              viewBox="0 0 100 100"
              preserveAspectRatio="none"
              role="img"
              :aria-label="`${sectionTitle}趋势图`"
            >
              <polyline
                v-for="series in trendByTab[
                  activeTab as Exclude<UsageTab, 'resource' | 'efficiency'>
                ]"
                :key="series.label"
                :points="chartPoints(series.values)"
                :class="`usage-statistics-page__line usage-statistics-page__line--${series.color}`"
              />
            </svg>
            <div class="usage-statistics-page__chart-labels">
              <span v-for="day in days" :key="day">{{ day }}</span>
            </div>
            <div class="usage-statistics-page__legend">
              <span
                v-for="series in trendByTab[
                  activeTab as Exclude<UsageTab, 'resource' | 'efficiency'>
                ]"
                :key="series.label"
                ><i :class="`usage-statistics-page__legend-dot--${series.color}`" />{{
                  series.label
                }}</span
              >
            </div>
          </div>
        </section>
      </template>

      <section
        v-if="activeTab !== 'efficiency'"
        class="usage-statistics-page__details"
        aria-label="统计明细"
      >
        <div class="usage-statistics-page__section-heading">
          <h2>
            {{
              activeTab === 'resource'
                ? '用量详情'
                : activeTab === 'login'
                  ? '登录详情'
                  : '活跃详情'
            }}
          </h2>
          <el-tooltip content="当前为前端展示数据，后续将接入统计接口" placement="top">
            <RiInformationFill aria-label="数据说明" />
          </el-tooltip>
          <span>{{
            activeTab === 'resource' ? '截至 2026-08-26 的最新数据' : '可查看最近一年数据'
          }}</span>
        </div>
        <div class="usage-statistics-page__table-area">
          <el-scrollbar class="usage-statistics-page__table-scrollbar">
            <EvolynTable
              class="usage-statistics-page__table"
              :columns="tableColumns"
              :records="pageRows"
              :options="{ defaultHeaderRowHeight: 44, defaultRowHeight: 52 }"
              :theme="isDark ? 'dark' : 'light'"
              empty-text="暂无符合筛选条件的数据"
            />
          </el-scrollbar>
        </div>
        <footer class="usage-statistics-page__pagination">
          <span>共 {{ filteredRows.length }} 条</span>
          <el-pagination
            v-model:current-page="currentPage"
            :page-size="PAGE_SIZE"
            :total="filteredRows.length"
            layout="prev, pager, next"
          />
        </footer>
      </section>
    </div>
  </section>
</template>

<style scoped lang="scss">
.usage-statistics-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--el-bg-color);
}
.usage-statistics-page__tabs {
  flex: 0 0 auto;
  padding: 0 var(--el-space-2xl);
  background: var(--el-fill-color-light);
}
.usage-statistics-page__tabs :deep(.el-tabs__header) {
  margin: 0;
}
.usage-statistics-page__tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}
.usage-statistics-page__tabs :deep(.el-tabs__item) {
  height: 44px;
  padding: 0 var(--el-space-3xl);
  color: var(--el-text-color-regular);
  font-size: var(--el-font-size-base);
}
.usage-statistics-page__tabs :deep(.el-tabs__item.is-active) {
  color: var(--el-color-primary);
  font-weight: 600;
}
.usage-statistics-page__tabs :deep(.el-tabs__active-bar) {
  height: 3px;
}
.usage-statistics-page__tabs :deep(.el-tabs__content) {
  display: none;
}
.usage-statistics-page__body {
  display: flex;
  min-height: 0;
  padding: var(--el-space-3xl);
  flex: 1;
  flex-direction: column;
  gap: var(--el-space-3xl);
}
.usage-statistics-page__filters {
  display: flex;
  min-height: 32px;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--el-space-lg);
}
.usage-statistics-page__filter {
  display: inline-flex;
  align-items: center;
  gap: var(--el-space-md);
  color: var(--el-text-color-regular);
  font-size: var(--el-font-size-base);
  white-space: nowrap;
}
.usage-statistics-page__dimension-select {
  width: 128px;
}
.usage-statistics-page__selector {
  width: 288px;
}
.usage-statistics-page__time-label {
  margin-left: var(--el-space-lg);
  color: var(--el-text-color-regular);
  font-size: var(--el-font-size-base);
}
.usage-statistics-page__granularity :deep(.el-radio-button__inner) {
  min-width: 36px;
  padding: var(--el-space-md) var(--el-space-lg);
}
.usage-statistics-page__date-range {
  width: 280px;
}
.usage-statistics-page__filter-actions {
  display: flex;
  margin-left: auto;
  gap: var(--el-space-lg);
}
.usage-statistics-page__filter-actions :deep(.el-button svg) {
  width: 16px;
  height: 16px;
}
.usage-statistics-page__section-heading {
  display: flex;
  min-height: 22px;
  align-items: center;
  gap: var(--el-space-md);
}
.usage-statistics-page__section-heading h2 {
  position: relative;
  margin: 0;
  padding-left: var(--el-space-lg);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
  line-height: 22px;
}
.usage-statistics-page__section-heading h2::before {
  position: absolute;
  top: var(--el-space-xs);
  bottom: var(--el-space-xs);
  left: 0;
  width: 3px;
  border-radius: var(--el-border-radius-small);
  background: var(--el-color-primary);
  content: '';
}
.usage-statistics-page__section-heading span,
.usage-statistics-page__section-heading svg {
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-small);
}
.usage-statistics-page__section-heading svg {
  width: 16px;
  height: 16px;
}
.usage-statistics-page__overview {
  display: grid;
  gap: var(--el-space-xl);
}
.usage-statistics-page__resource-overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-medium);
  overflow: hidden;
}
.usage-statistics-page__metric-card {
  position: relative;
  display: grid;
  min-height: 94px;
  padding: var(--el-space-xl) var(--el-space-2xl);
  border-right: 1px solid var(--el-border-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
  gap: var(--el-space-xs);
}
.usage-statistics-page__metric-card:nth-child(4n) {
  border-right: 0;
}
.usage-statistics-page__metric-card:nth-last-child(-n + 4) {
  border-bottom: 0;
}
.usage-statistics-page__metric-card span,
.usage-statistics-page__metric-card small {
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-extra-small);
}
.usage-statistics-page__metric-card strong {
  color: var(--el-text-color-primary);
  font-family: Georgia, serif;
  font-size: var(--el-font-size-large);
  line-height: 24px;
}
.usage-statistics-page__metric-icon {
  position: absolute;
  top: var(--el-space-xl);
  right: var(--el-space-xl);
  width: 22px;
  height: 22px;
  color: var(--el-color-primary);
}
.usage-statistics-page__efficiency {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--el-space-xl);
}
.usage-statistics-page__efficiency-card {
  position: relative;
  display: grid;
  min-height: 78px;
  padding: var(--el-space-2xl);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-medium);
  gap: var(--el-space-xs);
}
.usage-statistics-page__efficiency-card svg {
  position: absolute;
  top: var(--el-space-2xl);
  right: var(--el-space-2xl);
  width: 34px;
  height: 34px;
  padding: var(--el-space-md);
  border-radius: var(--el-border-radius-circle);
  background: var(--el-bg-color);
  box-shadow: var(--el-box-shadow-lighter);
}
.usage-statistics-page__efficiency-card--time {
  background: var(--el-color-primary-light-9);
}
.usage-statistics-page__efficiency-card--time svg {
  color: var(--el-color-primary);
}
.usage-statistics-page__efficiency-card--cost {
  background: var(--el-color-warning-light-9);
}
.usage-statistics-page__efficiency-card--cost svg {
  color: var(--el-color-warning);
}
.usage-statistics-page__efficiency-card span {
  color: var(--el-text-color-regular);
  font-size: var(--el-font-size-base);
}
.usage-statistics-page__efficiency-card strong {
  color: var(--el-text-color-primary);
  font-family: Georgia, serif;
  font-size: 28px;
}
.usage-statistics-page__efficiency-card em {
  font-family: inherit;
  font-size: var(--el-font-size-base);
  font-style: normal;
  font-weight: 400;
}
.usage-statistics-page__efficiency-notice {
  grid-column: 1 / -1;
  padding: var(--el-space-2xl);
  border-radius: var(--el-border-radius-medium);
  background: var(--el-fill-color-light);
}
.usage-statistics-page__efficiency-notice h2 {
  margin: 0 0 var(--el-space-md);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
}
.usage-statistics-page__efficiency-notice p {
  margin: 0;
  color: var(--el-text-color-regular);
  font-size: var(--el-font-size-base);
  line-height: 24px;
}
.usage-statistics-page__login-overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--el-space-xl);
}
.usage-statistics-page__login-overview article {
  position: relative;
  display: grid;
  min-height: 70px;
  padding: var(--el-space-xl) var(--el-space-2xl);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-medium);
  gap: var(--el-space-xs);
}
.usage-statistics-page__login-overview svg {
  position: absolute;
  top: var(--el-space-2xl);
  right: var(--el-space-2xl);
  width: 22px;
  height: 22px;
  padding: var(--el-space-md);
  border-radius: var(--el-border-radius-circle);
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.usage-statistics-page__login-overview span,
.usage-statistics-page__login-overview small {
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-extra-small);
}
.usage-statistics-page__login-overview strong {
  color: var(--el-text-color-primary);
  font-family: Georgia, serif;
  font-size: 26px;
}
.usage-statistics-page__trend {
  display: grid;
  gap: var(--el-space-xl);
}
.usage-statistics-page__chart {
  position: relative;
  height: 230px;
  padding: var(--el-space-lg) 0 0;
}
.usage-statistics-page__chart-grid {
  position: absolute;
  right: 0;
  bottom: 40px;
  left: 0;
  display: grid;
  height: 154px;
  grid-template-rows: repeat(4, 1fr);
}
.usage-statistics-page__chart-grid i {
  border-top: 1px dashed var(--el-border-color-lighter);
}
.usage-statistics-page__chart svg {
  position: absolute;
  right: 0;
  bottom: 40px;
  left: 0;
  width: 100%;
  height: 154px;
  overflow: visible;
}
.usage-statistics-page__line {
  fill: none;
  stroke-width: 1.5;
  vector-effect: non-scaling-stroke;
}
.usage-statistics-page__line--primary {
  stroke: var(--el-color-primary);
}
.usage-statistics-page__line--success {
  stroke: var(--el-color-success);
}
.usage-statistics-page__chart-labels {
  position: absolute;
  right: 0;
  bottom: 18px;
  left: 0;
  display: flex;
  justify-content: space-between;
  color: var(--el-text-color-secondary);
  font-size: 10px;
  transform: rotate(-24deg);
  transform-origin: center;
}
.usage-statistics-page__legend {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  justify-content: center;
  gap: var(--el-space-xl);
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-extra-small);
}
.usage-statistics-page__legend span {
  display: inline-flex;
  align-items: center;
  gap: var(--el-space-xs);
}
.usage-statistics-page__legend i {
  width: 8px;
  height: 2px;
}
.usage-statistics-page__legend-dot--primary {
  background: var(--el-color-primary);
}
.usage-statistics-page__legend-dot--success {
  background: var(--el-color-success);
}
.usage-statistics-page__details {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: var(--el-space-xl);
}
.usage-statistics-page__table-area,
.usage-statistics-page__table-scrollbar {
  min-height: 0;
  flex: 1;
}
.usage-statistics-page__table-scrollbar :deep(.el-scrollbar__view) {
  display: flex;
  min-height: 100%;
  flex-direction: column;
}
.usage-statistics-page__table {
  min-height: 230px;
  flex: 1;
}
.usage-statistics-page__pagination {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-small);
}
@media (max-width: 1280px) {
  .usage-statistics-page__resource-overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .usage-statistics-page__metric-card:nth-child(2n) {
    border-right: 0;
  }
  .usage-statistics-page__metric-card:nth-child(4n) {
    border-right: 1px solid var(--el-border-color-lighter);
  }
  .usage-statistics-page__metric-card:nth-last-child(-n + 4) {
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .usage-statistics-page__metric-card:nth-last-child(-n + 2) {
    border-bottom: 0;
  }
}
@media (max-width: 760px) {
  .usage-statistics-page__body {
    padding: var(--el-space-xl);
  }
  .usage-statistics-page__filters {
    align-items: flex-start;
    flex-direction: column;
  }
  .usage-statistics-page__filter-actions {
    margin-left: 0;
  }
  .usage-statistics-page__selector,
  .usage-statistics-page__date-range {
    width: 100%;
  }
  .usage-statistics-page__resource-overview,
  .usage-statistics-page__login-overview,
  .usage-statistics-page__efficiency {
    grid-template-columns: 1fr;
  }
  .usage-statistics-page__metric-card {
    border-right: 0 !important;
    border-bottom: 1px solid var(--el-border-color-lighter) !important;
  }
  .usage-statistics-page__metric-card:last-child {
    border-bottom: 0 !important;
  }
  .usage-statistics-page__efficiency-notice {
    grid-column: auto;
  }
  .usage-statistics-page__table {
    min-width: 1120px;
  }
}
</style>
