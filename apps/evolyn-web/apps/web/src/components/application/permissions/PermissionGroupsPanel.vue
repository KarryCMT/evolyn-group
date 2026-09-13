<script setup lang="ts">
import type { AssetPermissionGroup, PermissionAsset } from './permission.types';
import { RiAddFill, RiForbid2Fill } from '@remixicon/vue';
import PermissionGroupCard from './PermissionGroupCard.vue';

defineOptions({ name: 'PermissionGroupsPanel' });

const props = defineProps<{
  asset: PermissionAsset | undefined;
  /** 仅当前资产的权限详情加载；左侧资产树在此期间始终保留。 */
  loading: boolean;
  errorMessage: string;
  groups: AssetPermissionGroup[];
}>();

const emit = defineEmits<{
  addGroup: [];
  addSubjects: [groupId: string];
  cloneGroup: [groupId: string];
  disableAll: [];
  editGroup: [groupId: string];
  removeGroup: [groupId: string];
  reload: [];
  updateGroupEnabled: [payload: { groupId: string; enabled: boolean }];
}>();
</script>

<template>
  <section class="permission-groups-panel" aria-label="配置权限">
    <header class="permission-groups-panel__header">
      <div class="permission-groups-panel__heading">
        <p class="permission-groups-panel__step">02</p>
        <h1 class="permission-groups-panel__title">配置权限</h1>
        <p class="permission-groups-panel__description">
          <template v-if="props.asset">
            为「{{ props.asset.name }}」添加成员，并配置其访问和数据操作范围。
          </template>
          <template v-else> 请先在左侧选择一个表单。 </template>
        </p>
      </div>
      <div class="permission-groups-panel__header-actions">
        <button
          class="permission-groups-panel__add"
          type="button"
          :disabled="!props.asset || props.loading"
          @click="emit('addGroup')"
        >
          <RiAddFill aria-hidden="true" />
          添加成员
        </button>
        <button
          class="permission-groups-panel__disable-all"
          type="button"
          :disabled="props.loading || !props.groups.some((group) => group.enabled)"
          @click="emit('disableAll')"
        >
          <RiForbid2Fill aria-hidden="true" />
          停用全部
        </button>
      </div>
    </header>

    <!-- 仅权限组内容区域滚动，顶部标题和全局操作固定。 -->
    <el-scrollbar v-if="props.asset" class="permission-groups-panel__scrollbar">
      <div class="permission-groups-panel__content">
        <div class="permission-groups-panel__context">
          <span class="permission-groups-panel__context-label">当前资产</span>
          <strong>{{ props.asset.name }}</strong>
          <span>{{ props.asset.type === 'workflow-form' ? '流程表单权限' : '普通表单权限' }}</span>
        </div>
        <div v-if="props.loading" class="permission-groups-panel__loading" role="status">
          正在加载权限配置…
        </div>
        <div v-else-if="props.errorMessage" class="permission-groups-panel__error" role="alert">
          <p>{{ props.errorMessage }}</p>
          <button type="button" @click="emit('reload')">重新加载</button>
        </div>
        <div v-else-if="props.groups.length" class="permission-groups-panel__groups">
          <PermissionGroupCard
            v-for="group in props.groups"
            :key="group.code"
            :group="group"
            @add-subjects="emit('addSubjects', $event)"
            @clone="emit('cloneGroup', $event)"
            @edit="emit('editGroup', $event)"
            @remove="emit('removeGroup', $event)"
            @update-enabled="emit('updateGroupEnabled', $event)"
          />
        </div>
        <div v-else class="permission-groups-panel__empty">
          <p>尚未配置权限组</p>
          <span>添加成员后，成员才能在应用中看到并使用该资产。</span>
          <button type="button" @click="emit('addGroup')">
            <RiAddFill aria-hidden="true" /> 添加第一组成员
          </button>
        </div>
      </div>
    </el-scrollbar>
  </section>
</template>

