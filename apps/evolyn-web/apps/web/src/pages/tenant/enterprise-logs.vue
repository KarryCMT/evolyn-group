<script setup lang="ts">
import { EvolynTable, type EvolynTableColumn } from '@evolyn.do/ui';
import { RiDownload2Fill, RiInformationFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { isDark } from '~/composables/dark';

defineOptions({ name: 'TenantEnterpriseLogsPage' });

type LogTab = 'login' | 'operation';

interface LoginRecord {
  operator: string;
  loggedAt: string;
  location: string;
  platform: string;
  ip: string;
}

interface OperationRecord {
  operator: string;
  operatedAt: string;
  category: string;
  type: string;
  detail: string;
  ip: string;
}

const PAGE_SIZE = 10;
const activeTab = shallowRef<LogTab>('login');
const currentPage = shallowRef(1);

const loginOperator = shallowRef('');
const loginStartDate = shallowRef('');
const loginEndDate = shallowRef('');
const operationCategory = shallowRef('all');
const operationType = shallowRef('');
const operationOperator = shallowRef('');
const operationStartDate = shallowRef('');
const operationEndDate = shallowRef('');

/** 前端还原阶段的展示数据；后端接口接入后替换为对应分页请求。 */
const loginRecords: LoginRecord[] = [
  ['李同学', '2026-08-25 21:50:43', '广东省 深圳市', '电脑网页版', '27.46.93.3'],
  ['李同学', '2026-08-21 13:29:07', '广东省 深圳市', '电脑网页版', '210.21.226.222'],
  ['王同学', '2026-08-21 08:52:39', '广东省 深圳市', '电脑网页版', '210.21.226.222'],
  ['李同学', '2026-08-20 22:55:31', '广东省 深圳市', '电脑网页版', '27.46.93.3'],
  ['陈同学', '2026-08-20 11:16:28', '广东省 深圳市', '手机网页版', '210.21.226.222'],
  ['李同学', '2026-08-19 22:34:40', '广东省 深圳市', '电脑网页版', '27.46.93.4'],
  ['赵同学', '2026-08-18 12:38:38', '广东省 深圳市', '电脑网页版', '27.46.93.3'],
  ['李同学', '2026-08-18 07:26:22', '广东省 深圳市', '电脑网页版', '183.238.228.138'],
  ['孙同学', '2026-08-17 16:02:18', '广东省 广州市', '电脑网页版', '27.46.93.5'],
  ['周同学', '2026-08-16 09:13:45', '广东省 深圳市', '手机网页版', '119.123.31.10'],
  ['李同学', '2026-08-15 18:26:10', '广东省 深圳市', '电脑网页版', '27.46.93.3'],
  ['王同学', '2026-08-15 08:10:29', '广东省 深圳市', '电脑网页版', '210.21.226.222'],
].map(([operator, loggedAt, location, platform, ip]) => ({
  operator,
  loggedAt,
  location,
  platform,
  ip,
}));

const operationRecords: OperationRecord[] = [
  [
    '李同学',
    '2026-08-24 23:14:35',
    '成员管理',
    '个人设置成员权限',
    '修改了个人设置项「聘用形式」的成员可见状态',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-24 23:14:35',
    '成员管理',
    '个人设置成员权限',
    '修改了个人设置项「聘用形式」的成员可改状态',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-24 23:14:34',
    '成员管理',
    '个人设置成员权限',
    '修改了个人设置项「职务」的成员可见状态',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-24 23:14:34',
    '成员管理',
    '个人设置成员权限',
    '修改了个人设置项「职务」的成员可改状态',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-24 21:55:16',
    '产品管理',
    '设置产品可用范围',
    '设置了「简道云」的可见范围',
    '27.46.93.3',
  ],
  [
    '李同学',
    '2026-08-21 13:56:24',
    '产品管理',
    '启用插件',
    '启用了插件「未命名插件」',
    '210.21.226.222',
  ],
  [
    '李同学',
    '2026-08-21 13:56:20',
    '产品管理',
    '修改插件',
    '修改了插件「未命名插件」',
    '210.21.226.222',
  ],
  [
    '李同学',
    '2026-08-21 13:46:18',
    '产品管理',
    '启用插件',
    '启用了插件「金蝶精斗云同步简道云CRM套件」',
    '210.21.226.222',
  ],
  [
    '李同学',
    '2026-08-21 13:46:15',
    '产品管理',
    '安装插件',
    '安装了插件「金蝶精斗云同步简道云CRM套件」',
    '210.21.226.222',
  ],
  ['王同学', '2026-08-21 11:23:57', '企业设置', '新增密钥', '新增了密钥', '210.21.226.222'],
  [
    '李同学',
    '2026-08-21 11:22:31',
    '产品管理',
    '启用插件',
    '启用了插件「金蝶云星辰多账套同步简道云CRM套件」',
    '210.21.226.222',
  ],
  [
    '李同学',
    '2026-08-18 22:15:59',
    '组织管理',
    '添加部门',
    '添加了部门「产品部」',
    '183.238.228.138',
  ],
].map(([operator, operatedAt, category, type, detail, ip]) => ({
  operator,
  operatedAt,
  category,
  type,
  detail,
  ip,
}));

const categoryOptions = ['全部', '成员管理', '组织管理', '产品管理', '企业设置'];
const typeOptions = [
  '个人设置成员权限',
  '设置产品可用范围',
  '启用插件',
  '修改插件',
  '安装插件',
  '新增密钥',
  '添加部门',
];

const loginColumns: EvolynTableColumn[] = [
  { field: 'operator', title: '登录人', minWidth: 150 },
  { field: 'loggedAt', title: '登录时间', minWidth: 180 },
  { field: 'location', title: '登录地', minWidth: 180 },
  { field: 'platform', title: '登录平台', minWidth: 150 },
  { field: 'ip', title: 'IP', minWidth: 160 },
];

const operationColumns: EvolynTableColumn[] = [
  { field: 'operator', title: '操作人', minWidth: 150 },
  { field: 'operatedAt', title: '操作时间', minWidth: 180 },
  { field: 'type', title: '操作类型', minWidth: 190 },
  { field: 'detail', title: '操作详情', minWidth: 420 },
  { field: 'ip', title: 'IP', minWidth: 160 },
];

const filteredRecords = computed<Array<LoginRecord | OperationRecord>>(() => {
  if (activeTab.value === 'login') {
    return loginRecords.filter((record) => {
      const date = record.loggedAt.slice(0, 10);
      return (
        record.operator.includes(loginOperator.value.trim()) &&
        (!loginStartDate.value || date >= loginStartDate.value) &&
        (!loginEndDate.value || date <= loginEndDate.value)
      );
    });
  }
  return operationRecords.filter((record) => {
    const date = record.operatedAt.slice(0, 10);
    return (
      (operationCategory.value === 'all' || record.category === operationCategory.value) &&
      (!operationType.value || record.type === operationType.value) &&
      record.operator.includes(operationOperator.value.trim()) &&
      (!operationStartDate.value || date >= operationStartDate.value) &&
      (!operationEndDate.value || date <= operationEndDate.value)
    );
  });
});

const pageRecords = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE;
  return filteredRecords.value.slice(start, start + PAGE_SIZE);
});

