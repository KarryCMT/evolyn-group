<script setup lang="ts">
import type { InvoiceStatusFilter, OrderStatusFilter } from '~/components/tenant/orders/types';
import { ElMessage } from 'element-plus';
import { shallowRef } from 'vue';
import TenantOrdersFilterBar from '~/components/tenant/orders/TenantOrdersFilterBar.vue';
import TenantOrdersTable from '~/components/tenant/orders/TenantOrdersTable.vue';

defineOptions({ name: 'TenantOrdersPage' });

const orderStatus = shallowRef<OrderStatusFilter>('all');
const invoiceStatus = shallowRef<InvoiceStatusFilter>('all');

function viewInvoices() {
  ElMessage.info('发票中心将在订单服务接入后开放');
}

function mergeInvoices() {
  ElMessage.info('请先选择可开票订单，订单服务接入后即可合并开票');
}
</script>

<template>
  <section class="tenant-orders-page" aria-label="订单信息">
    <TenantOrdersFilterBar
      v-model:invoice-status="invoiceStatus"
      v-model:order-status="orderStatus"
      @merge-invoices="mergeInvoices"
      @view-invoices="viewInvoices"
    />
    <div class="tenant-orders-page__table-scroll">
      <el-scrollbar class="tenant-orders-page__scrollbar">
        <TenantOrdersTable />
      </el-scrollbar>
    </div>
  </section>
</template>

<style scoped lang="scss">
.tenant-orders-page {
  display: flex;
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  padding: var(--el-space-3xl) var(--el-space-4xl) var(--el-space-2xl);
  flex-direction: column;
  gap: var(--el-space-2xl);

  &__table-scroll,
  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__scrollbar :deep(.el-scrollbar__view) {
    display: flex;
    min-height: 100%;
    flex-direction: column;
  }
}

@media (max-width: 760px) {
  .tenant-orders-page {
    padding: var(--el-space-2xl);
  }
}
</style>
