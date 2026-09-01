<script setup lang="ts">
import {
  EvolynTable,
  type EvolynTableColumn,
  type EvolynTableCustomRenderElement,
  type EvolynTableCustomRenderObj,
} from '@evolyn.do/ui';
import { RiDownload2Fill, RiInformationFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, onMounted, reactive, shallowRef, useTemplateRef, watch } from 'vue';
import {
  createProductLogExport,
  downloadProductLogExport,
  listProductLogOptions,
  listProductLogs,
  type ProductApplicationOption,
  type ProductCategoryOption,
  type ProductExportTaskView,
  type ProductLogFilterQuery,
  type ProductMemberOption,
} from '~/api/productLog';
import { isDark } from '~/composables/dark';

defineOptions({ name: 'TenantProductLogsPage' });

/** 表格行（服务端受控投影；所属应用/对象为写时快照，空值渲染「—」） */
interface ProductLogRecord {
  id: number;
  operator: string;
  operatedAt: string;
  category: string;
  operationType: string;
  application: string;
  target: string;
  detail: string;
  ip: string;
}

/** 筛选草稿（查询点击后才生效，修改不自动请求） */
interface ProductLogFilters {
  categoryCode: string;
  eventCode: string;
  memberId?: number;
  startDate: string;
  endDate: string;
  applicationOrTarget: string;
}

const PAGE_SIZE = 12;
const ROW_HEIGHT = 56;
const CELL_HORIZONTAL_PADDING = 12;

// ---- 筛选项（分类/操作类型/操作人/应用均由 options 接口下发，不硬编码） ----
const categoryOptions = shallowRef<ProductCategoryOption[]>([]);
const memberOptions = shallowRef<ProductMemberOption[]>([]);
const applicationOptions = shallowRef<ProductApplicationOption[]>([]);

/** 操作类型清单：选中日志范围时收敛到该范围事件，未选时展示全部 */
const eventOptions = computed(() => {
  if (!filters.categoryCode) {
    return categoryOptions.value.flatMap((category) => category.events);
  }
  return (
    categoryOptions.value.find((category) => category.code === filters.categoryCode)?.events ?? []
  );
});

const filters = reactive<ProductLogFilters>({
  categoryCode: '',
  eventCode: '',
  memberId: undefined,
  startDate: '',
  endDate: '',
  applicationOrTarget: '',
});
/** 已生效筛选快照（查询按钮触发刷新；翻页沿用） */
const appliedFilters = shallowRef<ProductLogFilterQuery>({});
const currentPage = shallowRef(1);
const total = shallowRef(0);
const pageRecords = shallowRef<ProductLogRecord[]>([]);
const tableRoot = useTemplateRef<HTMLElement>('tableRoot');

/** VTable 是画布渲染，通过实际级联主题值生成首列文字头像。 */
const canvasTokens = computed(() => {
  const style = tableRoot.value ? getComputedStyle(tableRoot.value) : null;
  const read = (name: string, fallback: string) => style?.getPropertyValue(name).trim() || fallback;
  return {
    primary: read('--el-color-primary', '#1677ff'),
    text: read('--el-text-color-regular', '#606266'),
  };
});

/** VTable 的第 0 行是表头，富单元格行号须换算为当前页记录下标。 */
function recordAt(row: number): ProductLogRecord | undefined {
  const recordIndex = row - 1;
  return recordIndex >= 0 ? pageRecords.value[recordIndex] : undefined;
}

const columns = computed<EvolynTableColumn[]>(() => {
  const avatar: EvolynTableCustomRenderElement = {
    type: 'circle',
    x: CELL_HORIZONTAL_PADDING + 11,
    y: ROW_HEIGHT / 2,
    radius: 11,
    fill: canvasTokens.value.primary,
  };
  return [
    {
      field: 'operator',
      title: '操作人',
      width: 190,
      customRender: ({ row }): EvolynTableCustomRenderObj => {
        const operator = recordAt(row)?.operator || '';
        return {
          expectedWidth: 190,
          expectedHeight: ROW_HEIGHT,
          elements: [
            avatar,
            {
              type: 'text',
              x: CELL_HORIZONTAL_PADDING + 11,
              y: ROW_HEIGHT / 2,
              text: operator.charAt(0),
              fill: '#fff',
              fontSize: 11,
              textAlign: 'center',
              textBaseline: 'middle',
            },
            {
              type: 'text',
              x: CELL_HORIZONTAL_PADDING + 32,
              y: ROW_HEIGHT / 2,
              text: operator,
              fill: canvasTokens.value.text,
              fontSize: 14,
              textBaseline: 'middle',
            },
          ],
        };
      },
    },
    { field: 'operatedAt', title: '操作时间', width: 172 },
    { field: 'operationType', title: '操作类型', width: 160 },
    { field: 'application', title: '所属应用', width: 190 },
    { field: 'target', title: '操作对象', width: 180 },
    { field: 'detail', title: '操作详情', minWidth: 330 },
    { field: 'ip', title: 'IP', width: 170 },
  ];
});

