<script setup lang="ts">
import { EvolynTable, type EvolynTableColumn, type EvolynTableRow } from '@evolyn.do/ui';
import { RiDownload2Fill, RiInformationFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, onMounted, shallowRef, watch } from 'vue';
import { listMembers } from '~/api/member';
import {
  createEnterpriseLogExport,
  downloadEnterpriseLogExport,
  listEnterpriseLoginLogs,
  listEnterpriseOperationLogs,
  listOperationCategories,
  type EnterpriseLoginLogItem,
  type EnterpriseLogFilterQuery,
  type EnterpriseOperationLogItem,
  type OperationCategoryOption,
} from '~/api/enterpriseLog';
import { isDark } from '~/composables/dark';

defineOptions({ name: 'TenantEnterpriseLogsPage' });

type LogTab = 'login' | 'operation';

/** 登录平台展示文案（后端 client 枚举映射，未知值原样透出） */
const CLIENT_LABELS: Record<string, string> = {
  web: '电脑网页版',
  wap: '手机网页版',
  unknown: '未知',
};

/** 表格行（服务端受控投影 + 登录平台文案映射） */
interface LoginRow {
  operator: string;
  loggedAt: string;
  location: string;
  platform: string;
  ip: string;
}

interface OperationRow {
  operator: string;
  operatedAt: string;
  type: string;
  detail: string;
  ip: string;
}

const PAGE_SIZE = 10;
const activeTab = shallowRef<LogTab>('login');

// ---- 成员远程搜索（登录人/操作人筛选共用形态，各自独立状态） ----
interface MemberOption {
  id: number;
  name: string;
}

function createMemberPicker() {
  const options = shallowRef<MemberOption[]>([]);
  const loading = shallowRef(false);
  async function search(keyword: string) {
    loading.value = true;
    try {
      const page = await listMembers({ keyword: keyword.trim(), page: 1, pageSize: 20 });
      options.value = page.items.map((member) => ({ id: member.id, name: member.name }));
    } catch {
      options.value = [];
    } finally {
      loading.value = false;
    }
  }
  return { options, loading, search };
}

// ---- 页签各自独立的筛选草稿、生效快照、页码与列表状态（互不污染） ----
const loginMemberPicker = createMemberPicker();
const loginMemberId = shallowRef<number>();
const loginStartDate = shallowRef('');
const loginEndDate = shallowRef('');
const appliedLoginFilters = shallowRef<EnterpriseLogFilterQuery>({});
const loginPage = shallowRef(1);
const loginLoading = shallowRef(false);
const loginRows = shallowRef<LoginRow[]>([]);
const loginTotal = shallowRef(0);

const categoryOptions = shallowRef<OperationCategoryOption[]>([]);
const operationCategory = shallowRef('');
const operationEvent = shallowRef('');
const operationMemberPicker = createMemberPicker();
const operationMemberId = shallowRef<number>();
const operationStartDate = shallowRef('');
const operationEndDate = shallowRef('');
const appliedOperationFilters = shallowRef<EnterpriseLogFilterQuery>({});
const operationPage = shallowRef(1);
const operationLoading = shallowRef(false);
const operationRows = shallowRef<OperationRow[]>([]);
const operationTotal = shallowRef(0);

const exporting = shallowRef(false);

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

const tableColumns = computed(() =>
  activeTab.value === 'login' ? loginColumns : operationColumns,
);
// 两个页签的行结构不同，以 EvolynTableRow 收窄后整体替换引用（表格按引用增量刷新）
const tableRecords = computed<EvolynTableRow[]>(() =>
  activeTab.value === 'login' ? loginRows.value : operationRows.value,
);
const tableTotal = computed(() =>
  activeTab.value === 'login' ? loginTotal.value : operationTotal.value,
);
const tableOptions = { defaultHeaderRowHeight: 42, defaultRowHeight: 43 };
const tableLoading = computed(() =>
  activeTab.value === 'login' ? loginLoading.value : operationLoading.value,
);

/** 操作类型选项随日志范围联动：未选范围为全部分类的事件并集 */
const eventOptions = computed(() => {
  const category = operationCategory.value;
  if (!category) return categoryOptions.value.flatMap((option) => option.events);
  return categoryOptions.value.find((option) => option.code === category)?.events ?? [];
});

