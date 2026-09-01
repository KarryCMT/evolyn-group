<script setup lang="ts">
import { ElButtonGroup } from 'element-plus';
import { RiComputerFill, RiEyeFill, RiSmartphoneFill } from '@remixicon/vue';

defineOptions({ name: 'WorkbenchEditorToolbar' });

const props = withDefaults(
  defineProps<{
    device: 'desktop' | 'mobile';
    isDirty?: boolean;
    isSaving?: boolean;
  }>(),
  { isDirty: false, isSaving: false },
);
const emit = defineEmits<{
  'update:device': [value: 'desktop' | 'mobile'];
  pageStyle: [];
  preview: [];
  save: [];
}>();
</script>

<template>
  <div class="workbench-editor-toolbar">
    <el-button text type="primary">如何自定义工作台？</el-button>
    <div class="workbench-editor-toolbar__actions">
      <span v-if="isDirty" class="workbench-editor-toolbar__status">未保存</span>
      <el-button-group>
        <el-button
          :type="device === 'desktop' ? 'primary' : 'default'"
          :icon="RiComputerFill"
          @click="emit('update:device', 'desktop')"
        />
        <el-button
          :type="device === 'mobile' ? 'primary' : 'default'"
          :icon="RiSmartphoneFill"
          @click="emit('update:device', 'mobile')"
        />
      </el-button-group>
      <el-button @click="emit('pageStyle')">页面样式</el-button>
      <el-button :icon="RiEyeFill" @click="emit('preview')">预览</el-button>
      <el-button type="primary" :loading="isSaving" @click="emit('save')">
        {{ isSaving ? '保存中' : '保存' }}
      </el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.workbench-editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: 0 var(--el-space-xl);
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);

  &__actions {
    display: flex;
    align-items: center;
    gap: var(--el-space-md);
  }

  &__status {
    color: var(--el-color-warning);
    font-size: var(--el-font-size-small);
  }
}
</style>
