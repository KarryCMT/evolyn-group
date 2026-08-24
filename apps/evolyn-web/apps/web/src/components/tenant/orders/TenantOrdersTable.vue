<script setup lang="ts">
import { RiFileList3Fill } from '@remixicon/vue';

defineOptions({ name: 'TenantOrdersTable' });

const selectedAll = defineModel<boolean>('selectedAll', { required: true });

const columns = ['订单信息', '订单总价', '支付时间', '订单状态', '发票状态', '操作'];
</script>

<template>
  <section class="tenant-orders-table" aria-label="订单列表">
    <header class="tenant-orders-table__header" role="row">
      <div class="tenant-orders-table__selection">
        <el-checkbox v-model="selectedAll" aria-label="全选订单" />
      </div>
      <div
        v-for="column in columns"
        :key="column"
        class="tenant-orders-table__column"
        role="columnheader"
      >
        {{ column }}
      </div>
    </header>

    <div class="tenant-orders-table__empty" role="status">
      <div class="tenant-orders-table__illustration" aria-hidden="true">
        <span class="tenant-orders-table__ground" />
        <span class="tenant-orders-table__tree tenant-orders-table__tree--left" />
        <span class="tenant-orders-table__tree tenant-orders-table__tree--right" />
        <span class="tenant-orders-table__paper">
          <RiFileList3Fill />
          <i />
          <i />
        </span>
        <span class="tenant-orders-table__pencil" />
      </div>
      <p>暂无订单信息</p>
    </div>
  </section>
</template>

<style scoped lang="scss">
.tenant-orders-table {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;

  &__header {
    display: grid;
    min-height: 76px;
    grid-template-columns: 54px minmax(200px, 1.85fr) repeat(4, minmax(110px, 1fr)) minmax(
        80px,
        0.64fr
      );
    align-items: center;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-fill-color-light);
  }

  &__selection {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  &__column {
    padding: 0 8px;
    color: var(--el-text-color-regular);
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
    text-align: center;
    white-space: nowrap;
  }

  &__empty {
    display: flex;
    min-height: 340px;
    padding-bottom: 11%;
    align-items: center;
    justify-content: center;
    flex: 1;
    flex-direction: column;

    p {
      margin: 12px 0 0;
      color: var(--el-text-color-secondary);
      font-size: 16px;
      line-height: 24px;
    }
  }

  &__illustration {
    position: relative;
    width: 184px;
    height: 132px;
  }

  &__ground {
    position: absolute;
    bottom: 16px;
    left: 16px;
    width: 152px;
    height: 22px;
    border-radius: 50%;
    opacity: 0.82;
    background: linear-gradient(90deg, transparent, var(--el-fill-color), transparent);
  }

  &__tree {
    position: absolute;
    z-index: 1;
    bottom: 24px;
    width: 13px;
    height: 22px;
    border-radius: 9px 9px 7px 7px;
    background: var(--el-color-primary-light-8);

    &::after {
      position: absolute;
      bottom: -9px;
      left: 5px;
      width: 3px;
      height: 11px;
      border-radius: 3px;
      background: var(--el-color-primary-light-7);
      content: '';
    }

    &--left {
      left: 31px;
    }

    &--right {
      right: 26px;
      width: 11px;
      height: 18px;
    }
  }

  &__paper {
    position: absolute;
    z-index: 2;
    top: 25px;
    left: 68px;
    display: flex;
    width: 68px;
    height: 76px;
    border: 4px solid var(--el-color-primary-light-5);
    border-radius: 5px;
    align-items: center;
    justify-content: center;
    color: var(--el-color-primary-light-7);
    background: var(--el-bg-color);
    box-shadow: 7px 7px 0 var(--el-color-primary-light-9);
    transform: rotate(34deg);

    svg {
      width: 39px;
      height: 39px;
    }

    i {
      position: absolute;
      right: 8px;
      width: 13px;
      height: 3px;
      border-radius: 99px;
      background: var(--el-color-primary-light-5);

      &:first-of-type {
        top: 15px;
      }

      &:last-of-type {
        top: 23px;
      }
    }
  }

  &__pencil {
    position: absolute;
    z-index: 3;
    top: 13px;
    left: 58px;
    width: 6px;
    height: 53px;
    border-radius: 99px;
    background: var(--el-color-warning-light-5);
    transform: rotate(-30deg);

    &::after {
      position: absolute;
      bottom: -5px;
      left: 0;
      width: 0;
      height: 0;
      border-top: 7px solid var(--el-text-color-secondary);
      border-right: 3px solid transparent;
      border-left: 3px solid transparent;
      content: '';
    }
  }
}

@media (max-width: 980px) {
  .tenant-orders-table {
    min-width: 860px;
  }
}
</style>