/** 范围切换后事件选项集合变化，清空已选事件避免提交失效事件码 */
watch(operationCategory, () => {
  operationEvent.value = '';
});

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

/** 日期草稿校验：开始晚于结束直接拦截（与产品日志页同文案口径） */
function validateDateRange(startDate: string, endDate: string): boolean {
  if (startDate && endDate && startDate > endDate) {
    ElMessage.warning('开始日期不能晚于结束日期');
    return false;
  }
  return true;
}

function loadLoginLogs() {
  loginLoading.value = true;
  listEnterpriseLoginLogs({
    ...appliedLoginFilters.value,
    page: loginPage.value,
    pageSize: PAGE_SIZE,
  })
    .then((page: { items: EnterpriseLoginLogItem[]; total: number }) => {
      loginRows.value = page.items.map((item) => ({
        operator: item.actorName,
        loggedAt: item.loggedAt,
        location: item.location,
        platform: CLIENT_LABELS[item.client] ?? item.client,
        ip: item.ip,
      }));
      loginTotal.value = page.total;
    })
    .catch((error: unknown) => {
      loginRows.value = [];
      loginTotal.value = 0;
      ElMessage.error(errorMessage(error, '登录日志加载失败，请稍后重试'));
    })
    .finally(() => {
      loginLoading.value = false;
    });
}

function loadOperationLogs() {
  operationLoading.value = true;
  listEnterpriseOperationLogs({
    ...appliedOperationFilters.value,
    page: operationPage.value,
    pageSize: PAGE_SIZE,
  })
    .then((page: { items: EnterpriseOperationLogItem[]; total: number }) => {
      operationRows.value = page.items.map((item) => ({
        operator: item.actorName,
        operatedAt: item.operatedAt,
        type: item.eventName,
        detail: item.summary,
        ip: item.ip,
      }));
      operationTotal.value = page.total;
    })
    .catch((error: unknown) => {
      operationRows.value = [];
      operationTotal.value = 0;
      ElMessage.error(errorMessage(error, '操作日志加载失败，请稍后重试'));
    })
    .finally(() => {
      operationLoading.value = false;
    });
}

function loadCategories() {
  listOperationCategories()
    .then((options) => {
      categoryOptions.value = options;
    })
    .catch(() => {
      categoryOptions.value = [];
    });
}

function switchTab(tab: LogTab) {
  activeTab.value = tab;
}

/** 点击「查询」：快照当前页签筛选草稿并回到第一页重新拉取 */
function queryLogs() {
  if (activeTab.value === 'login') {
    if (!validateDateRange(loginStartDate.value, loginEndDate.value)) return;
    appliedLoginFilters.value = {
      memberId: loginMemberId.value,
      startAt: loginStartDate.value || undefined,
      endAt: loginEndDate.value || undefined,
    };
    if (loginPage.value === 1) {
      loadLoginLogs();
    } else {
      loginPage.value = 1;
    }
    return;
  }
  if (!validateDateRange(operationStartDate.value, operationEndDate.value)) return;
  appliedOperationFilters.value = {
    categoryCode: operationCategory.value || undefined,
    eventCode: operationEvent.value || undefined,
    memberId: operationMemberId.value,
    startAt: operationStartDate.value || undefined,
    endAt: operationEndDate.value || undefined,
  };
  if (operationPage.value === 1) {
    loadOperationLogs();
  } else {
    operationPage.value = 1;
  }
}