const tableColumns = computed(() =>
  activeTab.value === 'login' ? loginColumns : operationColumns,
);
const tableOptions = { defaultHeaderRowHeight: 42, defaultRowHeight: 43 };

function switchTab(tab: LogTab) {
  activeTab.value = tab;
  currentPage.value = 1;
}

function queryLogs() {
  currentPage.value = 1;
}

function exportLogs() {
  ElMessage.info('导出功能将在企业日志服务接入后开放');
}
</script>

<template>
  <section class="enterprise-logs-page" aria-label="企业日志">
    <nav class="enterprise-logs-page__tabs" aria-label="企业日志分类">
      <button
        class="enterprise-logs-page__tab"
        :class="{ 'enterprise-logs-page__tab--active': activeTab === 'login' }"
        type="button"
        @click="switchTab('login')"
      >
        登录日志
      </button>
      <button
        class="enterprise-logs-page__tab"
        :class="{ 'enterprise-logs-page__tab--active': activeTab === 'operation' }"
        type="button"
        @click="switchTab('operation')"
      >
        操作日志
      </button>
    </nav>

    <div class="enterprise-logs-page__body">
      <section
        v-if="activeTab === 'login'"
        class="enterprise-logs-page__filters"
        aria-label="登录日志筛选"
      >
        <label class="enterprise-logs-page__filter enterprise-logs-page__filter--member">
          <span>登录人</span>
          <el-input
            v-model="loginOperator"
            clearable
            placeholder="请输入登录人"
            @change="queryLogs"
          />
        </label>
        <label class="enterprise-logs-page__filter enterprise-logs-page__filter--date">
          <span>登录时间</span>
          <el-date-picker
            v-model="loginStartDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="开始日期"
            @change="queryLogs"
          />
          <em>~</em>
          <el-date-picker
            v-model="loginEndDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="结束日期"
            @change="queryLogs"
          />
        </label>
      </section>

      <section
        v-else
        class="enterprise-logs-page__filters enterprise-logs-page__filters--operation"
        aria-label="操作日志筛选"
      >
        <label class="enterprise-logs-page__filter">
          <span>日志范围</span>
          <el-select v-model="operationCategory" @change="queryLogs">
            <el-option
              v-for="label in categoryOptions"
              :key="label"
              :label="label"
              :value="label === '全部' ? 'all' : label"
            />
          </el-select>
        </label>
        <label class="enterprise-logs-page__filter">
          <span>操作类型</span>
          <el-select v-model="operationType" clearable placeholder="全部" @change="queryLogs">
            <el-option v-for="label in typeOptions" :key="label" :label="label" :value="label" />
          </el-select>
        </label>
        <label class="enterprise-logs-page__filter enterprise-logs-page__filter--member">
          <span>操作人</span>
          <el-input
            v-model="operationOperator"
            clearable
            placeholder="请输入操作人"
            @change="queryLogs"
          />
        </label>
        <label class="enterprise-logs-page__filter enterprise-logs-page__filter--date">
          <span>操作时间</span>
          <el-date-picker
            v-model="operationStartDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="开始日期"
            @change="queryLogs"
          />
          <em>~</em>
          <el-date-picker
            v-model="operationEndDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="结束日期"
            @change="queryLogs"
          />
        </label>
      </section>

      <div class="enterprise-logs-page__toolbar">
        <el-tooltip content="日志仅展示当前租户内、您有权限查看的记录" placement="top">
          <RiInformationFill class="enterprise-logs-page__hint" aria-label="日志说明" />
        </el-tooltip>
        <div class="enterprise-logs-page__toolbar-actions">
          <el-button v-if="activeTab === 'operation'" type="primary" @click="queryLogs"
            >查询</el-button
          >
          <el-button plain type="primary" @click="exportLogs"><RiDownload2Fill />导出</el-button>
        </div>
      </div>

      <div class="enterprise-logs-page__table-area">
        <el-scrollbar class="enterprise-logs-page__scrollbar">
          <EvolynTable
            class="enterprise-logs-page__table"
            :columns="tableColumns"
            :records="pageRecords"
            :options="tableOptions"
            :theme="isDark ? 'dark' : 'light'"
            empty-text="暂无日志记录"
          />
        </el-scrollbar>
      </div>

      <footer class="enterprise-logs-page__footer">
        <span>共 {{ filteredRecords.length }} 条</span>
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="PAGE_SIZE"
          :total="filteredRecords.length"
          layout="prev, pager, next"
        />
      </footer>
    </div>
  </section>
