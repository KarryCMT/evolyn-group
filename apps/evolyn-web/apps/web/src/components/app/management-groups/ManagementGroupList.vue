<script setup lang="ts">
import { RiDeleteBin6Fill, RiGroupFill } from '@remixicon/vue';
import type { ManagementGroup } from './managementGroup.types';

defineOptions({ name: 'ManagementGroupList' });

const props = defineProps<{
  groups: ManagementGroup[];
}>();

const emit = defineEmits<{
  remove: [id: string];
}>();
</script>

<template>
  <el-scrollbar class="management-group-list">
    <div class="management-group-list__content">
      <article v-for="group in props.groups" :key="group.id" class="management-group-list__card">
        <div class="management-group-list__icon"><RiGroupFill aria-hidden="true" /></div>
        <div class="management-group-list__summary">
          <h2>{{ group.name }}</h2>
          <p>{{ group.description || '未填写管理组描述' }}</p>
          <span
            >{{ group.managers.length }} 名管理员 ·
            {{
              group.permissionMode === 'all' ? '拥有应用全部权限' : '仅拥有部分表单/仪表盘权限'
            }}</span
          >
        </div>
        <button
          class="management-group-list__remove"
          type="button"
          :aria-label="`删除${group.name}`"
          @click="emit('remove', group.id)"
        >
          <RiDeleteBin6Fill aria-hidden="true" />
        </button>
      </article>
    </div>
  </el-scrollbar>
</template>

<style scoped lang="scss">
.management-group-list {
  min-height: 0;
  flex: 1;

  &__content {
    display: grid;
    padding: var(--el-space-3xl);
    grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));
    gap: var(--el-space-xl);
  }

  &__card {
    display: flex;
    min-height: 118px;
    padding: var(--el-space-xl);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-large);
    align-items: flex-start;
    gap: var(--el-space-lg);
    background: var(--el-fill-color-blank);
    box-shadow: var(--el-box-shadow-light);
  }

  &__icon {
    display: grid;
    width: 36px;
    height: 36px;
    flex: 0 0 auto;
    border-radius: var(--el-border-radius-medium);
    place-items: center;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);

    svg {
      width: 20px;
      height: 20px;
    }
  }

  &__summary {
    min-width: 0;
    flex: 1;

    h2,
    p {
      overflow: hidden;
      margin: 0;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    h2 {
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-medium);
      font-weight: 600;
      line-height: 24px;
    }

    p,
    span {
      display: block;
      color: var(--el-text-color-secondary);
      font-size: var(--el-font-size-small);
      line-height: 20px;
    }

    p {
      margin-top: var(--el-space-xs);
    }

    span {
      margin-top: var(--el-space-sm);
      color: var(--el-text-color-placeholder);
    }
  }

  &__remove {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-danger);
      background: var(--el-color-danger-light-9);
    }
  }
}
</style>
