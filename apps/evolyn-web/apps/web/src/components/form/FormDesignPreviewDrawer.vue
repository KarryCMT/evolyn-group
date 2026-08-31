<script setup lang="ts">
import type { FormRuntimeActionDefinition, FormRuntimeAdapter } from '@evolyn.do/form/runtime-web';
import type { FormSchemaDocument } from '@evolyn.do/form/schema';
import { FormWebRuntimeSurface } from '@evolyn.do/form/runtime-web';
import { RiCloseFill, RiComputerFill, RiSmartphoneFill } from '@remixicon/vue';
import { ref, shallowRef, watch } from 'vue';
import { getMemberFieldRegistry } from './memberFieldRegistry';
// 预览组件独立加载运行时样式，避免宿主页面依赖设计器样式副作用。
import '@evolyn.do/form/runtime-web/style.css';

defineOptions({ name: 'FormDesignPreviewDrawer' });

const props = defineProps<{
  schema: FormSchemaDocument;
  formId: string;
  adapter: FormRuntimeAdapter;
}>();

const emit = defineEmits<{
  unsupportedField: [info: { fieldKey: string; type: string }];
  submitSuccess: [];
  draftSuccess: [];
}>();

/** 抽屉显隐由设计页持有，组件通过标准 v-model 契约回传关闭状态。 */
const visible = defineModel<boolean>({ default: false });
/** 终端视口仅影响预览组件内部布局，不进入表单草稿协议。 */
const viewport = shallowRef<'desktop' | 'mobile'>('desktop');
/**
 * 运行时按不可变 Schema 创建填写会话，设计器草稿却会在原对象上原地修改。
 * 因此只在预览中以版本键重挂载运行时，使属性编辑立即反映；正式填写页仍维持
 * 已发布 Schema 的稳定会话，编辑中的填写值不受影响。
 */
const runtimeSessionVersion = ref(0);
watch(
  () => props.schema,
  () => {
    runtimeSessionVersion.value += 1;
  },
  { deep: true },
);
const previewActions: FormRuntimeActionDefinition[] = [
  {
    key: 'save-draft',
    label: '保存草稿',
    behavior: 'save-draft',
    intent: 'secondary',
    order: 10,
    mobilePresentation: 'compact',
  },
  {
    key: 'submit',
    label: '提交',
    behavior: 'submit',
    intent: 'primary',
    order: 100,
    mobilePresentation: 'button',
  },
];

function setViewport(value: 'desktop' | 'mobile'): void {
  viewport.value = value;
}
</script>

<template>
  <el-drawer
    v-model="visible"
    append-to-body
    direction="btt"
    size="calc(100% - var(--el-space-5xl))"
    title="表单预览"
    :show-close="false"
    body-class="form-design-preview-drawer__body"
    header-class="form-design-preview-drawer__header"
    class="form-design-preview-drawer"
  >
    <template #header="{ close: closeDrawer }">
      <div class="form-design-preview__header-content">
        <div class="form-design-preview__header-spacer" aria-hidden="true" />
        <div class="form-design-preview__viewport-switch" role="group" aria-label="预览设备">
          <button
            class="form-design-preview__viewport-button"
            :class="{
              'form-design-preview__viewport-button--active': viewport === 'desktop',
            }"
            type="button"
            :aria-pressed="viewport === 'desktop'"
            @click="setViewport('desktop')"
          >
            <RiComputerFill />
            <span>桌面端</span>
          </button>
          <button
            class="form-design-preview__viewport-button"
            :class="{
              'form-design-preview__viewport-button--active': viewport === 'mobile',
            }"
            type="button"
            :aria-pressed="viewport === 'mobile'"
            @click="setViewport('mobile')"
          >
            <RiSmartphoneFill />
            <span>移动端</span>
          </button>
        </div>
        <button
          class="form-design-preview__close"
          type="button"
          aria-label="关闭预览"
          @click="closeDrawer"
        >
          <RiCloseFill />
        </button>
      </div>
    </template>
    <section class="form-design-preview" aria-label="表单预览内容">
      <div class="form-design-preview__stage" :class="`form-design-preview__stage--${viewport}`">
        <FormWebRuntimeSurface
          :key="runtimeSessionVersion"
          class="form-design-preview__runtime"
          :schema="schema"
          :form-id="formId"
          :adapter="adapter"
          :registry="getMemberFieldRegistry()"
          :actions="previewActions"
          :layout="viewport"
          content-width="100%"
          @unsupported-field="emit('unsupportedField', $event)"
          @submit-success="emit('submitSuccess')"
          @draft-success="emit('draftSuccess')"
        />
      </div>
    </section>
  </el-drawer>
</template>

<style lang="scss">
// Drawer 传送到 body，使用唯一块类约束局部覆盖，避免影响其他抽屉。
.form-design-preview-drawer {
  border-radius: var(--el-border-radius-large) var(--el-border-radius-large) 0 0;
}

.form-design-preview-drawer__body {
  display: flex;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.form-design-preview-drawer__header {
  display: flex;
  align-items: center;
  height: 56px;
  min-height: 56px;
  padding: 0 var(--el-space-3xl);
  margin: 0;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

@media (width <= 620px) {
  .form-design-preview-drawer__header {
    height: 52px;
    min-height: 52px;
    padding: 0 var(--el-space-lg);
  }
}
</style>

<style scoped lang="scss">
.form-design-preview {
  // 抽屉 body 是 flex 容器，根节点必须主动铺满，避免按运行时内容宽度收缩。
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  height: 100%;
  min-height: 0;
  background: var(--el-bg-color-page);

  &__header-content {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    width: 100%;
  }

  &__viewport-switch {
    display: inline-flex;
    gap: var(--el-space-xs);
    padding: var(--el-space-xs);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);
  }

  &__viewport-button {
    display: inline-flex;
    gap: var(--el-space-sm);
    align-items: center;
    justify-content: center;
    height: 28px;
    padding: 0 var(--el-space-lg);
    font-size: var(--el-font-size-small);
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-small);

    svg {
      width: 16px;
      height: 16px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &--active {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      box-shadow: var(--el-box-shadow-lighter);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    padding: 0;
    margin-left: auto;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 22px;
      height: 22px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__stage {
    box-sizing: border-box;
    position: relative;
    width: 780px;
    max-width: 100%;
    height: 100%;
    min-height: 0;
    padding: 20px 0 0;
    margin: 0 auto;
    overflow: hidden;

    &--mobile {
      // 设计器中的移动端预览固定为常用手机视口，运行时内容区自行滚动。
      width: 286px;
      height: 599px;
      padding: 0;
      margin-block: var(--el-space-6xl) 0;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: var(--el-border-radius-large);
      box-shadow: var(--el-box-shadow-light);
    }
  }

  &__runtime {
    min-height: 0;
  }
}

@media (width <= 620px) {
  .form-design-preview {
    &__viewport-button {
      padding: 0 var(--el-space-md);

      span {
        display: none;
      }
    }

    &__stage {
      &--desktop,
      &--mobile {
        width: 100%;
        max-width: 100%;
        height: 100%;
        margin: 0;
        border: 0;
        border-radius: 0;
        box-shadow: none;
      }
    }
  }
}
</style>
