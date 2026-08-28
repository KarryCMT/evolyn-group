<script setup lang="ts">
import { RiCloseFill } from '@remixicon/vue';
import { computed, nextTick, onBeforeUnmount, ref, useAttrs, useTemplateRef, watch } from 'vue';
import EvolynButton from '../EvolynButton/Button.vue';
import EvolynScrollbar from '../EvolynScrollbar/EvolynScrollbar.vue';
import type {
  EvolynDialogEmits,
  EvolynDialogProps,
  EvolynDialogTransition,
} from './EvolynDialog.types';

defineOptions({ name: 'EvolynDialog', inheritAttrs: false });

const props = withDefaults(defineProps<EvolynDialogProps>(), {
  appendTo: 'body',
  cancelButtonText: '取消',
  closeDelay: 0,
  closeOnClickModal: true,
  closeOnPressEscape: true,
  confirmButtonText: '确定',
  confirmDisabled: false,
  confirmLoading: false,
  lockScroll: true,
  modal: true,
  openDelay: 0,
  showCancelButton: true,
  showClose: true,
  showFooter: false,
  title: '',
  transition: 'evolyn-dialog-fade',
  trapFocus: true,
});
const emit = defineEmits<EvolynDialogEmits>();
const attrs = useAttrs();
const panel = useTemplateRef<HTMLElement>('panel');
const rendered = ref(props.modelValue);
const displayed = ref(props.modelValue);
const dragOffset = ref({ x: 0, y: 0 });
const dragStart = ref<{ x: number; y: number }>();
const previousActiveElement = ref<HTMLElement>();
let openTimer: ReturnType<typeof setTimeout> | undefined;
let closeTimer: ReturnType<typeof setTimeout> | undefined;
let previousBodyOverflow = '';

const teleportTarget = computed(() => (props.appendToBody ? 'body' : props.appendTo));
const scrollbarHeight = computed(() =>
  typeof props.bodyHeight === 'number' ? `${props.bodyHeight}px` : props.bodyHeight,
);
const dialogStyle = computed(() => ({
  marginTop: props.alignCenter || props.fullscreen ? undefined : (props.top ?? '15vh'),
  transform: `translate(${dragOffset.value.x}px, ${dragOffset.value.y}px)`,
  width: props.fullscreen ? undefined : toCssSize(props.width),
  zIndex: props.zIndex,
}));
const transitionBindings = computed(() => toTransitionBindings(props.transition));

function toCssSize(value: number | string | undefined) {
  return typeof value === 'number' ? `${value}px` : value;
}

function toTransitionBindings(transition: EvolynDialogTransition) {
  return typeof transition === 'string' ? { name: transition } : transition;
}

function setVisible(value: boolean) {
  emit('update:modelValue', value);
}

function requestClose() {
  const done = (cancel?: boolean) => {
    if (!cancel) setVisible(false);
  };
  if (props.beforeClose) props.beforeClose(done);
  else done();
}

function handleCancel() {
  emit('cancel');
  requestClose();
}

function handleConfirm() {
  if (!props.confirmDisabled && !props.confirmLoading) emit('confirm');
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.closeOnPressEscape) requestClose();
}

function startDrag(event: PointerEvent) {
  if (!props.draggable || props.fullscreen || event.button !== 0) return;
  dragStart.value = {
    x: event.clientX - dragOffset.value.x,
    y: event.clientY - dragOffset.value.y,
  };
  window.addEventListener('pointermove', moveDrag);
  window.addEventListener('pointerup', stopDrag, { once: true });
}

function moveDrag(event: PointerEvent) {
  if (!dragStart.value) return;
  const next = { x: event.clientX - dragStart.value.x, y: event.clientY - dragStart.value.y };
  if (props.overflow || !panel.value) {
    dragOffset.value = next;
    return;
  }
  const rect = panel.value.getBoundingClientRect();
  dragOffset.value = {
    x: Math.min(Math.max(next.x, -rect.left), window.innerWidth - rect.right + dragOffset.value.x),
    y: Math.min(Math.max(next.y, -rect.top), window.innerHeight - rect.bottom + dragOffset.value.y),
  };
}

function stopDrag() {
  dragStart.value = undefined;
  window.removeEventListener('pointermove', moveDrag);
}

function resetPosition() {
  dragOffset.value = { x: 0, y: 0 };
}

function lockBodyScroll() {
  if (!props.lockScroll || typeof document === 'undefined') return;
  previousBodyOverflow = document.body.style.overflow;
  document.body.style.overflow = 'hidden';
}

function unlockBodyScroll() {
  if (!props.lockScroll || typeof document === 'undefined') return;
  document.body.style.overflow = previousBodyOverflow;
}

function handleAfterLeave() {
  if (props.destroyOnClose) rendered.value = false;
  previousActiveElement.value?.focus();
  emit('close-auto-focus');
  emit('closed');
}

