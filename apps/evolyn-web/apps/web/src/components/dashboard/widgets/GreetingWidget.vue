<script setup lang="ts">
import type { DashboardWidgetContent } from '~/types/dashboard';
import avatar from '~/assets/images/avatar.jpg';

defineOptions({ name: 'GreetingWidget' });
defineProps<{ widget: DashboardWidgetContent }>();
</script>

<template>
  <section class="greeting-widget">
    <img class="greeting-widget__avatar" :src="avatar" alt="李同学的头像" />
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
  height: 100%;
  padding: 0;
  gap: 12px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-lighter);

  &__avatar {
    flex: 0 0 64px;
    width: 64px;
    height: 100%;
    object-fit: cover;
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
