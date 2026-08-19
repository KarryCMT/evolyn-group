<script setup lang="ts">
// 认证页骨架：顶栏（Logo + 语言切换）+ 左侧品牌区 + 右侧表单卡片，
// 登录/注册/找回密码页共用；卡片内容与底部引导由各页面通过插槽注入
defineProps<{
  /** 卡片标题，如「欢迎来到 evolyn」 */
  title: string
  /** 卡片副标题（可选） */
  subtitle?: string
}>()
</script>

<template>
  <div class="auth-layout">
    <header class="auth-layout__header">
      <div class="auth-layout__logo">
        <span class="auth-layout__logo-mark">E</span>
        <span class="auth-layout__logo-name">evolyn</span>
      </div>
      <LocaleSwitch />
    </header>

    <main class="auth-layout__body">
      <BrandPanel class="auth-layout__brand" />
      <section class="auth-layout__panel">
        <div class="auth-layout__card">
          <h2 class="auth-layout__title">{{ title }}</h2>
          <p v-if="subtitle" class="auth-layout__subtitle">{{ subtitle }}</p>
          <slot />
        </div>
        <div v-if="$slots.footer" class="auth-layout__footer">
          <slot name="footer" />
        </div>
      </section>
    </main>
  </div>
</template>

<style lang="scss" scoped>
.auth-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background-color: var(--el-bg-color-page);
}

.auth-layout__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 32px;
}

.auth-layout__logo {
  display: flex;
  gap: 10px;
  align-items: center;
}

.auth-layout__logo-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  font-size: var(--el-font-size-large);
  font-weight: var(--el-font-weight-bold);
  color: var(--el-color-white);
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-light-3));
  border-radius: var(--el-border-radius-base);
}

.auth-layout__logo-name {
  font-size: var(--el-font-size-extra-large);
  font-weight: var(--el-font-weight-bold);
  color: var(--el-text-color-primary);
}

.auth-layout__body {
  display: grid;
  flex: 1;
  grid-template-columns: 1.1fr 1fr;
  gap: 48px;
  align-items: center;
  width: min(1120px, 100%);
  margin: 0 auto;
  padding: 24px 32px 64px;
}

.auth-layout__panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.auth-layout__card {
  width: min(400px, 100%);
  padding: 36px 36px 32px;
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-light);
  animation: auth-layout-card-in 0.4s ease both;
}

.auth-layout__title {
  margin: 0 0 6px;
  font-size: var(--el-font-size-extra-large);
  font-weight: var(--el-font-weight-bold);
  color: var(--el-text-color-primary);
}

.auth-layout__subtitle {
  margin: 0 0 24px;
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-secondary);
}

.auth-layout__footer {
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-regular);
}

// 小屏收敛为单栏：隐藏品牌区，仅保留表单卡片
@media (max-width: 992px) {
  .auth-layout__body {
    grid-template-columns: 1fr;
  }

  .auth-layout__brand {
    display: none;
  }
}

@keyframes auth-layout-card-in {
  from {
    opacity: 0;
    transform: translateY(12px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