/** 导出当前页签当前筛选条件（已生效快照，与列表可见数据一致）下的全部授权数据 */
async function exportLogs() {
  if (exporting.value) return;
  exporting.value = true;
  try {
    const kind = activeTab.value;
    const filters = kind === 'login' ? appliedLoginFilters.value : appliedOperationFilters.value;
    const task = await createEnterpriseLogExport({ kind, ...filters });
    if (task.status !== 'ready') {
      ElMessage.warning('导出文件尚未生成完成，请稍后重试');
      return;
    }
    const blob = await downloadEnterpriseLogExport(task.id);
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = task.fileName || `企业日志-${kind === 'login' ? '登录日志' : '操作日志'}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
    ElMessage.success(`已导出 ${task.total} 条记录`);
  } catch (error) {
    ElMessage.error(errorMessage(error, '导出失败，请稍后重试'));
  } finally {
    exporting.value = false;
  }
}

// 翻页触发当前页签拉取：查询按钮已保证回到第一页，这里只响应页码变化
watch(loginPage, () => {
  if (activeTab.value === 'login') loadLoginLogs();
});
watch(operationPage, () => {
  if (activeTab.value === 'operation') loadOperationLogs();
});

onMounted(() => {
  void loginMemberPicker.search('');
  void operationMemberPicker.search('');
  loadLoginLogs();
  loadOperationLogs();
  loadCategories();
});
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
          <el-select
            v-model="loginMemberId"
            clearable
            filterable
            placeholder="请选择登录人"
            remote
            :loading="loginMemberPicker.loading.value"
            :remote-method="loginMemberPicker.search"
          >
            <el-option
              v-for="member in loginMemberPicker.options.value"
              :key="member.id"
              :label="member.name"
              :value="member.id"
            />
          </el-select>
        </label>
        <label class="enterprise-logs-page__filter enterprise-logs-page__filter--date">
          <span>登录时间</span>
          <el-date-picker
            v-model="loginStartDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="开始日期"
          />
          <em>~</em>
          <el-date-picker
            v-model="loginEndDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="结束日期"
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
          <el-select v-model="operationCategory">
            <el-option label="全部" value="" />
            <el-option
              v-for="category in categoryOptions"
              :key="category.code"
              :label="category.name"
              :value="category.code"
            />
          </el-select>
        </label>
        <label class="enterprise-logs-page__filter">
          <span>操作类型</span>
          <el-select v-model="operationEvent" clearable placeholder="全部">
            <el-option
              v-for="event in eventOptions"
              :key="event.code"
              :label="event.name"
              :value="event.code"
            />
          </el-select>
        </label>
        <label class="enterprise-logs-page__filter enterprise-logs-page__filter--member">
          <span>操作人</span>
          <el-select
            v-model="operationMemberId"
            clearable
            filterable
            placeholder="请选择操作人"
            remote
            :loading="operationMemberPicker.loading.value"
            :remote-method="operationMemberPicker.search"
          >
            <el-option
              v-for="member in operationMemberPicker.options.value"
              :key="member.id"
              :label="member.name"
              :value="member.id"
            />
          </el-select>
        </label>
        <label class="enterprise-logs-page__filter enterprise-logs-page__filter--date">
          <span>操作时间</span>
          <el-date-picker
            v-model="operationStartDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="开始日期"
          />
          <em>~</em>
          <el-date-picker
            v-model="operationEndDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="结束日期"
          />
        </label>
      </section>

      <div class="enterprise-logs-page__toolbar">
        <el-tooltip content="日志仅展示当前租户内、您有权限查看的记录" placement="top">
          <RiInformationFill class="enterprise-logs-page__hint" aria-label="日志说明" />
        </el-tooltip>
        <div class="enterprise-logs-page__toolbar-actions">
          <el-button type="primary" @click="queryLogs">查询</el-button>
          <el-button :loading="exporting" plain type="primary" @click="exportLogs">
            <RiDownload2Fill />导出
          </el-button>
        </div>
      </div>

      <div v-loading="tableLoading" class="enterprise-logs-page__table-area">
        <el-scrollbar class="enterprise-logs-page__scrollbar">
          <EvolynTable
            class="enterprise-logs-page__table"
            :columns="tableColumns"
            :records="tableRecords"
            :options="tableOptions"
            :theme="isDark ? 'dark' : 'light'"
            empty-text="暂无日志记录"
          />
        </el-scrollbar>
      </div>

      <footer class="enterprise-logs-page__footer">
        <span>共 {{ tableTotal || 0 }} 条</span>
        <el-pagination
          v-if="activeTab === 'login'"
          v-model:current-page="loginPage"
          :page-size="PAGE_SIZE"
          :total="loginTotal"
          layout="prev, pager, next"
        />
        <el-pagination
          v-else
          v-model:current-page="operationPage"
          :page-size="PAGE_SIZE"
          :total="operationTotal"
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
