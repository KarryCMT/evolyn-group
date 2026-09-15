<script setup lang="ts">
defineOptions({ name: 'PermissionGroupEditorNamePanel' });

const groupName = defineModel<string>('groupName', { required: true });
const description = defineModel<string>('description', { required: true });
</script>

<template>
  <section class="permission-group-editor-name-panel" aria-label="名称信息">
    <p class="permission-group-editor-name-panel__intro">可设置权限组名称和描述信息</p>
    <label class="permission-group-editor-name-panel__field">
      <span class="permission-group-editor-name-panel__label">权限组名称</span>
      <el-input v-model="groupName" maxlength="30" placeholder="请输入权限组名称" />
    </label>
    <label class="permission-group-editor-name-panel__field">
      <span class="permission-group-editor-name-panel__label">描述信息</span>
      <el-input
        v-model="description"
        type="textarea"
        :autosize="{ minRows: 7, maxRows: 7 }"
        maxlength="200"
        placeholder="填写此分组成员可以执行的操作范围"
      />
    </label>
  </section>
</template>

<style scoped lang="scss">
.permission-group-editor-name-panel {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-xl);

  &__intro {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__field {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
  }

  &__label {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-small);
    font-weight: 600;
    line-height: 20px;
  }
}
</style>
