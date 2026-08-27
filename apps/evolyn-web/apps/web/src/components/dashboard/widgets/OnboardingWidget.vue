<script setup lang="ts">
import { computed } from 'vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import bannerBackground from '~/assets/images/banner_bg.png';
import exploreIcon from '~/assets/images/icon3.png';
import inviteIcon from '~/assets/images/icon1.png';
import learnIcon from '~/assets/images/icon4.png';
import createIcon from '~/assets/images/icon2.png';

defineOptions({ name: 'OnboardingWidget' });
const props = defineProps<{ widget: DashboardWidgetContent }>();
const variant = computed(() => props.widget.config?.variant ?? 'guide');

const steps = [
  { label: '了解产品', icon: learnIcon },
  { label: '探索应用', icon: exploreIcon },
  { label: '创建流程/表单/仪表盘', icon: createIcon },
  { label: '邀请成员', icon: inviteIcon },
];
</script>

<template>
  <section class="onboarding-widget" :style="{ backgroundImage: `url(${bannerBackground})` }">
    <span class="onboarding-widget__tag">{{
      variant === 'guide' ? props.widget.title : '自定义组件'
    }}</span>
    <div v-if="variant === 'guide'" class="onboarding-widget__steps">
      <div v-for="(step, index) in steps" :key="step.label" class="onboarding-widget__step">
        <img class="onboarding-widget__icon" :src="step.icon" alt="" />
        <div class="onboarding-widget__link">
          <span>{{ index + 1 }}.</span>
          <span text type="primary">{{ step.label }}</span>
        </div>
      </div>
    </div>
    <div v-else class="onboarding-widget__custom">
      <strong class="onboarding-widget__custom-title">{{
        variant === 'carousel' ? '轮播图' : '富文本'
      }}</strong>
      <span class="onboarding-widget__custom-description">可在右侧设置面板继续配置展示内容</span>
    </div>
  </section>
</template>

<style scoped lang="scss">
.onboarding-widget {
  position: relative;
  box-sizing: border-box;
  display: flex;
  align-items: flex-end;
  width: 100%;
  height: 100%;
  padding: var(--el-space-xl) var(--el-space-4xl) var(--el-space-2xl);
  background-color: var(--el-bg-color);
  background-position: center bottom;
  background-repeat: no-repeat;
  background-size: 100% 62%;
  border: 0;
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-lighter);

  &__tag {
    position: absolute;
    top: 12px;
    left: 20px;
    padding: var(--el-space-xs) var(--el-space-md);
    color: var(--el-color-white);
    font-size: var(--el-font-size-small);
    background: var(--el-color-primary);
    border-radius: var(--el-border-radius-round);
  }

  &__steps {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    width: 100%;
  }

  &__custom {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: var(--el-space-sm);
    width: 100%;
    min-height: 72px;
    padding: var(--el-space-lg) var(--el-space-xl);
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);
  }

  &__custom-description {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }

  &__step {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    gap: var(--el-space-xs);
    min-width: 0;
  }

  &__icon {
    width: 56px;
    height: 56px;
    object-fit: contain;
  }

  &__link {
    display: flex;
    align-items: center;
    color: var(--el-color-primary);
    font-size: var(--el-font-size-base);

    :deep(.el-button) {
      height: auto;
      padding: 0 var(--el-space-xs);
      font-size: inherit;
      text-decoration: underline;
    }
  }
}

@media (max-width: 768px) {
  .onboarding-widget {
    align-items: center;
    padding: var(--el-space-lg);

    &__icon {
      width: 40px;
      height: 40px;
    }
    &__link {
      font-size: var(--el-font-size-small);
    }
  }
}
</style>
