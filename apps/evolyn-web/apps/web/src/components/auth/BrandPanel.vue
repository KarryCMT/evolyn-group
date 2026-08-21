<script setup lang="ts">
// 左侧品牌展示区：插画、产品名称与特性点，纯展示组件（无 props/事件）
const features = [
  { title: '表单引擎', desc: '拖拽搭建业务表单' },
  { title: '流程编排', desc: '可视化审批流' },
  { title: '数据报表', desc: '实时统计与分析' },
];
</script>

<template>
  <section class="brand-panel">
    <!-- 低代码平台示意插画：浏览器窗口 + 侧栏 + 图表，配色取 Element Plus 主题变量 -->
    <svg
      class="brand-panel__art"
      viewBox="0 0 360 240"
      role="img"
      aria-label="evolyn 低代码平台示意插画"
    >
      <circle cx="56" cy="48" r="88" fill="var(--el-color-white)" fill-opacity="0.08" />
      <circle cx="314" cy="206" r="40" fill="var(--el-color-white)" fill-opacity="0.1" />

      <rect
        x="64"
        y="44"
        width="232"
        height="152"
        rx="10"
        fill="var(--el-color-white)"
        fill-opacity="0.96"
      />
      <circle cx="82" cy="60" r="4" fill="var(--el-color-danger)" fill-opacity="0.55" />
      <circle cx="94" cy="60" r="4" fill="var(--el-color-warning)" fill-opacity="0.55" />
      <circle cx="106" cy="60" r="4" fill="var(--el-color-success)" fill-opacity="0.55" />

      <rect x="76" y="74" width="52" height="110" rx="6" fill="var(--el-color-primary-light-9)" />
      <rect x="84" y="86" width="36" height="8" rx="4" fill="var(--el-color-primary-light-5)" />
      <rect x="84" y="102" width="36" height="8" rx="4" fill="var(--el-color-primary-light-7)" />
      <rect x="84" y="118" width="36" height="8" rx="4" fill="var(--el-color-primary-light-7)" />
      <rect x="84" y="134" width="24" height="8" rx="4" fill="var(--el-color-primary-light-7)" />

      <rect x="140" y="84" width="72" height="10" rx="5" fill="var(--el-color-primary-light-5)" />
      <rect x="228" y="84" width="44" height="10" rx="5" fill="var(--el-color-primary-light-7)" />
      <rect x="140" y="120" width="20" height="64" rx="4" fill="var(--el-color-primary-light-7)" />
      <rect x="168" y="100" width="20" height="84" rx="4" fill="var(--el-color-primary-light-5)" />
      <rect x="196" y="132" width="20" height="52" rx="4" fill="var(--el-color-primary)" />
      <rect x="228" y="120" width="48" height="8" rx="4" fill="var(--el-color-primary-light-7)" />
      <rect x="228" y="136" width="48" height="8" rx="4" fill="var(--el-color-primary-light-7)" />
      <rect x="228" y="152" width="34" height="8" rx="4" fill="var(--el-color-primary-light-7)" />

      <rect
        x="252"
        y="166"
        width="80"
        height="44"
        rx="8"
        fill="var(--el-color-white)"
        fill-opacity="0.95"
      />
      <circle cx="270" cy="188" r="11" fill="var(--el-color-success-light-7)" />
      <path
        d="m265 188 3.5 3.5 7-7"
        fill="none"
        stroke="var(--el-color-success)"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <rect x="288" y="180" width="34" height="6" rx="3" fill="var(--el-color-primary-light-5)" />
      <rect x="288" y="192" width="24" height="6" rx="3" fill="var(--el-color-primary-light-7)" />
    </svg>

    <div class="brand-panel__intro">
      <h1 class="brand-panel__title">企业级低代码平台</h1>
      <p class="brand-panel__slogan">像搭积木一样，快速构建属于你的业务系统</p>
    </div>

    <ul class="brand-panel__features">
      <li v-for="feature in features" :key="feature.title" class="brand-panel__feature">
        <span class="brand-panel__feature-dot" />
        <div>
          <p class="brand-panel__feature-title">{{ feature.title }}</p>
          <p class="brand-panel__feature-desc">{{ feature.desc }}</p>
        </div>
      </li>
    </ul>
  </section>
</template>

<style lang="scss" scoped>
.brand-panel {
  display: flex;
  flex-direction: column;
  gap: 32px;
  padding: 48px 56px;
  color: var(--el-color-white);
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-light-3));
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow);
}

.brand-panel__art {
  width: 100%;
  height: auto;
}

.brand-panel__title {
  margin: 0 0 8px;
  font-size: 26px;
  font-weight: var(--el-font-weight-bold);
}

.brand-panel__slogan {
  margin: 0;
  font-size: var(--el-font-size-medium);
  color: var(--el-color-white);
  opacity: 0.85;
}

.brand-panel__features {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 0;
  margin: 0;
  list-style: none;
}

.brand-panel__feature {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.brand-panel__feature-dot {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  background-color: var(--el-color-white);
  border-radius: var(--el-border-radius-round);
  opacity: 0.9;
}

.brand-panel__feature-title {
  margin: 0 0 2px;
  font-size: var(--el-font-size-medium);
  font-weight: 500;
}

.brand-panel__feature-desc {
  margin: 0;
  font-size: var(--el-font-size-base);
  opacity: 0.75;
}
</style>
