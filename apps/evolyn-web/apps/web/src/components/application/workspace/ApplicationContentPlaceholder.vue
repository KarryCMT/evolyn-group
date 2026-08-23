<script setup lang="ts">
import { RiFileList3Fill, RiLayoutGridFill } from '@remixicon/vue';
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceMode,
} from './applicationWorkspace.types';

defineOptions({ name: 'ApplicationContentPlaceholder' });

const props = defineProps<{
  asset: ApplicationWorkspaceAsset;
  mode: ApplicationWorkspaceMode;
}>();

const modeLabels: Record<ApplicationWorkspaceMode, string> = {
  fill: '数据填写区',
  design: '设计区',
  data: '数据管理区',
};
</script>

<template>
  <main
    class="application-content-placeholder"
    :aria-label="`${props.asset.label}${modeLabels[props.mode]}`"
  >
    <div class="application-content-placeholder__content">
      <component
        :is="props.asset.type === 'dashboard' ? RiLayoutGridFill : RiFileList3Fill"
        class="application-content-placeholder__icon"
        aria-hidden="true"
      />
      <h1 class="application-content-placeholder__title">{{ props.asset.label }}</h1>
      <p class="application-content-placeholder__description">
        {{ modeLabels[props.mode] }}已预留；表单渲染与设计能力将由后续独立包接入。
      </p>
    </div>
  </main>
</template>

<style scoped lang="scss">
.application-content-placeholder {
  display: grid;
  min-width: 0;
  min-height: 0;
  flex: 1;
  place-items: center;
  background: var(--el-bg-color);

  &__content {
    display: flex;
    max-width: 360px;
    padding: 32px;
    align-items: center;
    flex-direction: column;
    color: var(--el-text-color-secondary);
    text-align: center;
  }

  &__icon {
    margin-bottom: 16px;
    color: var(--el-color-primary-light-5);
    font-size: 56px;
  }

  &__title {
    margin: 0 0 8px;
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 650;
    line-height: 28px;
  }

  &__description {
    margin: 0;
    font-size: 14px;
    line-height: 22px;
  }
}
</style>
