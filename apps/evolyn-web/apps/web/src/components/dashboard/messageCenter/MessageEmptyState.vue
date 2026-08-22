<script setup lang="ts">
import { RiMailUnreadFill } from '@remixicon/vue';

defineOptions({ name: 'MessageEmptyState' });
</script>

<template>
  <div class="message-empty-state">
    <span class="message-empty-state__illustration" aria-hidden="true">
      <el-icon><RiMailUnreadFill /></el-icon>
      <i class="message-empty-state__spark message-empty-state__spark--left" />
      <i class="message-empty-state__spark message-empty-state__spark--right" />
    </span>
    <p class="message-empty-state__text">暂无消息</p>
  </div>
</template>

<style scoped lang="scss">
.message-empty-state {
  display: flex;
  min-height: 360px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 22px;

  &__illustration {
    position: relative;
    display: inline-flex;
    width: 116px;
    height: 96px;
    align-items: center;
    justify-content: center;
    background: linear-gradient(
      145deg,
      var(--el-color-primary-light-9),
      var(--el-color-primary-light-7)
    );
    border-radius: 18px 18px 32px 32px;
    box-shadow: inset 0 0 0 10px rgb(255 255 255 / 38%);

    .el-icon {
      color: var(--el-color-primary);
      font-size: 58px;
    }
  }

  &__spark {
    position: absolute;
    display: block;
    width: 16px;
    height: 16px;
    background: var(--el-color-primary-light-7);
    border-radius: 50%;

    &--left {
      left: -25px;
      bottom: -8px;
    }

    &--right {
      right: -25px;
      bottom: -8px;
    }
  }

  &__text {
    margin: 0;
    color: #8f98a7;
    font-size: 17px;
    line-height: 24px;
  }
}
</style>