watch(
  () => props.modelValue,
  (visible) => {
    clearTimeout(openTimer);
    clearTimeout(closeTimer);
    if (visible) {
      rendered.value = true;
      previousActiveElement.value =
        document.activeElement instanceof HTMLElement ? document.activeElement : undefined;
      openTimer = setTimeout(async () => {
        displayed.value = true;
        lockBodyScroll();
        emit('open');
        await nextTick();
        if (props.trapFocus) panel.value?.focus();
        emit('open-auto-focus');
      }, props.openDelay);
      return;
    }
    closeTimer = setTimeout(() => {
      if (!displayed.value) return;
      displayed.value = false;
      unlockBodyScroll();
      emit('close');
    }, props.closeDelay);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  clearTimeout(openTimer);
  clearTimeout(closeTimer);
  stopDrag();
  if (displayed.value) unlockBodyScroll();
});

defineExpose({ handleClose: requestClose, resetPosition });
</script>

<template>
  <Teleport v-if="rendered" :to="teleportTarget">
    <Transition
      v-bind="transitionBindings"
      @after-enter="emit('opened')"
      @after-leave="handleAfterLeave"
    >
      <div
        v-show="displayed"
        class="evolyn-dialog-wrapper"
        :class="{ 'evolyn-dialog-wrapper--align-center': props.alignCenter }"
        :style="{ zIndex: props.zIndex }"
        @keydown="handleKeydown"
      >
        <div
          v-if="props.modal"
          class="evolyn-dialog-wrapper__mask"
          :class="[props.modalClass, { 'is-penetrable': props.modalPenetrable }]"
          @click.self="props.closeOnClickModal && requestClose()"
        />
        <section
          ref="panel"
          class="evolyn-dialog"
          :class="{
            'evolyn-dialog--center': props.center,
            'evolyn-dialog--fullscreen': props.fullscreen,
          }"
          :style="dialogStyle"
          role="dialog"
          aria-modal="true"
          tabindex="-1"
          v-bind="attrs"
        >
          <header class="evolyn-dialog__header" :class="props.headerClass" @pointerdown="startDrag">
            <slot
              name="header"
              :close="requestClose"
              title-id="evolyn-dialog-title"
              title-class="evolyn-dialog__title"
            >
              <h2 id="evolyn-dialog-title" class="evolyn-dialog__title">{{ props.title }}</h2>
            </slot>
            <button
              v-if="props.showClose"
              class="evolyn-dialog__close"
              type="button"
              aria-label="关闭弹窗"
              @pointerdown.stop
              @click="requestClose"
            >
              <component :is="props.closeIcon ?? RiCloseFill" aria-hidden="true" />
            </button>
          </header>

          <!-- 内容区域统一使用 EvolynScrollbar，业务组件无需自行维护滚动条。 -->
          <EvolynScrollbar
            class="evolyn-dialog__body"
            :class="props.bodyClass"
            :height="scrollbarHeight"
          >
            <slot />
          </EvolynScrollbar>

          <footer
            v-if="$slots.footer || props.showFooter"
            class="evolyn-dialog__footer"
            :class="props.footerClass"
          >
            <slot name="footer">
              <div class="evolyn-dialog__footer-actions">
                <EvolynButton v-if="props.showCancelButton" @click="handleCancel">
                  {{ props.cancelButtonText }}
                </EvolynButton>
                <EvolynButton
                  type="primary"
                  :disabled="props.confirmDisabled || props.confirmLoading"
                  @click="handleConfirm"
                >
                  {{ props.confirmButtonText }}
                </EvolynButton>
              </div>
            </slot>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style lang="scss">
.evolyn-dialog-wrapper {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  pointer-events: none;

  &--align-center {
    align-items: center;
  }

  &__mask {
    position: absolute;
    inset: 0;
    pointer-events: auto;
    background: color-mix(in srgb, var(--el-color-black) 50%, var(--el-color-transparent));

    &.is-penetrable {
      pointer-events: none;
    }
  }
}

.evolyn-dialog {
  position: relative;
  box-sizing: border-box;
  width: 50%;
  max-width: calc(100vw - var(--el-space-6xl));
  max-height: calc(100vh - var(--el-space-6xl));
  color: var(--el-text-color-primary);
  pointer-events: auto;
  background: var(--el-bg-color-overlay);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow);
  outline: none;

  &--center .evolyn-dialog__header,
  &--center .evolyn-dialog__footer {
    justify-content: center;
    text-align: center;
  }
  &--fullscreen {
    width: 100%;
    max-width: 100%;
    height: 100%;
    max-height: 100%;
    margin: 0;
    border-radius: 0;
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 56px;
    padding: 0 var(--el-space-3xl);
    border-bottom: 1px solid var(--el-border-color-light);
  }
  &__title {
    margin: 0;
    font-size: var(--el-font-size-large);
    font-weight: 600;
    line-height: 26px;
  }
  &__close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    padding: 0;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
  }
  &__close:hover {
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
  }
  &__close svg {
    width: var(--el-font-size-medium);
    height: var(--el-font-size-medium);
  }
  &__body {
    max-height: calc(100vh - 240px);
    padding: var(--el-space-3xl);
  }
  &__footer {
    padding: var(--el-space-xl) var(--el-space-3xl) var(--el-space-3xl);
  }
  &__footer-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--el-space-lg);
  }
}

.evolyn-dialog-fade-enter-active,
.evolyn-dialog-fade-leave-active {
  transition: opacity 0.2s ease;
}
.evolyn-dialog-fade-enter-from,
.evolyn-dialog-fade-leave-to {
  opacity: 0;
}
</style>
