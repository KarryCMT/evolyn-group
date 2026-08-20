<script setup lang="ts">
import { ElButtonGroup } from 'element-plus';
import { Cellphone, Monitor, View } from '@element-plus/icons-vue';

defineOptions({ name: 'WorkbenchEditorToolbar' });

const props = defineProps<{ device: 'desktop' | 'mobile' }>();
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
      <el-button-group>
        <el-button :type="device === 'desktop' ? 'primary' : 'default'" :icon="Monitor" @click="emit('update:device', 'desktop')" />
        <el-button :type="device === 'mobile' ? 'primary' : 'default'" :icon="Cellphone" @click="emit('update:device', 'mobile')" />
      </el-button-group>
      <el-button @click="emit('pageStyle')">页面样式</el-button>
      <el-button :icon="View" @click="emit('preview')">预览</el-button>
      <el-button type="primary" @click="emit('save')">保存</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.workbench-editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: 0 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);

  &__actions { display: flex; align-items: center; gap: 8px; }
}
</style>
