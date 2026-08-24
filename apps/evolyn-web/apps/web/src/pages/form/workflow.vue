<script setup lang="ts">
import { RiEyeFill, RiFullscreenFill, RiPlayFill, RiSave3Fill } from '@remixicon/vue';
import {
  WorkflowDesigner,
  createDefaultWorkflowDocument,
  type WorkflowDocument,
  type WorkflowField,
} from '@evolyn.do/workflow';
import { ElMessage } from 'element-plus';
import { shallowRef } from 'vue';

defineOptions({ name: 'FormWorkflowPage' });

/** 应用层只向工作流内核注入字段契约，不让两个领域包互相依赖。 */
const workflowFields: WorkflowField[] = [
  { id: 'project-code', label: '项目编号', required: true },
  { id: 'project-name', label: '项目名称', required: true },
  { id: 'project-manager', label: '项目负责人' },
  { id: 'start-date', label: '实际开始时间' },
  { id: 'completion-date', label: '实际完成时间' },
  { id: 'description', label: '完成情况描述' },
];
const workflowDocument = shallowRef<WorkflowDocument>(
  createDefaultWorkflowDocument(workflowFields),
);

function updateDocument(document: WorkflowDocument) {
  workflowDocument.value = document;
}

function notifyUnavailable(action: string) {
  ElMessage.info(`${action}将在流程服务接入后提供`);
}
</script>

<template>
  <section class="form-workflow-page" aria-label="流程设计工作台">
    <div class="form-workflow-page__toolbar" aria-label="流程设计操作">
      <div class="form-workflow-page__toolbar-actions">
        <span class="form-workflow-page__version"><i />流程版本（V1）</span>
        <button
          class="form-workflow-page__action-button"
          type="button"
          @click="notifyUnavailable('预览')"
        >
          <RiEyeFill />
          <span>预览</span>
        </button>
        <button
          class="form-workflow-page__action-button"
          type="button"
          @click="notifyUnavailable('测试')"
        >
          <RiPlayFill />
          <span>测试</span>
        </button>
        <button
          class="form-workflow-page__action-button form-workflow-page__action-button--primary"
          type="button"
          @click="notifyUnavailable('保存')"
        >
          <RiSave3Fill />
          <span>保存</span>
        </button>
        <button
          class="form-workflow-page__icon-button"
          type="button"
          aria-label="全屏显示"
          @click="notifyUnavailable('全屏显示')"
        >
          <RiFullscreenFill />
        </button>
      </div>
    </div>

    <WorkflowDesigner
      class="form-workflow-page__workspace"
      :document="workflowDocument"
      :fields="workflowFields"
      @update-document="updateDocument"
    />
  </section>
</template>

<style scoped lang="scss">
.form-workflow-page {
  display: flex;
  min-height: 0;
  margin: 0 8px 8px;
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  box-shadow: var(--el-box-shadow-light);

  &__toolbar,
  &__toolbar-actions,
  &__action-button,
  &__version {
    display: flex;
    align-items: center;
  }

  &__toolbar {
    height: 50px;
    min-height: 50px;
    padding: 0 16px;
    justify-content: flex-end;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__toolbar-actions {
    gap: 8px;
  }

  &__icon-button,
  &__action-button {
    border: 0;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__icon-button {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__version {
    margin-right: 8px;
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    gap: 7px;

    i {
      display: block;
      width: 8px;
      height: 8px;
      background: var(--el-color-success);
      border-radius: 50%;
    }
  }

  &__action-button {
    height: 32px;
    padding: 0 10px;
    justify-content: center;
    gap: 6px;
    color: var(--el-color-primary);
    font-size: 14px;
    font-weight: 600;
    background: var(--el-bg-color);
    border: 1px solid var(--el-color-primary);
    border-radius: var(--el-border-radius-base);

    svg {
      width: 16px;
      height: 16px;
    }

    &:hover {
      background: var(--el-color-primary-light-9);
    }

    &--primary {
      color: var(--el-color-white);
      background: var(--el-color-primary);

      &:hover {
        background: var(--el-color-primary-light-3);
      }
    }
  }

  &__icon-button {
    border: 1px solid var(--el-border-color);

    &:hover {
      border-color: var(--el-color-primary);
    }
  }

  &__workspace {
    min-height: 0;
    flex: 1;
  }
}

@media (max-width: 760px) {
  .form-workflow-page {
    margin: 0 4px 4px;
    border-radius: 10px;

    &__toolbar {
      padding: 0 10px;
    }

    &__version,
    &__action-button span {
      display: none;
    }

    &__action-button {
      width: 32px;
      padding: 0;
    }
  }
}
</style>
