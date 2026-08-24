<script setup lang="ts">
import type { InvoiceStatusFilter, OrderStatusFilter } from './types';

defineOptions({ name: 'TenantOrdersFilterBar' });

const orderStatus = defineModel<OrderStatusFilter>('orderStatus', { required: true });
const invoiceStatus = defineModel<InvoiceStatusFilter>('invoiceStatus', { required: true });

const emit = defineEmits<{
  mergeInvoices: [];
  viewInvoices: [];
}>();

const orderStatusOptions: Array<{ label: string; value: OrderStatusFilter }> = [
  { label: '全部', value: 'all' },
  { label: '待支付', value: 'pending-payment' },
  { label: '已支付', value: 'paid' },
  { label: '已关闭', value: 'closed' },
];

const invoiceStatusOptions: Array<{ label: string; value: InvoiceStatusFilter }> = [
  { label: '全部', value: 'all' },
  { label: '未申请', value: 'not-applied' },
  { label: '开票中', value: 'processing' },
  { label: '已开票', value: 'issued' },
];
</script>

<template>
  <header class="tenant-orders-filter-bar">
    <div class="tenant-orders-filter-bar__filters" aria-label="订单筛选">
      <label class="tenant-orders-filter-bar__filter">
        <span>订单状态：</span>
        <el-select v-model="orderStatus" aria-label="订单状态" placeholder="请选择订单状态">
          <el-option
            v-for="item in orderStatusOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </label>
      <label class="tenant-orders-filter-bar__filter">
        <span>发票状态：</span>
        <el-select v-model="invoiceStatus" aria-label="发票状态" placeholder="请选择发票状态">
          <el-option
            v-for="item in invoiceStatusOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </label>
    </div>
    <div class="tenant-orders-filter-bar__actions">
      <el-button plain type="primary" @click="emit('viewInvoices')">我的发票</el-button>
      <el-button type="primary" @click="emit('mergeInvoices')">合并开票</el-button>
    </div>
  </header>
</template>

<style scoped lang="scss">
.tenant-orders-filter-bar {
  display: flex;
  min-height: 48px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;

  &__filters,
  &__actions,
  &__filter {
    display: flex;
    align-items: center;
  }

  &__filters {
    flex-wrap: wrap;
    gap: 20px 40px;
  }

  &__filter {
    gap: 12px;
    color: var(--el-text-color-regular);
    font-size: 16px;
    line-height: 24px;
    white-space: nowrap;

    .el-select {
      width: 194px;
    }
  }

  &__actions {
    flex: 0 0 auto;
    gap: 12px;
  }
}

@media (max-width: 760px) {
  .tenant-orders-filter-bar {
    align-items: flex-start;
    flex-direction: column;

    &__filters,
    &__actions {
      width: 100%;
    }

    &__filter {
      flex: 1;

      .el-select {
        min-width: 0;
        flex: 1;
      }
    }

    &__actions {
      justify-content: flex-end;
    }
  }
}
</style>
