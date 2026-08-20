<script setup lang="ts">
import { ElAvatar } from 'element-plus';
import { Bell, Grid, UserFilled } from '@element-plus/icons-vue';

defineOptions({ name: 'TopNavigation' });
</script>

<template>
  <header class="top-navigation">
    <div class="top-navigation__brand">
      <el-button class="top-navigation__switcher" circle :icon="Grid" aria-label="切换产品" />
      <span class="top-navigation__logo" aria-hidden="true">◆</span>
      <strong>工作台</strong>
    </div>

    <nav class="top-navigation__actions" aria-label="工作台导航">
      <el-button text>模板中心</el-button>
      <el-button text>通讯录</el-button>
      <el-button text circle :icon="Bell" aria-label="通知" />
      <el-dropdown>
        <el-avatar :size="28" :icon="UserFilled" />
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item>个人中心</el-dropdown-item>
            <el-dropdown-item>退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </nav>
  </header>
</template>

<style scoped lang="scss">
.top-navigation {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);

  &__brand,
  &__actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__switcher {
    color: var(--el-text-color-primary);
  }

  &__logo {
    color: var(--el-color-primary);
    font-size: var(--el-font-size-large);
  }
}
</style>