<style scoped lang="scss">
.permission-groups-panel {
  display: flex;
  min-height: 0;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  background: var(--el-fill-color-blank);

  &__header {
    display: flex;
    min-height: 106px;
    padding: var(--el-space-2xl) var(--el-space-4xl);
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--el-space-2xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__step,
  &__title,
  &__description {
    margin: 0;
  }

  &__step {
    margin-bottom: var(--el-space-xs);
    color: var(--el-color-primary);
    font-size: var(--el-font-size-extra-small);
    font-weight: 700;
    letter-spacing: 0.08em;
    line-height: 16px;
  }

  &__title {
    display: inline;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);
    font-weight: 650;
    line-height: 26px;
  }

  &__description {
    display: inline;
    margin-left: var(--el-space-lg);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    line-height: 22px;
  }

  &__header-actions {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    gap: var(--el-space-md);
  }

  &__add,
  &__disable-all,
  &__empty button {
    display: inline-flex;
    min-height: 36px;
    padding: 0 var(--el-space-lg);
    align-items: center;
    justify-content: center;
    gap: var(--el-space-sm);
    border: 0;
    border-radius: var(--el-border-radius-base);
    cursor: pointer;
    font: inherit;
    font-size: var(--el-font-size-small);
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 17px;
      height: 17px;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &:disabled {
      cursor: not-allowed;
      opacity: 0.5;
    }
  }

  &__add,
  &__empty button {
    color: var(--el-color-white);
    background: var(--el-color-primary);

    &:not(:disabled):hover {
      background: var(--el-color-primary-light-3);
    }
  }

  &__disable-all {
    color: var(--el-color-danger);
    background: transparent;

    &:not(:disabled):hover {
      background: var(--el-color-danger-light-9);
    }
  }

  &__scrollbar {
    min-height: 0;
    flex: 1;
  }

  &__content {
    box-sizing: border-box;
    min-height: 100%;
    padding: var(--el-space-3xl) var(--el-space-4xl) var(--el-space-5xl);
  }

  &__context {
    display: flex;
    margin-bottom: var(--el-space-2xl);
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);

    strong {
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-base);
      font-weight: 600;
    }
  }

  &__context-label {
    padding: var(--el-space-xs) var(--el-space-sm);
    border-radius: var(--el-border-radius-small);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    font-size: var(--el-font-size-extra-small);
  }

  &__groups {
    display: flex;
    width: 100%;
    flex-direction: column;
    gap: var(--el-space-lg);
  }

  &__loading,
  &__error {
    box-sizing: border-box;
    display: flex;
    width: 100%;
    min-height: 250px;
    padding: var(--el-space-3xl);
    align-items: center;
    justify-content: center;
    flex-direction: column;
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-large);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-lighter);
    font-size: var(--el-font-size-small);
  }

  &__error {
    p {
      margin: 0;
    }

    button {
      min-height: 32px;
      margin-top: var(--el-space-lg);
      padding: 0 var(--el-space-lg);
      border: 0;
      border-radius: var(--el-border-radius-base);
      color: var(--el-color-white);
      cursor: pointer;
      background: var(--el-color-primary);
      font: inherit;
    }
  }

  &__empty {
    box-sizing: border-box;
    display: flex;
    width: 100%;
    min-height: 250px;
    padding: var(--el-space-3xl);
    align-items: center;
    justify-content: center;
    flex-direction: column;
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-large);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-lighter);
    text-align: center;

    p {
      margin: 0;
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-medium);
      font-weight: 600;
    }

    span {
      margin-top: var(--el-space-sm);
      font-size: var(--el-font-size-small);
      line-height: 20px;
    }

    button {
      margin-top: var(--el-space-xl);
    }
  }
}

@media (max-width: 760px) {
  .permission-groups-panel {
    &__header {
      padding: var(--el-space-2xl);
      flex-direction: column;
    }

    &__description {
      display: block;
      margin: var(--el-space-xs) 0 0;
    }

    &__content {
      padding: var(--el-space-2xl);
    }
  }
}
</style>
