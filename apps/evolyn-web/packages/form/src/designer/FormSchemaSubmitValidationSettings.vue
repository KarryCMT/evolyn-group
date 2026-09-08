<script setup lang="ts">
import {
  RiAddLine,
  RiDeleteBin6Line,
  RiDragMoveLine,
  RiEditLine,
  RiFileCopyLine,
  RiInformationLine,
} from '@remixicon/vue';
import { computed, ref, shallowRef, watch } from 'vue';
import Draggable from 'vuedraggable';
import { ElIcon, ElMessageBox, ElTooltip } from 'element-plus';
import type { FormItem } from '../schema/types';
import { cloneSubmitValidatorDraft, type SubmitValidatorDraft } from './submit-validation-types';
import FormSchemaSubmitValidatorDialog from './FormSchemaSubmitValidatorDialog.vue';

const validators = defineModel<SubmitValidatorDraft[]>({ required: true });
const props = defineProps<{ items: FormItem[] }>();

const dialogOpen = shallowRef(false);
const editingIndex = shallowRef<number | null>(null);
const localValidators = ref<SubmitValidatorDraft[]>([]);
const dragKeys = new WeakMap<object, string>();
let nextDragKey = 0;

const configuredCountLabel = computed(() =>
  validators.value.length === 0 ? '添加校验条件' : `已配置 ${validators.value.length} 条校验`,
);
const canCreate = computed(() => validators.value.length < 50);

watch(
  validators,
  (next) => {
    localValidators.value = next.map(cloneSubmitValidatorDraft);
  },
  { immediate: true, deep: true },
);

function openCreate(): void {
  editingIndex.value = null;
  dialogOpen.value = true;
}

function openEdit(index: number): void {
  editingIndex.value = index;
  dialogOpen.value = true;
}

function saveValidator(next: SubmitValidatorDraft): void {
  const snapshot = cloneSubmitValidatorDraft(next);
  if (editingIndex.value === null) {
    validators.value = [...validators.value, snapshot];
    return;
  }
  validators.value = validators.value.map((validator, index) =>
    index === editingIndex.value ? snapshot : validator,
  );
}

function duplicateValidator(index: number): void {
  const source = validators.value[index];
  if (!source || !canCreate.value) return;
  validators.value = [
    ...validators.value.slice(0, index + 1),
    cloneSubmitValidatorDraft(source),
    ...validators.value.slice(index + 1),
  ];
}

async function removeValidator(index: number): Promise<void> {
  const source = validators.value[index];
  if (!source) return;
  try {
    await ElMessageBox.confirm(
      `删除后将不再执行「${source.remind || source.formula}」校验，是否继续？`,
      '删除校验条件',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    );
  } catch {
    return;
  }
  validators.value = validators.value.filter((_, currentIndex) => currentIndex !== index);
}

function emitReorder(): void {
  validators.value = localValidators.value.map(cloneSubmitValidatorDraft);
}

function summary(validator: SubmitValidatorDraft): string {
  return validator.remind || validator.formula || '未命名校验条件';
}

/** 拖拽键只存在于 UI 内存，永不写入 validators 协议数组。 */
function validatorKey(validator: SubmitValidatorDraft): string {
  let key = dragKeys.get(validator);
  if (!key) {
    nextDragKey += 1;
    key = `submit-validator-${nextDragKey}`;
    dragKeys.set(validator, key);
  }
  return key;
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
      :disabled="!canCreate"
      @click="openCreate"
    >
      <span>{{ configuredCountLabel }}</span>
      <span class="form-submit-validation-settings__add" aria-hidden="true">
        <el-icon><RiAddLine /></el-icon>
      </span>
    </button>

    <div v-else class="form-submit-validation-settings__configured">
      <div class="form-submit-validation-settings__configured-head">
        <strong>{{ configuredCountLabel }}</strong>
        <button
          type="button"
          class="form-submit-validation-settings__icon-button"
          aria-label="新增校验条件"
          :disabled="!canCreate"
          @click="openCreate"
        >
          <el-icon><RiAddLine /></el-icon>
        </button>
      </div>
      <Draggable
        :list="localValidators"
        :item-key="validatorKey"
        handle=".form-submit-validation-settings__drag"
        :animation="150"
        @end="emitReorder"
      >
        <template #item="{ element, index }">
          <article class="form-submit-validation-settings__summary">
            <el-icon class="form-submit-validation-settings__drag" aria-label="拖拽排序">
              <RiDragMoveLine />
            </el-icon>
            <button
              type="button"
              class="form-submit-validation-settings__summary-copy"
              :title="summary(element)"
              @click="openEdit(index)"
            >
              <strong>{{ element.failAction === 0 ? '阻止提交' : '可忽略告警' }}</strong>
              <small>{{ summary(element) }}</small>
            </button>
            <div class="form-submit-validation-settings__row-actions">
              <button type="button" aria-label="编辑校验条件" @click="openEdit(index)">
                <el-icon><RiEditLine /></el-icon>
              </button>
              <button
                type="button"
                aria-label="复制校验条件"
                :disabled="!canCreate"
                @click="duplicateValidator(index)"
              >
                <el-icon><RiFileCopyLine /></el-icon>
              </button>
              <button
                type="button"
                class="is-danger"
                aria-label="删除校验条件"
                @click="removeValidator(index)"
              >
                <el-icon><RiDeleteBin6Line /></el-icon>
              </button>
            </div>
          </article>
        </template>
      </Draggable>
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
    transition:
      border-color 0.2s,
      background-color 0.2s;

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
    gap: 8px;
  }

  &__configured-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 13px;
  }

  &__summary {
    display: flex;
    min-width: 0;
    min-height: 52px;
    padding: 8px 10px;
    align-items: center;
    gap: 8px;
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
    flex: 1;
    min-width: 0;
    padding: 0;
    color: inherit;
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;
    gap: 2px;

    strong,
    small {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 13px;
      font-weight: 600;
    }
    small {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  &__icon-button {
    align-self: center;
    padding: 0;
    cursor: pointer;

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary);
      outline: 2px solid var(--el-color-primary-light-7);
    }

    &--danger:hover,
    &--danger:focus-visible {
      color: var(--el-color-danger);
      outline-color: var(--el-color-danger-light-7);
    }
  }

  &__drag {
    flex: 0 0 auto;
    color: var(--el-text-color-secondary);
    cursor: grab;
  }

  &__row-actions {
    display: inline-flex;
    flex: 0 0 auto;
    gap: 2px;

    button {
      display: inline-grid;
      width: 26px;
      height: 26px;
      padding: 0;
      place-items: center;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      background: transparent;
      border: 0;
      border-radius: 5px;

      &:hover,
      &:focus-visible {
        color: var(--el-color-primary);
        background: var(--el-bg-color);
        outline: none;
      }

      &.is-danger:hover,
      &.is-danger:focus-visible {
        color: var(--el-color-danger);
      }
    }
  }
}
</style>