const tableOptions = { defaultHeaderRowHeight: 42, defaultRowHeight: ROW_HEIGHT };

/** 应用/对象选择值 → 查询参数：命中应用名走应用 ID 精确过滤，自由输入按关键词匹配 */
function resolveApplicationOrTarget(
  value: string,
): Pick<ProductLogFilterQuery, 'applicationId' | 'keyword'> {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }
  const matched = applicationOptions.value.find((application) => application.name === trimmed);
  return matched ? { applicationId: matched.applicationId } : { keyword: trimmed };
}

/** 筛选草稿 → 查询参数（日期闭区间直传，服务端换算半开区间） */
function buildQuery(): ProductLogFilterQuery {
  return {
    categoryCode: filters.categoryCode || undefined,
    eventCode: filters.eventCode || undefined,
    memberId: filters.memberId,
    startAt: filters.startDate || undefined,
    endAt: filters.endDate || undefined,
    ...resolveApplicationOrTarget(filters.applicationOrTarget),
  };
}

/** 拉取当前页数据（查询与翻页共用；失败保持原列表并提示） */
async function fetchPage() {
  try {
    const page = await listProductLogs({
      ...appliedFilters.value,
      page: currentPage.value,
      pageSize: PAGE_SIZE,
    });
    pageRecords.value = page.items.map((item, index) => ({
      id: (currentPage.value - 1) * PAGE_SIZE + index + 1,
      operator: item.actorName,
      operatedAt: item.operatedAt,
      category: item.categoryName,
      operationType: item.eventName,
      application: item.applicationName || '—',
      target: item.targetName || '—',
      detail: item.summary,
      ip: item.ip,
    }));
    total.value = page.total;
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '产品日志查询失败');
  }
}

function queryLogs() {
  if (filters.startDate && filters.endDate && filters.startDate > filters.endDate) {
    ElMessage.warning('开始日期不能晚于结束日期');
    return;
  }
  appliedFilters.value = buildQuery();
  // 已在第 1 页时页码不变、watch 不触发，需显式拉取；否则翻回第 1 页由 watch 拉取
  if (currentPage.value === 1) {
    void fetchPage();
  } else {
    currentPage.value = 1;
  }
}

/** 翻页沿用已生效筛选；日志范围切换时联动清空操作类型草稿，避免失效事件码 */
watch(currentPage, fetchPage);
watch(
  () => filters.categoryCode,
  () => {
    filters.eventCode = '';
  },
);

/** 导出当前筛选条件下的全部已授权数据（非当前页）：创建任务 → 同步就绪即下载 */
async function exportLogs() {
  let task: ProductExportTaskView;
  try {
    task = await createProductLogExport(appliedFilters.value);
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '产品日志导出失败');
    return;
  }
  try {
    const blob = await downloadProductLogExport(task.id);
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = task.fileName || '产品日志.csv';
    anchor.click();
    URL.revokeObjectURL(url);
    ElMessage.success(`已导出 ${task.total} 条产品日志`);
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '导出文件下载失败');
  }
}

onMounted(async () => {
  // 首屏并行拉取筛选项与首页数据（筛选失败不阻塞列表）
  void fetchPage();
  try {
    const options = await listProductLogOptions();
    categoryOptions.value = options.categories;
    memberOptions.value = options.members;
    applicationOptions.value = options.applications;
  } catch {
    ElMessage.warning('筛选项加载失败，请刷新重试');
  }
});
</script>

