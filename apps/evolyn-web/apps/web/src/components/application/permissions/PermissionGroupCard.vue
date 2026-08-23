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
  padding: 22px 24px 20px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-blank);
  box-shadow: 0 4px 18px rgb(31 45 61 / 3%);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease;

  &:hover {
    border-color: var(--el-color-primary-light-7);
    box-shadow: 0 8px 22px rgb(31 45 61 / 7%);
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
    gap: 24px;
  }

  &__heading {
    min-width: 0;
  }

  &__title-row {
    gap: 8px;
  }

  &__title,
  &__description {
    margin: 0;
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 650;
    line-height: 24px;
  }

  &__status {
    padding: 1px 7px;
    border-radius: var(--el-border-radius-small);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color);
    font-size: 12px;
    line-height: 20px;
  }

  &__description {
    margin-top: 5px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }

  &__actions {
    flex: 0 0 auto;
    gap: 3px;

    > button,
    .permission-group-card__more {
      display: inline-flex;
      min-width: 32px;
      min-height: 30px;
      padding: 0 7px;
      align-items: center;
      justify-content: center;
      border: 0;
      border-radius: var(--el-border-radius-small);
      color: var(--el-color-primary);
      cursor: pointer;
      background: transparent;
      font: inherit;
      font-size: 13px;

      &:hover {
        background: var(--el-color-primary-light-9);
      }

      &:focus-visible {
        outline: 2px solid var(--el-color-primary);
        outline-offset: 2px;
      }
    }

    :deep(.el-switch) {
      margin-left: 8px;
    }
  }

  &__more svg {
    width: 18px;
    height: 18px;
  }

  &__subjects {
    min-height: 76px;
    margin-top: 18px;
    padding: 13px 15px;
    justify-content: space-between;
    gap: 16px;
    border: 1px dashed var(--el-border-color);
    border-radius: 8px;
    background: var(--el-fill-color-lighter);
  }

  &__subject-list {
    min-width: 0;
    flex-wrap: wrap;
    gap: 7px;
  }

  &__subject {
    max-width: 190px;
    padding: 4px 9px;
    overflow: hidden;
    border: 1px solid var(--el-border-color-light);
    border-radius: 999px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-blank);
    font-size: 13px;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__subject-picker {
    min-height: 32px;
    padding: 0 8px;
    flex: 0 0 auto;
    gap: 5px;
    border: 0;
    border-radius: var(--el-border-radius-small);
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    font: inherit;
    font-size: 13px;

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
    padding: 18px;

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
