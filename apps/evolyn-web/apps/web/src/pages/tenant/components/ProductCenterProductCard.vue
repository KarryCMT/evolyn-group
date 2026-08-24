<script setup lang="ts">
import { RiArrowRightLine, RiGroupFill, RiQuestionFill, RiUserFill } from '@remixicon/vue';
import { ElRadio, ElRadioGroup, ElSwitch, ElTooltip } from 'element-plus';
import type { EvolynMemberDepartmentRolePickerSelection } from '@evolyn.do/ui';

export type ProductMemberScope = 'all' | 'partial';

interface ProductCenterProduct {
  memberCount: number;
  name: string;
  versionName: string;
}

const props = defineProps<{
  enabled: boolean;
  product: ProductCenterProduct;
  selections: EvolynMemberDepartmentRolePickerSelection[];
  scope: ProductMemberScope;
}>();

const emit = defineEmits<{
  enterProduct: [];
  editMemberScope: [];
  selectPartialScope: [];
  updateEnabled: [enabled: boolean];
  updateScope: [scope: ProductMemberScope];
}>();

function updateScope(scope: ProductMemberScope) {
  emit('updateScope', scope);
}

/** Element Plus 的通用 v-model 事件包含多种基础类型，这里收窄为产品开关需要的布尔值。 */
function updateEnabled(enabled: string | number | boolean) {
  if (typeof enabled === 'boolean') {
    emit('updateEnabled', enabled);
  }
}

/** 可用范围只接受本页定义的两个稳定枚举值，避免组件内部状态被无效值污染。 */
function selectScope(scope: string | number | boolean | undefined) {
  if (scope === 'all') updateScope(scope);
  if (scope === 'partial') emit('selectPartialScope');
}
</script>

<template>
  <article class="product-center-card" :class="{ 'product-center-card--disabled': !props.enabled }">
    <header class="product-center-card__header">
      <div class="product-center-card__identity">
        <span class="product-center-card__mark" aria-hidden="true">
          <i class="product-center-card__mark-petal product-center-card__mark-petal--left" />
          <i class="product-center-card__mark-petal product-center-card__mark-petal--top" />
          <i class="product-center-card__mark-petal product-center-card__mark-petal--right" />
        </span>
        <h1 class="product-center-card__name">{{ props.product.name }}</h1>
      </div>
      <ElSwitch
        class="product-center-card__switch"
        :model-value="props.enabled"
        aria-label="启用简道云"
        @update:model-value="updateEnabled"
      />
    </header>

    <div class="product-center-card__details">
      <p class="product-center-card__row">
        <span class="product-center-card__label">当前版本：</span>
        <span>{{ props.product.versionName }}</span>
        <button
          class="product-center-card__text-button"
          type="button"
          @click="emit('enterProduct')"
        >
          查看
        </button>
      </p>

      <div class="product-center-card__row product-center-card__scope-row">
        <span class="product-center-card__label">可用范围：</span>
        <ElRadioGroup
          class="product-center-card__scope-group"
          :model-value="props.scope"
          :disabled="!props.enabled"
          aria-label="简道云可用范围"
          @update:model-value="selectScope"
        >
          <ElRadio value="all">
            全部成员
            <span class="product-center-card__member-count"
              >已使用：{{ props.product.memberCount }}人</span
            >
          </ElRadio>
          <ElTooltip content="可限制仅部分成员使用该产品" placement="top">
            <RiQuestionFill class="product-center-card__help" aria-label="范围说明" />
          </ElTooltip>
          <ElRadio value="partial">部分成员</ElRadio>
        </ElRadioGroup>
      </div>

      <button
        v-if="props.scope === 'partial'"
        class="product-center-card__selection-field"
        type="button"
        aria-label="编辑可使用简道云的成员和部门"
        @click="emit('editMemberScope')"
      >
        <span
          v-for="selection in props.selections"
          :key="`${selection.type}:${selection.id}`"
          class="product-center-card__selection-tag"
        >
          <RiGroupFill v-if="selection.type === 'department'" aria-hidden="true" />
          <RiUserFill v-else aria-hidden="true" />
          {{ selection.label }}
        </span>
      </button>
    </div>

    <footer class="product-center-card__footer">
      <button
        class="product-center-card__enter-button"
        type="button"
        :disabled="!props.enabled"
        @click="emit('enterProduct')"
      >
        进入产品
        <RiArrowRightLine aria-hidden="true" />
      </button>
    </footer>
  </article>
</template>