<template>
  <section ref="tableRoot" class="product-logs-page" aria-label="产品日志">
    <section class="product-logs-page__filters" aria-label="产品日志筛选">
      <label class="product-logs-page__filter">
        <span>日志范围</span>
        <el-select v-model="filters.categoryCode">
          <el-option label="全部" value="" />
          <el-option
            v-for="category in categoryOptions"
            :key="category.code"
            :label="category.name"
            :value="category.code"
          />
        </el-select>
      </label>

      <label class="product-logs-page__filter">
        <span>操作类型</span>
        <el-select v-model="filters.eventCode" clearable placeholder="全部">
          <el-option
            v-for="event in eventOptions"
            :key="event.code"
            :label="event.name"
            :value="event.code"
          />
        </el-select>
      </label>

      <label class="product-logs-page__filter">
        <span>操作人</span>
        <el-select v-model="filters.memberId" clearable filterable placeholder="全部">
          <el-option
            v-for="member in memberOptions"
            :key="member.memberId"
            :label="member.name"
            :value="member.memberId"
          />
        </el-select>
      </label>

      <label class="product-logs-page__filter product-logs-page__filter--date">
        <span>操作时间</span>
        <el-date-picker
          v-model="filters.startDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="开始日期"
        />
        <em>~</em>
        <el-date-picker
          v-model="filters.endDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="结束日期"
        />
      </label>

      <label class="product-logs-page__filter product-logs-page__filter--application">
        <span>应用/对象</span>
        <el-select
          v-model="filters.applicationOrTarget"
          clearable
          filterable
          allow-create
          default-first-option
          placeholder="搜索并选择应用/对象"
        >
          <el-option
            v-for="application in applicationOptions"
            :key="application.applicationId"
            :label="application.name"
            :value="application.name"
          />
        </el-select>
      </label>
    </section>

    <div class="product-logs-page__toolbar">
      <el-tooltip content="日志仅展示当前租户内、您有权限查看的产品操作记录" placement="top">
        <RiInformationFill class="product-logs-page__hint" aria-label="日志说明" />
      </el-tooltip>
      <div class="product-logs-page__toolbar-actions">
        <el-button type="primary" @click="queryLogs">查询</el-button>
        <el-button plain type="primary" @click="exportLogs"><RiDownload2Fill />导出</el-button>
      </div>
    </div>

    <div class="product-logs-page__table-area">
      <el-scrollbar class="product-logs-page__scrollbar">
        <EvolynTable
          class="product-logs-page__table"
          :columns="columns"
          :records="pageRecords"
          :options="tableOptions"
          :theme="isDark ? 'dark' : 'light'"
          empty-text="暂无产品日志"
        />
      </el-scrollbar>
    </div>

    <footer class="product-logs-page__footer">
      <span>共 {{ total }} 条</span>
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="PAGE_SIZE"
        :total="total"
        layout="prev, pager, next"
      />
    </footer>
  </section>
</template>

<style scoped lang="scss">
.product-logs-page {
  display: flex;
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  padding: var(--el-space-2xl) var(--el-space-3xl) 0;
  flex-direction: column;

  &__filters {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: var(--el-space-lg) var(--el-space-4xl);
  }

  &__filter {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    white-space: nowrap;

    .el-select,
    .el-input {
      min-width: 0;
      flex: 1;
    }

    .el-date-editor {
      min-width: 0;
      flex: 1;
    }

    em {
      color: var(--el-text-color-secondary);
      font-style: normal;
    }
  }

  &__filter--date {
    grid-column: span 2;
    gap: var(--el-space-sm);
  }

  &__toolbar {
    display: flex;
    min-height: 58px;
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
    min-height: 420px;
    flex: 1;
  }

  &__footer {
    display: flex;
    min-height: 58px;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }
}

@media (max-width: 1280px) {
  .product-logs-page {
    &__filters {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    &__filter--date {
      grid-column: span 1;
    }
  }
}

@media (max-width: 760px) {
  .product-logs-page {
    padding: var(--el-space-xl) var(--el-space-lg) 0;

    &__filters {
      grid-template-columns: minmax(0, 1fr);
    }

    &__filter,
    &__filter--date {
      width: 100%;
      grid-column: span 1;
    }

    &__table {
      min-width: 1180px;
    }
  }
}
</style>
