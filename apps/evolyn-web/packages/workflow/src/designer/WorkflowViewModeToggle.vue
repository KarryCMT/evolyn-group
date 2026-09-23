<script setup lang="ts">
import { ElTooltip } from 'element-plus';

defineOptions({ name: 'WorkflowViewModeToggle' });

defineProps<{ mode: 'compact' | 'detailed' }>();

const emit = defineEmits<{
  updateMode: [mode: 'compact' | 'detailed'];
}>();
</script>

<template>
  <div class="workflow-view-mode" aria-label="节点显示方式">
    <ElTooltip content="简略视图：节点仅显示节点名称" placement="bottom-start">
      <button
        type="button"
        :class="{ 'is-active': mode === 'compact' }"
        aria-label="简略视图"
        @click="emit('updateMode', 'compact')"
      >
        <span class="workflow-view-mode__compact-icon"><i /><i /><i /></span>
      </button>
    </ElTooltip>
    <ElTooltip content="详细视图：节点显示负责人、抄送人等信息" placement="bottom-start">
      <button
        type="button"
        :class="{ 'is-active': mode === 'detailed' }"
        aria-label="详细视图"
        @click="emit('updateMode', 'detailed')"
      >
        <span class="workflow-view-mode__detail-icon"><i /><i /></span>
      </button>
    </ElTooltip>
  </div>
</template>

<style scoped lang="scss">
.workflow-view-mode {
  position: absolute;
  z-index: 3;
  top: 16px;
  left: 16px;
  display: flex;
  padding: 3px;
  background: var(--el-fill-color-light);
  border-radius: 9px;
  box-shadow: var(--el-box-shadow-lighter);

  button {
    display: inline-flex;
    width: 48px;
    height: 38px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-primary);
    background: transparent;
    border: 0;
    border-radius: 7px;
    cursor: pointer;

    &.is-active {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      box-shadow: var(--el-box-shadow-lighter);
    }
  }

  &__compact-icon,
  &__detail-icon {
    display: flex;
    width: 20px;
    flex-direction: column;
    gap: 4px;
  }

  &__compact-icon i {
    width: 20px;
    height: 2px;
    background: currentcolor;
    border-radius: 99px;
  }

  &__detail-icon i {
    width: 20px;
    height: 7px;
    border: 2px solid currentcolor;
    border-radius: 3px;
  }
}
</style>
