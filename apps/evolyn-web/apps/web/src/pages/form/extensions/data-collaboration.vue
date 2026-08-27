<script setup lang="ts">
import { ElMessage, ElRadio, ElRadioGroup } from 'element-plus';
import { shallowRef } from 'vue';

defineOptions({ name: 'FormDataCollaborationPage' });

type DataTitleMode = 'default' | 'custom';

const titleMode = shallowRef<DataTitleMode>('default');
const customTitle = shallowRef('');
const allowTableEditing = shallowRef(false);

function saveSettings() {
  ElMessage.success('数据协作设置已保存');
}
</script>

<template>
  <section class="form-data-collaboration-page" aria-label="数据协作设置">
    <header class="form-data-collaboration-page__header">
      <h1 class="form-data-collaboration-page__title">数据协作</h1>
      <p class="form-data-collaboration-page__description">
        设置数据标题和编辑方式，控制成员在数据列表中的协作体验。
      </p>
    </header>

    <div class="form-data-collaboration-page__body">
      <section class="form-data-collaboration-page__setting-group">
        <h2 class="form-data-collaboration-page__setting-title">数据标题</h2>
        <el-radio-group v-model="titleMode" class="form-data-collaboration-page__title-mode">
          <el-radio value="default">默认标题</el-radio>
          <el-radio value="custom">自定义标题</el-radio>
        </el-radio-group>
        <el-input
          v-if="titleMode === 'custom'"
          v-model="customTitle"
          class="form-data-collaboration-page__custom-title"
          maxlength="50"
          placeholder="请输入数据标题"
          show-word-limit
        />
        <div v-else class="form-data-collaboration-page__default-title">项目编号</div>
      </section>

      <el-divider />

      <section class="form-data-collaboration-page__setting-group">
        <div class="form-data-collaboration-page__setting-heading">
          <div>
            <h2 class="form-data-collaboration-page__setting-title">表格视图直接编辑</h2>
            <p class="form-data-collaboration-page__setting-description">
              启用后，拥有编辑权限的成员可在数据表格中直接修改字段值。
            </p>
          </div>
          <el-switch v-model="allowTableEditing" />
        </div>
      </section>

      <el-button type="primary" @click="saveSettings">保存</el-button>
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-data-collaboration-page {
  min-height: 100%;

  &__header {
    display: flex;
    min-height: 68px;
    padding: 0 var(--el-space-3xl);
    align-items: center;
    gap: var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__title,
  &__setting-title {
    margin: 0;
    color: var(--el-text-color-primary);
  }

  &__title {
    font-size: var(--el-font-size-large);
    line-height: 28px;
  }

  &__description,
  &__setting-description {
    margin: 0;
    color: var(--el-text-color-secondary);
  }

  &__description {
    font-size: var(--el-font-size-base);
  }

  &__body {
    max-width: 760px;
    padding: var(--el-space-4xl) var(--el-space-3xl);
  }

  &__setting-group {
    padding: var(--el-space-xs) 0 var(--el-space-3xl);
  }

  &__setting-title {
    font-size: var(--el-font-size-medium);
    line-height: 24px;
  }

  &__title-mode {
    display: flex;
    margin-top: var(--el-space-xl);
  }

  &__custom-title,
  &__default-title {
    width: min(100%, 480px);
    margin-top: var(--el-space-lg);
  }

  &__default-title {
    height: 42px;
    padding: 0 var(--el-space-lg);
    display: flex;
    align-items: center;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__setting-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--el-space-3xl);
  }

  &__setting-description {
    margin-top: var(--el-space-md);
    font-size: var(--el-font-size-base);
    line-height: 22px;
  }
}

@media (max-width: 620px) {
  .form-data-collaboration-page {
    &__header {
      min-height: auto;
      padding: var(--el-space-xl);
      align-items: flex-start;
      flex-direction: column;
      gap: var(--el-space-sm);
    }

    &__body {
      padding: var(--el-space-3xl) var(--el-space-xl);
    }
  }
}
</style>
