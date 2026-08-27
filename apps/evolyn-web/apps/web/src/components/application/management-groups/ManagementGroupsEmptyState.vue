<script setup lang="ts">
import { RiAddFill } from '@remixicon/vue';

defineOptions({ name: 'ManagementGroupsEmptyState' });

const emit = defineEmits<{
  create: [];
}>();
</script>

<template>
  <section class="management-groups-empty-state" aria-label="暂无应用管理组">
    <div class="management-groups-empty-state__illustration" aria-hidden="true">
      <span class="management-groups-empty-state__beam management-groups-empty-state__beam--left" />
      <span
        class="management-groups-empty-state__beam management-groups-empty-state__beam--right"
      />
      <span class="management-groups-empty-state__moon" />
      <span class="management-groups-empty-state__tower">
        <i class="management-groups-empty-state__tower-light" />
        <i class="management-groups-empty-state__tower-door" />
      </span>
      <span class="management-groups-empty-state__tree management-groups-empty-state__tree--left" />
      <span
        class="management-groups-empty-state__tree management-groups-empty-state__tree--right"
      />
      <span class="management-groups-empty-state__island" />
    </div>
    <p class="management-groups-empty-state__description">
      应用管理组可以为成员配置不同表单或仪表盘的设计权限
    </p>
    <button class="management-groups-empty-state__create" type="button" @click="emit('create')">
      <RiAddFill aria-hidden="true" />
      添加管理组
    </button>
  </section>
</template>

<style scoped lang="scss">
.management-groups-empty-state {
  display: flex;
  min-height: 0;
  padding-bottom: 9%;
  align-items: center;
  justify-content: center;
  flex: 1;
  flex-direction: column;

  &__illustration {
    position: relative;
    width: 320px;
    height: 220px;
  }

  &__island {
    position: absolute;
    z-index: 1;
    right: 22px;
    bottom: 20px;
    left: 22px;
    height: 42px;
    border-radius: var(--el-border-radius-half) var(--el-border-radius-half) 15% 15%;
    background: linear-gradient(180deg, var(--el-fill-color-light), var(--el-fill-color));
  }

  &__beam {
    position: absolute;
    top: 38px;
    width: 130px;
    height: 28px;
    opacity: 0.62;
    background: linear-gradient(90deg, transparent, #fff1bc);

    &--left {
      left: 41px;
      clip-path: polygon(0 32%, 100% 0, 100% 100%, 0 68%);
    }

    &--right {
      right: 46px;
      transform: scaleX(-1);
      clip-path: polygon(0 32%, 100% 0, 100% 100%, 0 68%);
    }
  }

  &__moon {
    position: absolute;
    top: 16px;
    left: 174px;
    width: 30px;
    height: 30px;
    border-radius: var(--el-border-radius-half);
    background: var(--el-color-primary-light-8);
  }

  &__tower {
    position: absolute;
    z-index: 2;
    bottom: 39px;
    left: 131px;
    width: 58px;
    height: 126px;
    border-radius: var(--el-border-radius-medium)
      var(--el-border-radius-medium) var(--el-border-radius-small)
      var(--el-border-radius-small);
    background:
      linear-gradient(90deg, transparent 46%, rgb(110 132 161 / 13%) 48%, transparent 50%),
      linear-gradient(180deg, #e7eff8, #cbd9e9);
    clip-path: polygon(19% 0, 81% 0, 88% 100%, 12% 100%);

    &::before {
      position: absolute;
      top: -16px;
      left: -9px;
      width: 76px;
      height: 18px;
      border: 5px solid var(--el-border-color-darker);
      border-bottom-width: 7px;
      content: '';
    }

    &::after {
      position: absolute;
      top: -30px;
      left: 24px;
      width: 10px;
      height: 16px;
      border-radius: var(--el-border-radius-base)
        var(--el-border-radius-base) 0 0;
      background: #9cb2cb;
      content: '';
    }
  }

  &__tower-light {
    position: absolute;
    top: -10px;
    left: 22px;
    width: 14px;
    height: 10px;
    border-radius: var(--el-border-radius-small);
    background: #fff6cf;
  }

  &__tower-door {
    position: absolute;
    bottom: 0;
    left: 21px;
    width: 17px;
    height: 25px;
    border-radius: var(--el-border-radius-large)
      var(--el-border-radius-large) 0 0;
    background: #9eb5cf;
  }

  &__tree {
    position: absolute;
    z-index: 3;
    bottom: 43px;
    width: 26px;
    height: 37px;
    border-radius: var(--el-border-radius-half) var(--el-border-radius-half) 43% 43%;
    background: #c7d5e5;

    &::after {
      position: absolute;
      bottom: -12px;
      left: 11px;
      width: 4px;
      height: 15px;
      background: #b2c3d4;
      content: '';
    }

    &--left {
      left: 83px;
    }

    &--right {
      right: 78px;
      width: 22px;
      height: 31px;
    }
  }

  &__description {
    margin: var(--el-space-xs) 0 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__create {
    display: inline-flex;
    min-height: 40px;
    margin-top: var(--el-space-xl);
    padding: 0 var(--el-space-xl);
    border: 0;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    color: var(--el-color-white);
    cursor: pointer;
    background: var(--el-color-primary);
    font: inherit;
    font-size: var(--el-font-size-base);
    gap: var(--el-space-xs);

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      background: var(--el-color-primary-light-3);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }
}

@media (max-width: 760px) {
  .management-groups-empty-state {
    &__illustration {
      transform: scale(0.84);
    }

    &__description {
      padding: 0 var(--el-space-3xl);
      text-align: center;
    }
  }
}
</style>
