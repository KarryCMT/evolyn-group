<script setup lang="ts">
import * as EvolynUi from '@evolyn.do/ui';
import type { EvolynIconPickerValue } from '@evolyn.do/ui';
import { computed, shallowRef } from 'vue';
import { uploadFile } from '~/api/file';

defineOptions({ name: 'ApplicationIconPicker' });

const iconValue = defineModel<EvolynIconPickerValue>({ required: true });
const pickerVisible = shallowRef(false);
const EvolynIconPicker = EvolynUi.EvolynIconPicker;

const selectedSystemIcon = computed(() => {
  if (iconValue.value.type !== 'remix') return undefined;
  return EvolynUi.defaultSystemIcons?.find((option) => option.name === iconValue.value.name)?.icon;
});
const previewStyle = computed(() => {
  if (iconValue.value.type !== 'remix') return undefined;
  return { backgroundImage: `linear-gradient(135deg, ${iconValue.value.background})` };
});

function updateIcon(value: EvolynIconPickerValue | undefined) {
  if (!value) return;
  iconValue.value = value;
  if (value.type === 'remix') pickerVisible.value = false;
}

async function uploadIcon(file: File) {
  // 应用层统一负责上传，基础组件只处理裁剪与文件选择。
  iconValue.value = { type: 'custom', name: await uploadFile(file) };
  pickerVisible.value = false;
}
</script>

<template>
  <el-popover
    v-model:visible="pickerVisible"
    popper-class="application-icon-picker__popper"
    placement="bottom-start"
    :show-arrow="false"
    :width="320"
    trigger="click"
    :teleported="true"
  >
    <template #reference>
      <button
        class="application-icon-picker"
        type="button"
        aria-label="修改图标"
        :style="previewStyle"
      >
        <img v-if="iconValue.type === 'custom'" :src="iconValue.name" alt="自定义应用图标" />
        <component v-else-if="selectedSystemIcon" :is="selectedSystemIcon" />
        <span>修改图标</span>
      </button>
    </template>
    <EvolynIconPicker
      :model-value="iconValue"
      @update:model-value="updateIcon"
      @upload="uploadIcon"
    />
  </el-popover>
</template>

<!-- 浮层传送至 body，因此用组件唯一类名限制样式范围。 -->
<style lang="scss">
.application-icon-picker {
  position: relative;
  display: inline-flex;
  width: 56px;
  height: 56px;
  padding: 0;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 0;
  border-radius: var(--el-border-radius-large);
  color: var(--el-color-white);
  cursor: pointer;
  background: linear-gradient(135deg, #f7be54, #eda426);
  font-size: 28px;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  span {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0;
    color: var(--el-color-white);
    background: var(--el-overlay-color-lighter);
    font-size: var(--el-font-size-extra-small);
    transition: opacity 0.18s ease;
  }

  &:hover span,
  &:focus-visible span {
    opacity: 1;
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.application-icon-picker__popper.el-popper {
  padding: 0;
  border: 0;
  background: transparent;
  box-shadow: var(--el-box-shadow-light);
}
</style>
