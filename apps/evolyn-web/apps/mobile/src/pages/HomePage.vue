<script setup lang="ts">
import {
  Cell as VanCell,
  CellGroup as VanCellGroup,
  Icon as VanIcon,
  Tag as VanTag,
} from 'vant';
</script>

<template>
  <section class="home-page">
    <header class="home-page__hero">
      <VanTag class="home-page__eyebrow" plain round type="primary">
        MOBILE WORKSPACE
      </VanTag>
      <h1>随时处理你的业务</h1>
      <p>移动宿主已接入共享表单协议、校验与提交状态机。</p>
    </header>

    <VanCellGroup class="home-page__group" inset title="开发验收">
      <VanCell
        title="移动表单运行时"
        label="验证 Vant 字段、规则校验与安全区操作栏"
        to="/runtime-preview"
        is-link
      >
        <template #icon>
          <span class="home-page__cell-icon"><VanIcon name="records-o" /></span>
        </template>
      </VanCell>
    </VanCellGroup>

    <VanCellGroup class="home-page__group" inset title="基础能力">
      <VanCell title="共享 Runtime Core" label="桌面与移动端使用同一 Schema 和值协议">
        <template #value>
          <VanTag type="success">
            已接入
          </VanTag>
        </template>
      </VanCell>
      <VanCell title="Vant 移动组件层" label="移动输入、选择器与反馈交互">
        <template #value>
          <VanTag type="success">
            已接入
          </VanTag>
        </template>
      </VanCell>
      <VanCell title="原生应用容器" label="需要上架时再接入 Capacitor">
        <template #value>
          <VanTag plain>
            待启用
          </VanTag>
        </template>
      </VanCell>
    </VanCellGroup>
  </section>
</template>

<style scoped>
.home-page {
  padding: var(--van-padding-md) 0 var(--van-padding-xl);
}

.home-page__hero {
  padding: 28px var(--van-padding-lg) 24px;
  margin: 0 var(--van-padding-md) var(--van-padding-md);
  overflow: hidden;
  color: #fff;
  background:
    radial-gradient(circle at 90% 10%, rgb(255 255 255 / 24%), transparent 34%),
    linear-gradient(135deg, #2563eb, #4f46e5);
  border-radius: 20px;
  box-shadow: 0 12px 30px rgb(37 99 235 / 20%);
}

.home-page__eyebrow {
  color: #fff;
  border-color: rgb(255 255 255 / 55%);
}

.home-page__hero h1 {
  margin: 18px 0 8px;
  font-size: 26px;
  line-height: 1.2;
}

.home-page__hero p {
  margin: 0;
  font-size: var(--van-font-size-md);
  line-height: 1.7;
  color: rgb(255 255 255 / 82%);
}

.home-page__group + .home-page__group {
  margin-top: var(--van-padding-md);
}

.home-page__cell-icon {
  display: inline-grid;
  width: 36px;
  height: 36px;
  margin-right: var(--van-padding-sm);
  font-size: 20px;
  color: var(--van-primary-color);
  background: color-mix(in srgb, var(--van-primary-color) 10%, transparent);
  border-radius: var(--van-radius-lg);
  place-items: center;
}
</style>
