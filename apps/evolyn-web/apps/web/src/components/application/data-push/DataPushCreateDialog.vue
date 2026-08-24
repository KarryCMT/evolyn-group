<script setup lang="ts">
defineOptions({ name: 'DataPushCreateDialog' });

const emit = defineEmits<{
  confirm: [];
}>();
const visible = defineModel<boolean>('visible', { required: true });
const selectedForm = defineModel<string>('selectedForm', { required: true });

const formOptions = [
  { value: 'order-management', label: '订单管理' },
  { value: 'purchase-request', label: '采购申请' },
  { value: 'employee-profile', label: '员工档案' },
  { value: 'office-supplies', label: '办公用品申请' },
];

function confirm() {
  emit('confirm');
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="data-push-create-dialog"
    title="新建数据推送"
    width="880px"
    align-center
    append-to-body
    destroy-on-close
  >
    <div class="data-push-create-dialog__body">
      <label class="data-push-create-dialog__label" for="data-push-form">推送表单</label>
      <el-select id="data-push-form" v-model="selectedForm" class="data-push-create-dialog__select">
        <el-option
          v-for="item in formOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>
    </div>
    <template #footer>
      <div class="data-push-create-dialog__footer">
        <el-button @click="visible = false"> 取消 </el-button>
        <el-button type="primary" @click="confirm"> 确定 </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.data-push-create-dialog {
  &__body {
    min-height: 410px;
    padding: 28px;
  }

  &__label {
    display: block;
    margin-bottom: 12px;
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
  }

  &__select {
    width: 100%;
  }

  &__footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
}
</style>
