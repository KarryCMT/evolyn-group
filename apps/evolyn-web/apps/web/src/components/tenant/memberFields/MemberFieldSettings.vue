<script setup lang="ts">
import { ref } from 'vue';
import type { MemberProfileField } from './memberField.types';
import { memberProfileFields } from './memberField.types';

defineOptions({ name: 'MemberFieldSettings' });

/** 本页先以本地状态呈现，后续接入字段配置接口时可直接替换为服务端数据。 */
const fields = ref(memberProfileFields.map((field) => ({ ...field })));

function handleVisibleChange(field: MemberProfileField) {
  // 隐藏字段后个人设置页不应再保留编辑入口。
  if (!field.visible) {
    field.editable = false;
  }
}
</script>

<template>
  <section class="member-field-settings" aria-labelledby="member-field-settings-title">
    <p id="member-field-settings-title" class="member-field-settings__description">
      管理成员的信息字段及权限
    </p>

    <section class="member-field-settings__table" aria-label="成员字段权限设置">
      <el-table
        :data="fields"
        row-key="key"
        height="100%"
        table-layout="fixed"
        scrollbar-always-on
        class="member-field-settings__data-table"
      >
        <el-table-column prop="label" label="成员信息" min-width="180" />
        <el-table-column prop="type" label="字段类型" min-width="250" />
        <el-table-column label="成员在「个人设置」页的权限" min-width="320">
          <template #default="{ row: field }">
            <div class="member-field-settings__permissions">
              <el-checkbox
                v-model="field.visible"
                :disabled="field.visibilityLocked"
                :aria-label="`${field.label}在个人设置页可见`"
                @change="handleVisibleChange(field as MemberProfileField)"
              >
                可见
              </el-checkbox>
              <el-checkbox
                v-model="field.editable"
                :disabled="field.editableLocked || !field.visible"
                :aria-label="`${field.label}在个人设置页可编辑`"
              >
                可编辑
              </el-checkbox>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<style scoped lang="scss">
.member-field-settings {
  display: flex;
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  padding: 22px 22px 20px;

  &__description {
    flex: 0 0 auto;
    margin: 0 0 9px;
    color: var(--el-text-color-secondary);
    font-size: 16px;
    line-height: 24px;
  }

  &__table {
    min-height: 0;
    flex: 1;
    overflow: hidden;
  }

  &__data-table {
    height: 100%;
    --el-table-border-color: var(--el-border-color-lighter);
    --el-table-header-bg-color: var(--el-fill-color-light);
    --el-table-header-text-color: var(--el-text-color-primary);
    --el-table-row-hover-bg-color: var(--el-fill-color-light);
    --el-table-text-color: var(--el-text-color-primary);
    font-size: 16px;

    :deep(.el-table__header-wrapper .el-table__cell) {
      height: 56px;
      color: var(--el-text-color-primary);
      font-weight: 700;
    }

    :deep(.el-table__body-wrapper .el-table__cell) {
      height: 56px;
    }

    :deep(.el-table__cell) {
      padding: 0 16px;
    }
  }

  &__permissions {
    display: flex;
    align-items: center;
    gap: 18px;
  }
}

@media (max-width: 840px) {
  .member-field-settings {
    padding: 18px 16px 16px;

    &__description {
      margin-bottom: 16px;
    }
  }
}
</style>
