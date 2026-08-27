<script setup lang="ts">
import {
  EvolynTable,
  type EvolynTableColumn,
  type EvolynTableCustomRenderElement,
  type EvolynTableCustomRenderObj,
} from '@evolyn.do/ui';
import { RiDownload2Fill, RiInformationFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, reactive, shallowRef, useTemplateRef } from 'vue';
import { isDark } from '~/composables/dark';

defineOptions({ name: 'TenantProductLogsPage' });

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

interface ProductLogFilters {
  category: string;
  operationType: string;
  operator: string;
  startDate: string;
  endDate: string;
  applicationOrTarget: string;
}

const PAGE_SIZE = 12;
const ROW_HEIGHT = 56;
const CELL_HORIZONTAL_PADDING = 12;

const categoryOptions = ['全部', '应用管理', '表单管理', '流程管理', '应用权限'];
const operationTypeOptions = [
  '创建表单',
  '删除表单',
  '修改表单设计',
  '修改表单名称',
  '开启流程分析',
  '添加普通管理员组',
  '开启应用管理组',
];
const applicationOptions = ['测试应用', '简道云示例应用', 'IT项目管理'];

/** 前端还原阶段的本地展示数据；后端产品日志接口接入后替换为分页结果。 */
const records: ProductLogRecord[] = [
  [
    '李同学',
    '2026-08-25 22:44:30',
    '表单管理',
    '删除表单',
    '测试应用',
    '未命名表单',
    '删除了表单「未命名表单」',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-25 22:44:26',
    '表单管理',
    '删除表单',
    '测试应用',
    '未命名表单',
    '删除了表单「未命名表单」',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-25 22:44:23',
    '表单管理',
    '删除表单',
    '测试应用',
    '未命名表单名称',
    '删除了表单「未命名表单名称」',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-24 23:17:25',
    '应用权限',
    '添加普通管理员组',
    '—',
    '测试',
    '添加了普通管理员组「测试」',
    '183.238.228.138',
  ],
  [
    '李同学',
    '2026-08-24 12:46:29',
    '流程管理',
    '开启流程分析',
    '简道云示例应用',
    '采购申请',
    '开启了表单「采购申请」的流程分析',
    '27.46.93.3',
  ],
  [
    '李同学',
    '2026-08-24 07:22:58',
    '表单管理',
    '修改表单设计',
    '简道云示例应用',
    '订单管理',
    '修改了表单「订单管理」的表单设计',
    '183.238.228.138',
  ],
  [
    '李同学',
    '2026-08-24 07:22:41',
    '表单管理',
    '修改表单设计',
    '简道云示例应用',
    '订单管理',
    '修改了表单「订单管理」的表单设计，删除了 1 个字段',
    '183.238.228.138',
  ],
  [
    '李同学',
    '2026-08-23 23:46:19',
    '应用权限',
    '开启应用管理组',
    '—',
    '测试',
    '开启了应用管理组',
    '27.46.93.4',
  ],
  [
    '李同学',
    '2026-08-23 13:12:50',
    '表单管理',
    '修改表单设计',
    'IT项目管理',
    '1.1 项目档案',
    '修改了表单「1.1 项目档案」的表单设计，删除了 2 个字段',
    '183.238.228.138',
  ],
  [
    '李同学',
    '2026-08-23 10:37:13',
    '表单管理',
    '创建表单',
    '测试应用',
    '未命名表单',
    '创建了表单「未命名表单」',
    '183.238.228.139',
  ],
  [
    '李同学',
    '2026-08-23 10:33:32',
    '表单管理',
    '创建表单',
    '测试应用',
    '未命名表单',
    '创建了表单「未命名表单」',
    '183.238.228.138',
  ],
  [
    '李同学',
    '2026-08-23 10:13:51',
    '表单管理',
    '修改表单名称',
    '测试应用',
    '未命名的',
    '将表单「未命名的」重命名为「未命名表单」',
    '183.238.228.138',
  ],
  [
    '李同学',
    '2026-08-23 09:51:25',
    '应用权限',
    '调整表单权限',
    '测试应用',
    '未命名的',
    '修改了权限组「添加并管理本人数据」的成员范围',
    '27.46.93.3',
  ],
  [
    '李同学',
    '2026-08-23 09:46:03',
    '表单管理',
    '创建表单',
    '测试应用',
    '未命名表单',
    '创建了表单「未命名表单」',
    '183.238.228.138',
  ],
  [
    '王同学',
    '2026-08-22 16:20:18',
    '应用管理',
    '修改应用名称',
    '销售管理',
    '销售管理',
    '将应用「销售管理」重命名为「销售管理」',
    '119.123.31.10',
  ],
].map(
  ([operator, operatedAt, category, operationType, application, target, detail, ip], index) => ({
    id: index + 1,
    operator,
    operatedAt,
    category,
    operationType,
    application,
    target,
    detail,
    ip,
  }),
);

