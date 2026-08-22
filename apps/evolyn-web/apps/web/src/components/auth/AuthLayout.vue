<script setup lang="ts">
// 认证页骨架：顶栏（Logo + 语言切换）+ 左侧品牌区 + 右侧表单卡片，
// 登录/注册/找回密码页共用；卡片内容与底部引导由各页面通过插槽注入
import loginBackground from '~/assets/images/login_bg.jpg';
import brandLogo from '~/assets/images/logo.png';

defineProps<{
  /** 卡片标题，如「欢迎来到 evolyn」 */
  title: string;
  /** 卡片副标题（可选） */
  subtitle?: string;
  /** 登录页使用无卡片的全屏双栏版式，其他认证页仍沿用默认骨架 */
  variant?: 'default' | 'login';
}>();
</script>

<template>
  <div class="auth-layout" :class="{ 'auth-layout--login': variant === 'login' }">
    <header class="auth-layout__header">
      <img
        v-if="variant === 'login'"
        class="auth-layout__logo-image"
        :src="brandLogo"
        alt="简道云"
      />
      <div v-else class="auth-layout__logo">
        <span class="auth-layout__logo-mark">E</span>
        <span class="auth-layout__logo-name">evolyn</span>
      </div>
      <LocaleSwitch />
    </header>

    <main class="auth-layout__body">
      <!-- 登录页以现有背景插画承载品牌视觉，避免再叠加一张有边界的展示卡片。 -->
      <aside
        v-if="variant === 'login'"
        class="auth-layout__visual"
        :style="{ backgroundImage: `url(${loginBackground})` }"
        aria-hidden="true"
      />
      <BrandPanel v-else class="auth-layout__brand" />
      <section class="auth-layout__panel">
        <div class="auth-layout__card">
          <!-- 注册向导等场景可把进度放在标题前，避免影响普通认证页的标题层级。 -->
          <slot name="before-title" />
          <h2 class="auth-layout__title">{{ title }}</h2>
          <p v-if="subtitle" class="auth-layout__subtitle">{{ subtitle }}</p>
          <slot name="after-title" />
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

.auth-layout__logo-image {
  display: block;
  width: 142px;
  height: auto;
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

// 简道云式登录：视觉区与表单区直接拼接，表单不再包裹在带阴影的卡片中。
.auth-layout--login {
  // 认证流程沿用项目品牌蓝，与业务界面和弹层保持一致。
  --el-color-primary: #1677ff;
  --el-color-primary-light-3: #5ca0ff;
  --el-color-primary-light-5: #8bbbff;
  --el-color-primary-light-7: #b9d6ff;
  --el-color-primary-light-8: #d0e4ff;
  --el-color-primary-light-9: #e8f1ff;
  --el-color-primary-dark-2: #125fcc;

  overflow: hidden;
  background-color: var(--el-bg-color);

  .auth-layout__header {
    position: fixed;
    z-index: 1;
    top: 0;
    right: 0;
    left: 0;
    padding: 22px 24px;
    pointer-events: none;
  }

  .auth-layout__header :deep(.locale-switch),
  .auth-layout__logo-image {
    pointer-events: auto;
  }

  .auth-layout__body {
    display: grid;
    grid-template-columns: minmax(0, 58%) minmax(420px, 42%);
    gap: 0;
    width: 100%;
    min-height: 100vh;
    padding: 0;
  }

  .auth-layout__visual {
    min-height: 100%;
    background-color: #f7fcfc;
    background-repeat: no-repeat;
    background-position: 55% 56%;
    background-size: cover;
  }

  .auth-layout__panel {
    justify-content: center;
    min-height: 100%;
    padding: 120px 32px 56px;
    background-color: var(--el-bg-color);
  }

  .auth-layout__card {
    width: min(352px, 100%);
    padding: 0;
    background: transparent;
    border: 0;
    border-radius: 0;
    box-shadow: none;
  }

  .auth-layout__title {
    margin-bottom: 28px;
    font-size: 28px;
    text-align: center;
  }

  .auth-layout__footer {
    margin-top: 6px;
  }
}

// 小屏收敛为单栏：隐藏品牌区，仅保留表单卡片
@media (max-width: 992px) {
  .auth-layout__body {
    grid-template-columns: 1fr;
  }

  .auth-layout__brand {
    display: none;
  }

  .auth-layout--login {
    overflow: auto;

    .auth-layout__body {
      display: block;
      min-height: 100vh;
      padding-top: 1px;
    }

    .auth-layout__visual {
      display: none;
    }

    .auth-layout__panel {
      justify-content: flex-start;
      min-height: 100vh;
      padding-top: 108px;
    }
  }
}

@media (max-width: 576px) {
  .auth-layout--login {
    .auth-layout__header {
      padding: 16px;
    }

    .auth-layout__logo-image {
      width: 120px;
    }

    .auth-layout__panel {
      align-items: stretch;
      padding: 104px 24px 36px;
    }

    .auth-layout__card {
      width: 100%;
    }
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
