<script setup lang="ts">
import { RiAddLine, RiDeleteBin6Line, RiEditLine, RiInformationLine } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import { ElIcon, ElTooltip } from 'element-plus';
import type { FormItem } from '../schema/types';
import type { SubmitValidatorDraft } from './submit-validation-types';
import FormSchemaSubmitValidatorDialog from './FormSchemaSubmitValidatorDialog.vue';

const validators = defineModel<SubmitValidatorDraft[]>({ required: true });
const props = defineProps<{ items: FormItem[] }>();

const dialogOpen = shallowRef(false);
const editingIndex = shallowRef<number | null>(null);

const configuredCountLabel = computed(() =>
  validators.value.length === 0 ? '添加校验条件' : `已配置 ${validators.value.length} 条校验`,
);

const validatorSummary = computed(() => {
  const first = validators.value[0];
  if (!first) return '';
  return first.remind || first.formula || '未命名校验条件';
});

function openCreate(): void {
  editingIndex.value = null;
  dialogOpen.value = true;
}

function openEdit(): void {
  if (validators.value.length === 0) return;
  editingIndex.value = 0;
  dialogOpen.value = true;
}

function saveValidator(next: SubmitValidatorDraft): void {
  const snapshot = structuredClone(next);
  if (editingIndex.value === null) {
    validators.value = [...validators.value, snapshot];
    return;
  }
  validators.value = validators.value.map((validator, index) =>
    index === editingIndex.value ? snapshot : validator,
  );
}

function removeFirstValidator(): void {
  if (validators.value.length > 0) validators.value = validators.value.slice(1);
}
</script>

<template>
  <section class="form-submit-validation-settings" aria-label="提交时校验数据">
    <div class="form-submit-validation-settings__heading">
      <span class="form-submit-validation-settings__title">校验数据</span>
      <el-tooltip content="提交前按公式校验多个字段的业务条件" placement="top">
        <el-icon class="form-submit-validation-settings__help" aria-label="校验数据说明">
          <RiInformationLine />
        </el-icon>
      </el-tooltip>
    </div>

    <button
      v-if="validators.length === 0"
      class="form-submit-validation-settings__entry"
      type="button"
      @click="openCreate"
    >
      <span>{{ configuredCountLabel }}</span>
      <span class="form-submit-validation-settings__add" aria-hidden="true">
        <el-icon><RiAddLine /></el-icon>
      </span>
    </button>

    <div v-else class="form-submit-validation-settings__configured">
      <button
        type="button"
        class="form-submit-validation-settings__summary"
        :title="validatorSummary"
        @click="openEdit"
      >
        <span class="form-submit-validation-settings__summary-copy">
          <strong>{{ configuredCountLabel }}</strong>
          <small>{{ validatorSummary }}</small>
        </span>
        <el-icon><RiEditLine /></el-icon>
      </button>
      <button
        type="button"
        class="form-submit-validation-settings__icon-button"
        aria-label="新增校验条件"
        @click="openCreate"
      >
        <el-icon><RiAddLine /></el-icon>
      </button>
      <button
        type="button"
        class="form-submit-validation-settings__icon-button form-submit-validation-settings__icon-button--danger"
        aria-label="删除首条校验条件"
        @click="removeFirstValidator"
      >
        <el-icon><RiDeleteBin6Line /></el-icon>
      </button>
    </div>

    <FormSchemaSubmitValidatorDialog
      v-model="dialogOpen"
      :items="props.items"
      :validator="editingIndex === null ? undefined : validators[editingIndex]"
      @save="saveValidator"
    />
  </section>
</template>

<style scoped lang="scss">
.form-submit-validation-settings {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;

  &__heading {
    display: inline-flex;
    gap: 6px;
    align-items: center;
  }

  &__title {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__help {
    font-size: 16px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  &__entry {
    display: flex;
    width: 100%;
    height: 42px;
    padding: 0 8px 0 12px;
    align-items: center;
    justify-content: space-between;
    font: inherit;
    font-size: 14px;
    color: var(--el-text-color-primary);
    text-align: left;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
    transition: border-color 0.2s, background-color 0.2s;

    &:hover,
    &:focus-visible {
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary);
      outline: none;
    }
  }

  &__add,
  &__icon-button {
    display: inline-flex;
    width: 30px;
    height: 30px;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    border: 0;
    border-radius: 8px;
  }

  &__configured {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 30px 30px;
    gap: 6px;
  }

  &__summary {
    display: flex;
    min-width: 0;
    height: 52px;
    padding: 8px 10px;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-primary);
    text-align: left;
    cursor: pointer;
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-7);
    border-radius: var(--el-border-radius-base);

    &:hover,
    &:focus-visible {
      border-color: var(--el-color-primary);
      outline: none;
    }
  }

  &__summary-copy {
    display: grid;
    min-width: 0;
    gap: 2px;

    strong,
    small {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong { font-size: 13px; font-weight: 600; }
    small { font-size: 12px; color: var(--el-text-color-secondary); }
  }

  &__icon-button {
    align-self: center;
    padding: 0;
    cursor: pointer;

    &:hover,
    &:focus-visible { color: var(--el-color-primary); outline: 2px solid var(--el-color-primary-light-7); }

    &--danger:hover,
    &--danger:focus-visible { color: var(--el-color-danger); outline-color: var(--el-color-danger-light-7); }
  }
}
</style>