const filters = reactive<ProductLogFilters>({
  category: 'all',
  operationType: '',
  operator: '',
  startDate: '',
  endDate: '',
  applicationOrTarget: '',
});
const appliedFilters = shallowRef<ProductLogFilters>({ ...filters });
const currentPage = shallowRef(1);
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

const filteredRecords = computed(() => {
  const query = appliedFilters.value;
  const keyword = query.applicationOrTarget.trim();
  const operator = query.operator.trim();
  return records.filter((record) => {
    const date = record.operatedAt.slice(0, 10);
    const matchesKeyword =
      !keyword ||
      record.application.includes(keyword) ||
      record.target.includes(keyword) ||
      record.detail.includes(keyword);
    return (
      (query.category === 'all' || record.category === query.category) &&
      (!query.operationType || record.operationType === query.operationType) &&
      (!operator || record.operator.includes(operator)) &&
      (!query.startDate || date >= query.startDate) &&
      (!query.endDate || date <= query.endDate) &&
      matchesKeyword
    );
  });
});

const pageRecords = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE;
  return filteredRecords.value.slice(start, start + PAGE_SIZE);
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

function queryLogs() {
  if (filters.startDate && filters.endDate && filters.startDate > filters.endDate) {
    ElMessage.warning('开始日期不能晚于结束日期');
    return;
  }
  appliedFilters.value = { ...filters };
  currentPage.value = 1;
}

function exportLogs() {
  const header = ['操作人', '操作时间', '操作类型', '所属应用', '操作对象', '操作详情', 'IP'];
  const rows = filteredRecords.value.map((record) => [
    record.operator,
    record.operatedAt,
    record.operationType,
    record.application,
    record.target,
    record.detail,
    record.ip,
  ]);
  const escape = (value: string) => `"${value.replaceAll('"', '""')}"`;
  const csv = [header, ...rows].map((row) => row.map(escape).join(',')).join('\n');
  const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = '产品日志.csv';
  anchor.click();
  URL.revokeObjectURL(url);
  ElMessage.success(`已导出 ${rows.length} 条产品日志`);
}
</script>

<template>
  <section ref="tableRoot" class="product-logs-page" aria-label="产品日志">
    <section class="product-logs-page__filters" aria-label="产品日志筛选">
      <label class="product-logs-page__filter">
        <span>日志范围</span>
        <el-select v-model="filters.category">
          <el-option
            v-for="category in categoryOptions"
            :key="category"
            :label="category"
            :value="category === '全部' ? 'all' : category"
          />
        </el-select>
      </label>

      <label class="product-logs-page__filter">
        <span>操作类型</span>
        <el-select v-model="filters.operationType" clearable placeholder="全部">
          <el-option v-for="type in operationTypeOptions" :key="type" :label="type" :value="type" />
        </el-select>
      </label>

      <label class="product-logs-page__filter">
        <span>操作人</span>
        <el-input v-model="filters.operator" clearable placeholder="请输入操作人" />
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
          <el-option v-for="item in applicationOptions" :key="item" :label="item" :value="item" />
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
      <span>共 {{ filteredRecords.length }} 条</span>
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="PAGE_SIZE"
        :total="filteredRecords.length"
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
