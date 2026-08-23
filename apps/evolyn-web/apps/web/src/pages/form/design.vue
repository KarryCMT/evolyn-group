<script setup lang="ts">
import { RiEyeFill, RiLightbulbFlashFill, RiSave3Fill, RiShareForwardFill } from '@remixicon/vue';
import { FormDesigner, type FormFieldPreset } from '@evolyn.do/form';
import { ElMessage } from 'element-plus';

defineOptions({ name: 'FormDesignPage' });

/** 保存、预览和字段添加依赖后续设计器内核，本期保留明确的交互反馈。 */
function notifyUnavailable(action: string) {
  ElMessage.info(`${action}将在表单设计器接入后提供`);
}

function handleFieldSelect(field: FormFieldPreset) {
  notifyUnavailable(`添加${field.title}`);
}
</script>

<template>
  <section class="form-design-page" aria-label="表单设计工作台">
    <div class="form-design-page__toolbar" aria-label="表单设计操作">
      <button
        class="form-design-page__guide-button"
        type="button"
        @click="notifyUnavailable('新手引导')"
      >
        <RiLightbulbFlashFill />
        <span class="form-design-page__guide-label">查看新手引导</span>
      </button>
      <div class="form-design-page__toolbar-actions">
        <button
          class="form-design-page__action-button form-design-page__action-button--secondary"
          type="button"
          @click="notifyUnavailable('预览')"
        >
          <RiEyeFill />
          <span class="form-design-page__action-label">预览</span>
        </button>
        <button
          class="form-design-page__action-button form-design-page__action-button--primary"
          type="button"
          @click="notifyUnavailable('保存')"
        >
          <RiSave3Fill />
          <span class="form-design-page__action-label">保存</span>
        </button>
        <button
          class="form-design-page__icon-button form-design-page__share-button"
          type="button"
          aria-label="分享表单"
          @click="notifyUnavailable('分享')"
        >
          <RiShareForwardFill />
        </button>
      </div>
    </div>

    <FormDesigner
      class="form-design-page__workspace"
      @select-field="handleFieldSelect"
      @open-recycle-bin="notifyUnavailable('字段回收站')"
    >
      <template #canvas>
        <section class="form-design-page__canvas-placeholder" aria-label="待接入的表单设计器" />
      </template>
    </FormDesigner>
  </section>
</template>

<style scoped lang="scss">
.form-design-page {
  display: flex;
  min-height: 0;
  margin: 0 8px 8px;
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  box-shadow: var(--el-box-shadow-light);

  &__toolbar,
  &__toolbar-actions,
  &__guide-button,
  &__action-button {
    display: flex;
    align-items: center;
  }

  &__toolbar {
    height: 50px;
    min-height: 50px;
    padding: 0 16px 0 24px;
    justify-content: space-between;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__guide-button,
  &__action-button,
  &__icon-button {
    border: 0;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__guide-button,
  &__action-button {
    justify-content: center;
    gap: 7px;
    font-size: 14px;
    font-weight: 600;
  }

  &__guide-button {
    height: 32px;
    padding: 0 10px;
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 18px;
      height: 18px;
      color: var(--el-color-primary);
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__toolbar-actions {
    gap: 10px;
  }

  &__action-button {
    min-width: 76px;
    height: 32px;
    padding: 0 12px;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 17px;
      height: 17px;
    }

    &--secondary {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      border: 1px solid var(--el-color-primary);

      &:hover {
        background: var(--el-color-primary-light-9);
      }
    }

    &--primary {
      color: var(--el-color-white);
      background: var(--el-color-primary);

      &:hover {
        background: var(--el-color-primary-light-3);
      }
    }
  }

  &__icon-button {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 20px;
      height: 20px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__share-button {
    border: 1px solid var(--el-border-color);

    &:hover {
      border-color: var(--el-color-primary);
    }
  }

  &__workspace {
    min-height: 0;
    flex: 1;
  }

  &__canvas-placeholder {
    width: 100%;
    min-height: 100%;
  }
}

@media (max-width: 620px) {
  .form-design-page {
    margin: 0 4px 4px;
    border-radius: 10px;

    &__toolbar {
      padding: 0 10px 0 12px;
    }

    &__guide-button {
      padding: 0 4px;
    }

    &__guide-label,
    &__action-label {
      display: none;
    }

    &__toolbar-actions {
      gap: 6px;
    }

    &__action-button {
      min-width: 34px;
      padding: 0 8px;
    }
  }
}
</style>
