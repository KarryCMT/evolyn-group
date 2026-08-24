<script setup lang="ts">
defineOptions({ name: 'DataPushFilterBar' });

const props = defineProps<{
  serverAddresses: string[];
  formNames: string[];
}>();

const emit = defineEmits<{
  query: [];
}>();

const serverAddress = defineModel<string>('serverAddress', { required: true });
const formName = defineModel<string>('formName', { required: true });
</script>

<template>
  <div class="data-push-filter-bar" aria-label="数据推送筛选">
    <label class="data-push-filter-bar__field">
      <span>服务器地址：</span>
      <el-select v-model="serverAddress" clearable placeholder="全部服务器">
        <el-option
          v-for="address in props.serverAddresses"
          :key="address"
          :label="address"
          :value="address"
        />
      </el-select>
    </label>
    <label class="data-push-filter-bar__field">
      <span>推送表单：</span>
      <el-select v-model="formName" clearable placeholder="全部表单">
        <el-option v-for="name in props.formNames" :key="name" :label="name" :value="name" />
      </el-select>
    </label>
    <el-button type="primary" class="data-push-filter-bar__query" @click="emit('query')">
      查询
    </el-button>
  </div>
</template>

<style scoped lang="scss">
.data-push-filter-bar {
  display: flex;
  min-height: 64px;
  padding: 14px 28px 0;
  align-items: center;
  flex: 0 0 auto;
  gap: 30px;

  &__field {
    display: inline-flex;
    align-items: center;
    color: var(--el-text-color-primary);
    font-size: 16px;
    line-height: 32px;
    white-space: nowrap;
    gap: 12px;

    :deep(.el-select) {
      width: 360px;
    }
  }

  &__query {
    min-width: 74px;
    height: 40px;
    font-size: 15px;
  }
}

@media (max-width: 1080px) {
  .data-push-filter-bar {
    flex-wrap: wrap;
    gap: 10px 20px;

    &__field :deep(.el-select) {
      width: 250px;
    }
  }
}
</style>