<style scoped lang="scss">
.product-center-card {
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
  box-shadow: 0 1px 2px rgb(25 36 53 / 4%);

  &__header {
    display: flex;
    min-height: 96px;
    padding: 24px 22px 10px 26px;
    align-items: flex-start;
    justify-content: space-between;
  }

  &__identity,
  &__scope-row,
  &__scope-group,
  &__enter-button {
    display: flex;
    align-items: center;
  }

  &__identity {
    gap: 18px;
  }

  &__mark {
    position: relative;
    display: block;
    width: 62px;
    height: 46px;
    flex: 0 0 62px;
  }

  &__mark-petal {
    position: absolute;
    display: block;
    width: 32px;
    height: 32px;
    border-radius: 50% 50% 50% 4px;
    transform-origin: 50% 100%;

    &--left {
      top: 10px;
      left: 4px;
      background: #56ddd2;
      transform: rotate(-44deg);
    }

    &--top {
      top: 0;
      left: 19px;
      background: #7ce9df;
      transform: rotate(1deg);
    }

    &--right {
      top: 12px;
      right: 3px;
      background: #18bdb4;
      transform: rotate(47deg);
    }
  }

  &__name {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 600;
    letter-spacing: 0.01em;
    line-height: 30px;
  }

  &__switch {
    margin-top: 2px;
    --el-switch-on-color: var(--el-color-primary);
  }

  &__details {
    padding: 0 32px 24px;
  }

  &__row {
    display: flex;
    min-height: 32px;
    margin: 0;
    align-items: center;
    color: var(--el-text-color-regular);
    font-size: 15px;
    line-height: 24px;
  }

  &__label {
    color: var(--el-text-color-primary);
  }

  &__text-button,
  &__enter-button {
    border: 0;
    cursor: pointer;
  }

  &__text-button {
    margin-left: 14px;
    padding: 3px 8px;
    border-radius: 5px;
    color: var(--el-color-primary);
    background: transparent;
    font-size: 15px;
    line-height: 22px;
    transition: background-color 0.18s ease;

    &:hover {
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__scope-row {
    margin-top: 5px;
  }

  &__scope-group {
    min-height: 32px;
    gap: 8px;
  }

  &__selection-field {
    display: flex;
    width: calc(100% - 110px);
    min-height: 108px;
    margin: 8px 0 0 110px;
    padding: 12px;
    border: 1px dashed var(--el-border-color);
    border-radius: 6px;
    flex-wrap: wrap;
    align-content: flex-start;
    align-items: flex-start;
    gap: 8px;
    background: transparent;
    cursor: pointer;
    transition:
      border-color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      border-color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__selection-tag {
    display: inline-flex;
    min-height: 30px;
    align-items: center;
    gap: 5px;
    padding: 0 10px;
    border-radius: 5px;
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    font-size: 14px;
    line-height: 22px;

    svg {
      width: 16px;
      height: 16px;
      color: var(--el-color-primary);
    }
  }

  &__member-count {
    margin-left: 8px;
    padding: 1px 4px;
    border-radius: 2px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    font-size: 14px;
  }

  &__help {
    width: 18px;
    height: 18px;
    color: var(--el-text-color-secondary);
  }

  &__footer {
    display: flex;
    min-height: 76px;
    margin: 0 22px;
    border-top: 1px solid var(--el-border-color-lighter);
    align-items: center;
  }

  &__enter-button {
    gap: 6px;
    padding: 6px 10px;
    border-radius: 6px;
    color: var(--el-color-primary);
    background: transparent;
    font-size: 16px;
    line-height: 24px;
    transition:
      background-color 0.18s ease,
      color 0.18s ease;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover:not(:disabled) {
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &:disabled {
      color: var(--el-text-color-placeholder);
      cursor: not-allowed;
    }
  }

  &--disabled {
    .product-center-card__mark,
    .product-center-card__details,
    .product-center-card__name {
      opacity: 0.56;
    }
  }
}

@media (max-width: 640px) {
  .product-center-card {
    &__header {
      min-height: auto;
      padding: 20px 18px 14px;
    }

    &__details {
      padding: 0 20px 20px;
    }

    &__scope-row {
      align-items: flex-start;
    }

    &__scope-group {
      flex-wrap: wrap;
      align-items: flex-start;
    }

    &__selection-field {
      width: 100%;
      margin-left: 0;
    }

    &__footer {
      min-height: 66px;
      margin: 0 18px;
    }
  }
}
</style>