</template>

<style scoped lang="scss">
.enterprise-logs-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;

  &__tabs {
    display: flex;
    height: 56px;
    flex: 0 0 56px;
    padding: 0 var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: stretch;
    gap: var(--el-space-xs);
  }

  &__tab {
    min-width: 102px;
    padding: 0 var(--el-space-xl);
    border: 0;
    border-bottom: 2px solid transparent;
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: pointer;
    font: inherit;
    font-size: var(--el-font-size-base);
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      border-color 0.18s ease;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-fill-color-light);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--active {
      border-bottom-color: var(--el-color-primary);
      color: var(--el-color-primary);
      font-weight: 600;
    }
  }

  &__body {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    padding: var(--el-space-2xl) var(--el-space-3xl) 0;
  }

  &__filters {
    display: flex;
    min-height: 40px;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--el-space-xl) var(--el-space-4xl);
  }

  &__filter {
    display: flex;
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    white-space: nowrap;

    .el-select,
    .el-input {
      width: 242px;
    }

    .el-date-editor {
      width: 156px;
    }

    em {
      color: var(--el-text-color-secondary);
      font-style: normal;
    }
  }

  &__filter--member .el-input {
    width: 246px;
  }

  &__filter--date {
    gap: var(--el-space-sm);
  }

  &__toolbar {
    display: flex;
    min-height: 52px;
    align-items: center;
    justify-content: space-between;
  }

  &__hint {
    width: 18px;
    height: 18px;
    color: var(--el-text-color-secondary);
  }

  &__toolbar-actions {
    display: flex;
    align-items: center;
    gap: var(--el-space-lg);
  }

  &__toolbar-actions :deep(.el-button svg) {
    width: 16px;
    height: 16px;
  }

  &__table-area,
  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__scrollbar :deep(.el-scrollbar__view) {
    display: flex;
    min-height: 100%;
    flex-direction: column;
  }

  &__table {
    min-height: 350px;
    flex: 1;
  }

  &__footer {
    display: flex;
    min-height: 62px;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }
}

@media (max-width: 1180px) {
  .enterprise-logs-page {
    &__filter--date {
      width: 100%;
    }
  }
}

@media (max-width: 760px) {
  .enterprise-logs-page {
    &__body {
      padding: var(--el-space-xl) var(--el-space-lg) 0;
    }

    &__filters {
      align-items: flex-start;
      flex-direction: column;
      gap: var(--el-space-lg);
    }

    &__filter,
    &__filter--date {
      width: 100%;
    }

    &__filter .el-select,
    &__filter .el-input,
    &__filter--member .el-input,
    &__filter .el-date-editor {
      min-width: 0;
      flex: 1;
    }

    &__table {
      min-width: 860px;
    }
  }
}
</style>
