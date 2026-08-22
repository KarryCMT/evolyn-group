<script setup lang="ts">
import { RiSettings3Fill } from '@remixicon/vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import avatar from '~/assets/images/avatar.jpg';

defineOptions({ name: 'GreetingWidget' });
const props = withDefaults(
  defineProps<{
    widget: DashboardWidgetContent;
    /** 仅在工作台编辑器中显示头像配置提示。 */
    editorMode?: boolean;
  }>(),
  { editorMode: false },
);
</script>

<template>
  <section class="greeting-widget" :class="{ 'greeting-widget--editor': props.editorMode }">
    <div class="greeting-widget__avatar">
      <img class="greeting-widget__avatar-image" :src="avatar" alt="李同学的头像" />
      <span class="greeting-widget__avatar-settings" aria-hidden="true">
        <RiSettings3Fill />
      </span>
    </div>
    <div class="greeting-widget__content">
      <strong>李同学</strong>
      <span>下午好！</span>
    </div>
  </section>
</template>

<style scoped lang="scss">
.greeting-widget {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  /* 卡片高度跟随网格内容区，头像不会因网格边距被裁切。 */
  height: 100%;
  /* 与工作台卡片左侧保持 16px 的视觉留白。 */
  padding: 0 0 0 16px;
  overflow: hidden;
  gap: 16px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border: 0;
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-lighter);

  &__avatar {
    position: relative;
    flex: 0 0 40px;
    width: 40px;
    height: 40px;
    overflow: hidden;
    border-radius: 8px;
  }

  &__avatar-image {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 8px;
  }

  &__avatar-settings {
    position: absolute;
    top: 50%;
    left: 50%;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    color: var(--el-color-white);
    background: rgba(75, 85, 99, 0.5);
    border-radius: var(--el-border-radius-base);
    opacity: 0;
    pointer-events: none;
    transform: translate(-50%, -50%);
    transition: opacity var(--el-transition-duration-fast);

    svg {
      width: 14px;
      height: 14px;
    }
  }

  &--editor:hover &__avatar-settings {
    opacity: 1;
  }

  &__content {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: var(--el-font-size-base);

    strong {
      font-size: var(--el-font-size-medium);
    }
  }
}
</style>
