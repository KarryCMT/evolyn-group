<script setup lang="ts">
import { CircleCheck, DocumentAdd, User } from '@element-plus/icons-vue';
import type { DashboardWidgetContent } from '~/types/dashboard';

defineOptions({ name: 'OnboardingWidget' });
defineProps<{ widget: DashboardWidgetContent }>();

const steps = [
  { label: '了解产品', icon: CircleCheck },
  { label: '探索应用', icon: DocumentAdd },
  { label: '邀请成员', icon: User },
];
</script>

<template>
  <section class="onboarding-widget">
    <span class="onboarding-widget__tag">新手引导</span>
    <div class="onboarding-widget__steps">
      <el-button v-for="step in steps" :key="step.label" text :icon="step.icon">{{ step.label }}</el-button>
    </div>
  </section>
</template>

<style scoped lang="scss">
.onboarding-widget {
  position: relative;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 16px;
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: var(--el-border-radius-base);

  &__tag {
    position: absolute;
    top: 12px;
    left: 16px;
    padding: 4px 8px;
    color: var(--el-color-white);
    font-size: var(--el-font-size-small);
    background: var(--el-color-primary);
    border-radius: var(--el-border-radius-round);
  }

  &__steps {
    display: flex;
    gap: 48px;
  }
}
</style>
