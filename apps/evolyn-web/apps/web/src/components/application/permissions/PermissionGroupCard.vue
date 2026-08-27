<script setup lang="ts">
import { RiAddFill, RiMoreFill } from '@remixicon/vue';
import type { AssetPermissionGroup } from './permission.types';

defineOptions({ name: 'PermissionGroupCard' });

const props = defineProps<{
  group: AssetPermissionGroup;
}>();

const emit = defineEmits<{
  addSubjects: [groupId: string];
  clone: [groupId: string];
  edit: [groupId: string];
  remove: [groupId: string];
  updateEnabled: [payload: { groupId: string; enabled: boolean }];
}>();

function updateEnabled(enabled: boolean | string | number) {
  emit('updateEnabled', { groupId: props.group.id, enabled: Boolean(enabled) });
}
</script>

<template>
  <article
    class="permission-group-card"
    :class="{ 'permission-group-card--disabled': !props.group.enabled }"
  >
    <header class="permission-group-card__header">
      <div class="permission-group-card__heading">
        <div class="permission-group-card__title-row">
          <h3 class="permission-group-card__title">{{ props.group.name }}</h3>
          <span v-if="!props.group.enabled" class="permission-group-card__status">已停用</span>
        </div>
        <p class="permission-group-card__description">{{ props.group.description }}</p>
      </div>

      <div class="permission-group-card__actions">
        <button type="button" @click="emit('edit', props.group.id)">编辑</button>
        <button type="button" @click="emit('clone', props.group.id)">复制</button>
        <el-dropdown trigger="click" placement="bottom-end">
          <button class="permission-group-card__more" type="button" aria-label="更多权限组操作">
            <RiMoreFill aria-hidden="true" />
          </button>
          <template #dropdown>
            <el-dropdown-menu class="permission-group-card__menu">
              <el-dropdown-item>权限设置</el-dropdown-item>
              <el-dropdown-item divided @click="emit('remove', props.group.id)"
                >删除权限组</el-dropdown-item
              >
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-switch
          :model-value="props.group.enabled"
          aria-label="启用权限组"
          @update:model-value="updateEnabled"
        />
      </div>
    </header>

    <div class="permission-group-card__subjects">
      <div v-if="props.group.subjects.length" class="permission-group-card__subject-list">
        <span
          v-for="subject in props.group.subjects"
          :key="subject.id"
          class="permission-group-card__subject"
        >
          {{ subject.name }}
        </span>
      </div>
      <button
        class="permission-group-card__subject-picker"
        type="button"
        @click="emit('addSubjects', props.group.id)"
      >
        <RiAddFill aria-hidden="true" />
        {{ props.group.subjects.length ? '添加成员或部门' : '选择成员、部门或角色' }}
      </button>
    </div>
  </article>
</template>

<style scoped lang="scss">
.permission-group-card {
  padding: var(--el-space-2xl) var(--el-space-3xl) var(--el-space-2xl);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  background: var(--el-fill-color-blank);
  box-shadow: var(--el-box-shadow-light);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease;

  &:hover {
    border-color: var(--el-color-primary-light-7);
    box-shadow: var(--el-box-shadow-light);
  }

  &--disabled {
    background: var(--el-fill-color-lighter);
    box-shadow: none;
    opacity: 0.74;
  }

  &__header,
  &__title-row,
  &__actions,
  &__subjects,
  &__subject-list,
  &__subject-picker {
    display: flex;
    align-items: center;
  }

  &__header {
    justify-content: space-between;
    gap: var(--el-space-3xl);
  }

  &__heading {
    min-width: 0;
  }

  &__title-row {
    gap: var(--el-space-md);
  }

  &__title,
  &__description {
    margin: 0;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 650;
    line-height: 24px;
  }

  &__status {
    padding: var(--el-space-xs) var(--el-space-sm);
    border-radius: var(--el-border-radius-small);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color);
    font-size: var(--el-font-size-extra-small);
    line-height: 20px;
  }

  &__description {
    margin-top: var(--el-space-xs);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    line-height: 20px;
  }

  &__actions {
    flex: 0 0 auto;
    gap: var(--el-space-xs);

    > button,
    .permission-group-card__more {
      display: inline-flex;
      min-width: 32px;
      min-height: 30px;
      padding: 0 var(--el-space-sm);
      align-items: center;
      justify-content: center;
      border: 0;
      border-radius: var(--el-border-radius-small);
      color: var(--el-color-primary);
      cursor: pointer;
      background: transparent;
      font: inherit;
      font-size: var(--el-font-size-small);

      &:hover {
        background: var(--el-color-primary-light-9);
      }

      &:focus-visible {
        outline: 2px solid var(--el-color-primary);
        outline-offset: 2px;
      }
    }

    :deep(.el-switch) {
      margin-left: var(--el-space-md);
    }
  }

  &__more svg {
    width: 18px;
    height: 18px;
  }

  &__subjects {
    min-height: 76px;
    margin-top: var(--el-space-xl);
    padding: var(--el-space-lg) var(--el-space-xl);
    justify-content: space-between;
    gap: var(--el-space-xl);
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-medium);
    background: var(--el-fill-color-lighter);
  }

  &__subject-list {
    min-width: 0;
    flex-wrap: wrap;
    gap: var(--el-space-sm);
  }

  &__subject {
    max-width: 190px;
    padding: var(--el-space-xs) var(--el-space-md);
    overflow: hidden;
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--el-border-radius-half);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-blank);
    font-size: var(--el-font-size-small);
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__subject-picker {
    min-height: 32px;
    padding: 0 var(--el-space-md);
    flex: 0 0 auto;
    gap: var(--el-space-xs);
    border: 0;
    border-radius: var(--el-border-radius-small);
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    font: inherit;
    font-size: var(--el-font-size-small);

    svg {
      width: 16px;
      height: 16px;
    }

    &:hover {
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }
}

@media (max-width: 760px) {
  .permission-group-card {
    padding: var(--el-space-xl);

    &__header,
    &__subjects {
      align-items: flex-start;
      flex-direction: column;
    }

    &__actions {
      flex-wrap: wrap;
    }

    &__subject-picker {
      align-self: flex-end;
    }
  }
}
</style>
