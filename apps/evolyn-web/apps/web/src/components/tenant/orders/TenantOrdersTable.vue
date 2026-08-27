<script setup lang="ts">
import type { EvolynTableColumn, EvolynTableRow } from '@evolyn.do/ui';
import { EvolynTable } from '@evolyn.do/ui';
import { isDark } from '~/composables/dark';

defineOptions({ name: 'TenantOrdersTable' });

// 前端还原阶段暂无订单数据；后端订单接口接入后替换为分页结果。
const records: EvolynTableRow[] = [];

// 列结构沿用原实现的比例：选择列固定窄宽，订单信息为主列，其余等宽；
// headerType 显式给 checkbox 才会在表头渲染全选框（VTable 默认表头是 text）。
const columns: EvolynTableColumn[] = [
  { field: 'checked', title: '', width: 54, cellType: 'checkbox', headerType: 'checkbox' },
  { field: 'orderInfo', title: '订单信息', minWidth: 200, align: 'center' },
  { field: 'totalPrice', title: '订单总价', minWidth: 110, align: 'center' },
  { field: 'paidAt', title: '支付时间', minWidth: 110, align: 'center' },
  { field: 'orderStatus', title: '订单状态', minWidth: 110, align: 'center' },
  { field: 'invoiceStatus', title: '发票状态', minWidth: 110, align: 'center' },
  { field: 'operation', title: '操作', minWidth: 80, align: 'center' },
];

// 表头高度沿用原设计 76px；行高预留订单信息多行内容的空间。
const tableOptions = { defaultHeaderRowHeight: 76, defaultRowHeight: 64 };
</script>

<template>
  <EvolynTable
    class="tenant-orders-table"
    aria-label="订单列表"
    :columns="columns"
    :records="records"
    :options="tableOptions"
    :theme="isDark ? 'dark' : 'light'"
    empty-text="暂无订单信息"
  />
</template>

<style scoped lang="scss">
.tenant-orders-table {
  min-height: 340px;
  flex: 1;
}

@media (max-width: 980px) {
  .tenant-orders-table {
    min-width: 860px;
  }
}
</style>
