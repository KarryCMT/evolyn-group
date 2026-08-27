<script setup lang="ts">
import { RiBarChartBoxFill, RiFileList3Fill, RiGitBranchFill, RiSearchFill } from '@remixicon/vue';
import { computed } from 'vue';
import type { PermissionAsset, PermissionAssetType } from './permission.types';

defineOptions({ name: 'PermissionAssetList' });

const props = defineProps<{
  assets: PermissionAsset[];
  keyword: string;
  selectedAssetId: string;
}>();

const emit = defineEmits<{
  batchSelect: [];
  updateKeyword: [keyword: string];
  select: [assetId: string];
}>();

const assetGroups: Array<{
  type: PermissionAssetType;
  label: string;
  icon: typeof RiFileList3Fill;
}> = [
  { type: 'workflow-form', label: '流程表单', icon: RiGitBranchFill },
  { type: 'form', label: '表单', icon: RiFileList3Fill },
  { type: 'dashboard', label: '仪表盘', icon: RiBarChartBoxFill },
];

const normalizedKeyword = computed(() => props.keyword.trim().toLocaleLowerCase());
const visibleGroups = computed(() =>
  assetGroups
    .map((group) => ({
      ...group,
      assets: props.assets.filter(
        (asset) =>
          asset.type === group.type &&
          (!normalizedKeyword.value ||
            asset.name.toLocaleLowerCase().includes(normalizedKeyword.value)),
      ),
    }))
    .filter((group) => group.assets.length > 0),
);

function updateKeyword(event: Event) {
  emit('updateKeyword', (event.target as HTMLInputElement).value);
}
</script>

<template>
  <aside class="permission-asset-list" aria-label="选择表单或仪表盘">
    <header class="permission-asset-list__header">
      <div>
        <p class="permission-asset-list__step">01</p>
        <h2 class="permission-asset-list__title">选择资产</h2>
      </div>
      <button class="permission-asset-list__batch" type="button" @click="emit('batchSelect')">
        批量选择
      </button>
    </header>

    <label class="permission-asset-list__search">
      <RiSearchFill aria-hidden="true" />
      <input
        :value="props.keyword"
        type="search"
        placeholder="搜索表单或仪表盘"
        aria-label="搜索表单或仪表盘"
        @input="updateKeyword"
      />
    </label>

    <!-- 资产列表独立滚动，搜索框与批量操作始终保留在可见区域。 -->
    <el-scrollbar class="permission-asset-list__scrollbar">
      <div v-if="visibleGroups.length" class="permission-asset-list__groups">
        <section
          v-for="group in visibleGroups"
          :key="group.type"
          class="permission-asset-list__group"
          :aria-label="group.label"
        >
          <p class="permission-asset-list__group-title">{{ group.label }}</p>
          <button
            v-for="asset in group.assets"
            :key="asset.id"
            class="permission-asset-list__asset"
            :class="{ 'permission-asset-list__asset--active': props.selectedAssetId === asset.id }"
            type="button"
            @click="emit('select', asset.id)"
          >
            <component :is="group.icon" aria-hidden="true" />
            <span>{{ asset.name }}</span>
          </button>
        </section>
      </div>
      <div v-else class="permission-asset-list__empty">未找到匹配的应用资产</div>
    </el-scrollbar>
  </aside>
</template>

<style scoped lang="scss">
.permission-asset-list {
  box-sizing: border-box;
  display: flex;
  min-height: 0;
  width: 292px;
  min-width: 292px;
  padding: var(--el-space-2xl) var(--el-space-xl);
  overflow: hidden;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);

  &__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--el-space-lg);
  }

  &__step,
  &__title,
  &__group-title {
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
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);
    font-weight: 650;
    line-height: 26px;
  }

  &__batch {
    min-height: 30px;
    padding: 0 var(--el-space-sm);
    border: 0;
    border-radius: var(--el-border-radius-small);
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    font-size: var(--el-font-size-small);

    &:hover {
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__search {
    display: flex;
    height: 38px;
    margin-top: var(--el-space-2xl);
    padding: 0 var(--el-space-lg);
    align-items: center;
    gap: var(--el-space-md);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-placeholder);
    background: var(--el-fill-color-light);

    &:focus-within {
      outline: 2px solid var(--el-color-primary-light-5);
      background: var(--el-fill-color-blank);
    }

    svg {
      width: 17px;
      height: 17px;
      flex: 0 0 auto;
    }

    input {
      width: 100%;
      min-width: 0;
      border: 0;
      outline: 0;
      color: var(--el-text-color-primary);
      background: transparent;
      font: inherit;
      font-size: var(--el-font-size-small);

      &::placeholder {
        color: var(--el-text-color-placeholder);
      }
    }
  }

  &__scrollbar {
    height: 0;
    margin-top: var(--el-space-2xl);
    min-height: 0;
    flex: 1;
  }

  &__groups {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-xl);
  }

  &__group {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-xs);
  }

  &__group-title {
    padding: 0 var(--el-space-md) var(--el-space-xs);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    line-height: 18px;
  }

  &__asset {
    display: flex;
    min-height: 38px;
    padding: 0 var(--el-space-md);
    align-items: center;
    gap: var(--el-space-md);
    border: 0;
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    font: inherit;
    font-size: var(--el-font-size-base);
    text-align: left;

    svg {
      width: 18px;
      height: 18px;
      flex: 0 0 auto;
      color: var(--el-text-color-secondary);
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-fill-color-light);

      svg {
        color: var(--el-color-primary);
      }
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--active {
      color: var(--el-color-primary);
      font-weight: 600;
      background: var(--el-color-primary-light-9);

      svg {
        color: var(--el-color-primary);
      }
    }
  }

  &__empty {
    display: grid;
    min-height: 180px;
    place-items: center;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }
}

@media (max-width: 900px) {
  .permission-asset-list {
    width: 240px;
    min-width: 240px;
  }
}
</style>
